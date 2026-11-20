package codegen

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"reflect"

	"golang.org/x/tools/go/packages"
)

// key identifies a value in the graph. typ is the fully qualified type string
// so keys from different package loads compare equal.
type key struct {
	typ   string
	name  string
	group bool
}

func (k key) String() string {
	switch {
	case k.group:
		return fmt.Sprintf("%s (group %q)", k.typ, k.name)
	case k.name != "":
		return fmt.Sprintf("%s (name %q)", k.typ, k.name)
	}
	return k.typ
}

func typeKey(t types.Type) key { return key{typ: types.TypeString(types.Unalias(t), nil)} }

type scope struct {
	name   string
	parent *scope
	decs   map[key][]*fn
}

type fnKind int

const (
	kProvider fnKind = iota
	kInvoke
	kDecorator
)

// fn is a function passed to Provide, Lifecycled, Invoke or Decorate.
type fn struct {
	kind        fnKind
	obj         *types.Func
	params      []param
	result      types.Type // first result; nil for invokes
	outs        []field    // values provided; a single unnamed one unless the result is an Out struct
	hasErr      bool
	start, stop bool // Lifecycled: register these methods as a hook
	scope       *scope
	pos         string
}

func (f *fn) String() string {
	return fmt.Sprintf("%s.%s (%s)", f.obj.Pkg().Name(), f.obj.Name(), f.pos)
}

type param struct {
	app    bool       // solder.Lifecycle
	key    key        // plain parameter
	inType types.Type // In struct; fields hold its dependencies
	fields []field
}

// field is a struct field of an In or Out struct, or (name "") a plain value.
type field struct {
	name string
	key  key
	typ  types.Type
}

type prodKind int

const (
	pInput prodKind = iota
	pSupply
	pCall
	pBind
)

type producer struct {
	kind   prodKind
	pos    string
	input  *variable         // pInput
	expr   ast.Expr          // pSupply
	pkg    *packages.Package // pSupply
	typ    types.Type
	fn     *fn    // pCall
	field  string // pCall: Out struct field, or ""
	target key    // pBind
}

type graph struct {
	root      *scope
	producers map[key]*producer
	groups    map[key][]*producer
	groupElem map[key]types.Type
	roots     []*fn // Lifecycled providers and invokes, in declaration order
}

type parser struct {
	ld   *loader
	cur  *types.Package
	g    *graph
	seen map[string]bool
}

func (p *parser) errorf(pkg *packages.Package, pos token.Pos, format string, args ...any) error {
	return fmt.Errorf("%s: %s", p.ld.pos(pkg.Fset, pos), fmt.Sprintf(format, args...))
}

func (p *parser) add(k key, pr *producer) error {
	if k.group {
		p.g.groups[k] = append(p.g.groups[k], pr)
		p.g.groupElem[k] = pr.typ
		return nil
	}
	if old, ok := p.g.producers[k]; ok {
		return fmt.Errorf("%s: %s is provided twice\n\talso provided at %s", pr.pos, k, old.pos)
	}
	p.g.producers[k] = pr
	return nil
}

func (p *parser) options(pkg *packages.Package, args []ast.Expr, sc *scope) error {
	for _, a := range args {
		if err := p.option(pkg, a, sc); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) option(pkg *packages.Package, e ast.Expr, sc *scope) error {
	e = ast.Unparen(e)
	info := pkg.TypesInfo
	if call, ok := e.(*ast.CallExpr); ok {
		name, id := markerName(info, call)
		switch name {
		case "Provide", "Lifecycled":
			for _, a := range call.Args {
				f, err := p.fn(pkg, a, kProvider, sc)
				if err != nil {
					return err
				}
				if name == "Lifecycled" {
					if err := p.lifecycled(pkg, a, f); err != nil {
						return err
					}
					p.g.roots = append(p.g.roots, f)
				}
				for _, o := range f.outs {
					if err := p.add(o.key, &producer{kind: pCall, pos: f.pos, fn: f, field: o.name, typ: o.typ}); err != nil {
						return err
					}
				}
			}
			return nil
		case "Invoke":
			for _, a := range call.Args {
				f, err := p.fn(pkg, a, kInvoke, sc)
				if err != nil {
					return err
				}
				p.g.roots = append(p.g.roots, f)
			}
			return nil
		case "Decorate":
			for _, a := range call.Args {
				f, err := p.fn(pkg, a, kDecorator, sc)
				if err != nil {
					return err
				}
				if sc.decs == nil {
					sc.decs = map[key][]*fn{}
				}
				k := f.outs[0].key
				sc.decs[k] = append(sc.decs[k], f)
			}
			return nil
		case "Supply":
			if pkg.Types.Path() != p.cur.Path() {
				return p.errorf(pkg, call.Pos(), "solder.Supply outside the injector's package is not supported yet; use a constructor")
			}
			for _, a := range call.Args {
				tv := info.Types[a]
				if tv.IsNil() || untypedConst(info, a) {
					return p.errorf(pkg, a.Pos(), "solder.Supply(%s): untyped constant has no type to provide; convert it, e.g. Supply(MyType(%s))", types.ExprString(a), types.ExprString(a))
				}
				pr := &producer{kind: pSupply, pos: p.ld.pos(pkg.Fset, a.Pos()), expr: a, pkg: pkg, typ: tv.Type}
				if err := p.add(typeKey(tv.Type), pr); err != nil {
					return err
				}
			}
			return nil
		case "Bind":
			inst := info.Instances[id]
			iface, impl := inst.TypeArgs.At(0), inst.TypeArgs.At(1)
			it, ok := iface.Underlying().(*types.Interface)
			if !ok {
				return p.errorf(pkg, call.Pos(), "solder.Bind: %s is not an interface", iface)
			}
			if !types.Implements(impl, it) {
				hint := ""
				if types.Implements(types.NewPointer(impl), it) {
					hint = fmt.Sprintf(" (but *%s does)", impl)
				}
				return p.errorf(pkg, call.Pos(), "solder.Bind: %s does not implement %s%s", impl, iface, hint)
			}
			return p.add(typeKey(iface), &producer{kind: pBind, pos: p.ld.pos(pkg.Fset, call.Pos()), target: typeKey(impl), typ: iface})
		case "Module":
			if len(call.Args) == 0 {
				return p.errorf(pkg, call.Pos(), "solder.Module needs a name")
			}
			tv := info.Types[call.Args[0]]
			if tv.Value == nil || tv.Value.Kind() != constant.String {
				return p.errorf(pkg, call.Args[0].Pos(), "module name must be a string constant")
			}
			child := &scope{name: constant.StringVal(tv.Value), parent: sc}
			return p.options(pkg, call.Args[1:], child)
		case "Build":
			return p.errorf(pkg, call.Pos(), "solder.Build is only allowed as the body of an injector")
		}
	}
	// A package-level variable holding an option, e.g. db.Module.
	if id := identOf(e); id != nil {
		if v, ok := info.Uses[id].(*types.Var); ok && v.Pkg() != nil && v.Pkg().Scope().Lookup(v.Name()) == v {
			full := v.Pkg().Path() + "." + v.Name()
			if p.seen[full] {
				return p.errorf(pkg, e.Pos(), "%s is included more than once", full)
			}
			p.seen[full] = true
			vpkg, err := p.ld.syntax(v.Pkg().Path())
			if err != nil {
				return err
			}
			init := varInit(vpkg, v.Name())
			if init == nil {
				return p.errorf(pkg, e.Pos(), "cannot find the value of %s; declare it as var %s = solder.Module(...)", full, v.Name())
			}
			return p.option(vpkg, init, sc)
		}
	}
	return p.errorf(pkg, e.Pos(), "%s is not a solder option; use solder.Provide, Module, ... or a package-level variable holding one", types.ExprString(e))
}

func varInit(pkg *packages.Package, name string) ast.Expr {
	for _, f := range pkg.Syntax {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, s := range gd.Specs {
				vs := s.(*ast.ValueSpec)
				for i, n := range vs.Names {
					if n.Name == name && i < len(vs.Values) {
						return vs.Values[i]
					}
				}
			}
		}
	}
	return nil
}

func (p *parser) fn(pkg *packages.Package, e ast.Expr, kind fnKind, sc *scope) (*fn, error) {
	var obj *types.Func
	if id := identOf(e); id != nil {
		obj, _ = pkg.TypesInfo.Uses[id].(*types.Func)
	}
	if obj == nil || obj.Signature().Recv() != nil || obj.Signature().TypeParams().Len() > 0 {
		return nil, p.errorf(pkg, e.Pos(), "%s must be a package-level, non-generic function", types.ExprString(e))
	}
	if obj.Pkg().Path() != p.cur.Path() && !obj.Exported() {
		return nil, p.errorf(pkg, e.Pos(), "%s is unexported, so generated code in package %s cannot call it", types.ExprString(e), p.cur.Name())
	}
	sig := obj.Signature()
	if sig.Variadic() {
		return nil, p.errorf(pkg, e.Pos(), "%s is variadic, which is not supported", types.ExprString(e))
	}
	f := &fn{kind: kind, obj: obj, scope: sc, pos: p.ld.pos(pkg.Fset, e.Pos())}
	for i := range sig.Params().Len() {
		pr, err := p.param(pkg, e.Pos(), sig.Params().At(i).Type())
		if err != nil {
			return nil, err
		}
		f.params = append(f.params, pr)
	}
	res := sig.Results()
	n := res.Len()
	if n > 0 && isError(res.At(n-1).Type()) {
		f.hasErr = true
		n--
	}
	if kind == kInvoke {
		if n != 0 {
			return nil, p.errorf(pkg, e.Pos(), "invoked function %s may only return an error", obj.Name())
		}
		return f, nil
	}
	if n != 1 {
		return nil, p.errorf(pkg, e.Pos(), "%s must return one value and optionally an error", obj.Name())
	}
	t := res.At(0).Type()
	f.result = t
	if st := markedStruct(t, "Out"); kind == kProvider && st != nil {
		fields, err := p.fields(pkg, e.Pos(), st, "Out")
		if err != nil {
			return nil, err
		}
		f.outs = fields
	} else {
		f.outs = []field{{key: typeKey(t), typ: t}}
	}
	if kind == kDecorator {
		if len(f.params) == 0 || f.params[0].app || f.params[0].inType != nil || f.params[0].key != f.outs[0].key {
			return nil, p.errorf(pkg, e.Pos(), "decorator %s must look like func(T, deps...) T", obj.Name())
		}
	}
	return f, nil
}

func (p *parser) param(pkg *packages.Package, pos token.Pos, t types.Type) (param, error) {
	if isSolder(t, "Lifecycle") {
		return param{app: true}, nil
	}
	if st := markedStruct(t, "In"); st != nil {
		fields, err := p.fields(pkg, pos, st, "In")
		return param{inType: t, fields: fields}, err
	}
	return param{key: typeKey(t)}, nil
}

// fields reads the tagged fields of an In or Out struct. In group fields are
// []T; Out group fields are T.
func (p *parser) fields(pkg *packages.Package, pos token.Pos, st *types.Struct, marker string) ([]field, error) {
	var out []field
	for i := range st.NumFields() {
		v := st.Field(i)
		if v.Embedded() && isSolder(v.Type(), marker) {
			continue
		}
		if !v.Exported() && v.Pkg().Path() != p.cur.Path() {
			return nil, p.errorf(pkg, pos, "field %s is unexported, so generated code in package %s cannot use it", v.Name(), p.cur.Name())
		}
		tag := reflect.StructTag(st.Tag(i))
		name, group := tag.Get("name"), tag.Get("group")
		f := field{name: v.Name(), key: typeKey(v.Type()), typ: v.Type()}
		switch {
		case name != "" && group != "":
			return nil, p.errorf(pkg, pos, "field %s has both a name and a group tag", v.Name())
		case group != "" && marker == "In":
			sl, ok := v.Type().Underlying().(*types.Slice)
			if !ok {
				return nil, p.errorf(pkg, pos, "group field %s must be a slice", v.Name())
			}
			f.key = key{typ: typeKey(sl.Elem()).typ, name: group, group: true}
			p.g.groupElem[f.key] = sl.Elem()
		case group != "":
			f.key = key{typ: f.key.typ, name: group, group: true}
		case name != "":
			f.key.name = name
		}
		out = append(out, f)
	}
	return out, nil
}

func (p *parser) lifecycled(pkg *packages.Package, e ast.Expr, f *fn) error {
	if len(f.outs) != 1 || f.outs[0].name != "" {
		return p.errorf(pkg, e.Pos(), "solder.Lifecycled(%s): the constructor must return a single value, not an Out struct", f.obj.Name())
	}
	var err error
	if f.start, err = hookMethod(f.result, "Start"); err == nil {
		f.stop, err = hookMethod(f.result, "Stop")
	}
	if err != nil {
		return p.errorf(pkg, e.Pos(), "solder.Lifecycled(%s): %v", f.obj.Name(), err)
	}
	if !f.start && !f.stop {
		return p.errorf(pkg, e.Pos(), "solder.Lifecycled(%s): %s has no Start(context.Context) error or Stop(context.Context) error method", f.obj.Name(), f.result)
	}
	return nil
}

func hookMethod(t types.Type, name string) (bool, error) {
	obj, _, _ := types.LookupFieldOrMethod(t, true, nil, name)
	m, ok := obj.(*types.Func)
	if !ok {
		return false, nil
	}
	sig := m.Signature()
	if sig.Params().Len() != 1 || types.TypeString(sig.Params().At(0).Type(), nil) != "context.Context" ||
		sig.Results().Len() != 1 || !isError(sig.Results().At(0).Type()) {
		return false, fmt.Errorf("%s.%s must be func(context.Context) error", t, name)
	}
	return true, nil
}

func isError(t types.Type) bool { return types.Identical(t, types.Universe.Lookup("error").Type()) }

func isSolder(t types.Type, name string) bool {
	n, ok := types.Unalias(t).(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == solderPath && n.Obj().Name() == name
}

// markedStruct returns t's struct if it embeds solder.<marker>.
func markedStruct(t types.Type, marker string) *types.Struct {
	st, ok := types.Unalias(t).Underlying().(*types.Struct)
	if !ok {
		return nil
	}
	for i := range st.NumFields() {
		if f := st.Field(i); f.Embedded() && isSolder(f.Type(), marker) {
			return st
		}
	}
	return nil
}

// untypedConst reports whether e is an untyped constant such as 5, "x" or an
// untyped const. go/types records such arguments to ...any with their default
// type, so this looks at the syntax.
func untypedConst(info *types.Info, e ast.Expr) bool {
	if info.Types[e].Value == nil {
		return false
	}
	if _, basic := types.Unalias(info.Types[e].Type).(*types.Basic); !basic {
		return false // a named type was given explicitly
	}
	switch x := ast.Unparen(e).(type) {
	case *ast.BasicLit:
		return true
	case *ast.Ident, *ast.SelectorExpr:
		c, ok := info.Uses[identOf(x)].(*types.Const)
		if !ok {
			return false
		}
		b, ok := c.Type().(*types.Basic)
		return ok && b.Info()&types.IsUntyped != 0
	case *ast.UnaryExpr:
		return untypedConst(info, x.X)
	case *ast.BinaryExpr:
		return untypedConst(info, x.X) && untypedConst(info, x.Y)
	}
	return false // conversions like int(5)
}
