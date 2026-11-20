package codegen

import (
	"fmt"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

type variable struct {
	name string // set by the emitter; injector parameters may arrive named
	typ  types.Type
}

// ref is a variable, or a field of an Out struct variable.
type ref struct {
	v     *variable
	field string
}

type arg struct {
	app    bool
	ref    ref
	inType types.Type
	fields []fieldVal
}

type fieldVal struct {
	name string
	ref  ref
}

type stepKind int

const (
	sApp    stepKind = iota // app := solder.New()
	sCall                   // [out[, err] :=] fn(args)
	sSupply                 // out := expr
	sGroup                  // out := []T{elems}
	sHook                   // app.AppendService(out)
)

type step struct {
	kind   stepKind
	fn     *fn
	args   []arg
	out    *variable
	supply *producer
	elems  []ref
}

type plan struct {
	inj        *injector
	params     []*variable
	steps      []step
	app        *variable
	result     ref
	returnsApp bool
	hasErr     bool
}

type resolveKey struct {
	k  key
	sc *scope
}

type planner struct {
	g     *graph
	pl    *plan
	memo  map[any]ref
	busy  map[any]bool
	chain []string
}

func planInjector(ld *loader, pkg *packages.Package, inj *injector) (*plan, error) {
	sig := inj.sig
	name := inj.decl.Name.Name
	if sig.TypeParams().Len() > 0 || sig.Variadic() {
		return nil, fmt.Errorf("%s: injector %s must not be generic or variadic", inj.pos, name)
	}
	pl := &plan{inj: inj}
	res := sig.Results()
	n := res.Len()
	if n > 0 && isError(res.At(n-1).Type()) {
		pl.hasErr = true
		n--
	}
	if n != 1 {
		return nil, fmt.Errorf("%s: injector %s must return one value and optionally an error", inj.pos, name)
	}
	want := res.At(0).Type()

	g := &graph{root: &scope{}, producers: map[key]*producer{}, groups: map[key][]*producer{}, groupElem: map[key]types.Type{}}
	p := &parser{ld: ld, cur: pkg.Types, g: g, seen: map[string]bool{}}
	for i := range sig.Params().Len() {
		v := sig.Params().At(i)
		pv := &variable{typ: v.Type()}
		if v.Name() != "_" {
			pv.name = v.Name()
		}
		pl.params = append(pl.params, pv)
		if err := p.add(typeKey(v.Type()), &producer{kind: pInput, pos: inj.pos, input: pv, typ: v.Type()}); err != nil {
			return nil, err
		}
	}
	if err := p.options(pkg, inj.build.Args, g.root); err != nil {
		return nil, err
	}

	pr := &planner{g: g, pl: pl, memo: map[any]ref{}, busy: map[any]bool{}}
	if ptr, ok := types.Unalias(want).(*types.Pointer); ok && isSolder(ptr.Elem(), "App") {
		pl.returnsApp = true
		pl.app = &variable{typ: want}
		pl.steps = append(pl.steps, step{kind: sApp, out: pl.app})
	}
	top := fmt.Sprintf("injector %s (%s)", name, inj.pos)
	for _, f := range g.roots {
		pr.chain = []string{top}
		var err error
		if f.kind == kInvoke {
			err = pr.invoke(f)
		} else {
			_, err = pr.call(f)
		}
		if err != nil {
			return nil, err
		}
	}
	if !pl.returnsApp {
		pr.chain = []string{top}
		r, err := pr.resolve(typeKey(want), g.root)
		if err != nil {
			return nil, err
		}
		pl.result = r
	}
	return pl, nil
}

func (pr *planner) errorf(format string, args ...any) error {
	return fmt.Errorf("%s", fmt.Sprintf(format, args...)+pr.trace())
}

func (pr *planner) trace() string {
	var b strings.Builder
	for i := len(pr.chain) - 1; i >= 0; i-- {
		b.WriteString("\n\tneeded by " + pr.chain[i])
	}
	return b.String()
}

func (pr *planner) injName() string { return pr.pl.inj.decl.Name.Name }

func (pr *planner) needApp(why string) error {
	if pr.pl.app == nil {
		return fmt.Errorf("%s: %s, so injector %s must return *solder.App", pr.pl.inj.pos, why, pr.injName())
	}
	return nil
}

func (pr *planner) needErr(f *fn) error {
	if f.hasErr && !pr.pl.hasErr {
		return fmt.Errorf("%s: %s returns an error, so injector %s must return an error too", f.pos, f.obj.Name(), pr.injName())
	}
	return nil
}

// call constructs provider f once and returns its result variable.
func (pr *planner) call(f *fn) (*variable, error) {
	if r, ok := pr.memo[f]; ok {
		return r.v, nil
	}
	if pr.busy[f] {
		return nil, fmt.Errorf("dependency cycle: %s depends on itself%s", f, pr.trace())
	}
	if err := pr.needErr(f); err != nil {
		return nil, err
	}
	if f.start || f.stop {
		if err := pr.needApp(fmt.Sprintf("solder.Lifecycled(%s) registers hooks", f.obj.Name())); err != nil {
			return nil, err
		}
	}
	pr.busy[f] = true
	pr.chain = append(pr.chain, f.String())
	args, err := pr.args(f, f.params)
	pr.chain = pr.chain[:len(pr.chain)-1]
	delete(pr.busy, f)
	if err != nil {
		return nil, err
	}
	v := &variable{typ: f.result}
	pr.pl.steps = append(pr.pl.steps, step{kind: sCall, fn: f, args: args, out: v})
	if f.start || f.stop {
		pr.pl.steps = append(pr.pl.steps, step{kind: sHook, out: v})
	}
	pr.memo[f] = ref{v: v}
	return v, nil
}

func (pr *planner) invoke(f *fn) error {
	if err := pr.needApp(fmt.Sprintf("solder.Invoke(%s) is used", f.obj.Name())); err != nil {
		return err
	}
	if err := pr.needErr(f); err != nil {
		return err
	}
	pr.chain = append(pr.chain, f.String())
	args, err := pr.args(f, f.params)
	pr.chain = pr.chain[:len(pr.chain)-1]
	if err != nil {
		return err
	}
	pr.pl.steps = append(pr.pl.steps, step{kind: sCall, fn: f, args: args})
	return nil
}

func (pr *planner) args(f *fn, params []param) ([]arg, error) {
	var args []arg
	for _, p := range params {
		switch {
		case p.app:
			if err := pr.needApp(fmt.Sprintf("%s takes solder.Lifecycle", f.obj.Name())); err != nil {
				return nil, err
			}
			args = append(args, arg{app: true})
		case p.inType != nil:
			a := arg{inType: p.inType}
			for _, fl := range p.fields {
				r, err := pr.resolve(fl.key, f.scope)
				if err != nil {
					return nil, err
				}
				a.fields = append(a.fields, fieldVal{name: fl.name, ref: r})
			}
			args = append(args, a)
		default:
			r, err := pr.resolve(p.key, f.scope)
			if err != nil {
				return nil, err
			}
			args = append(args, arg{ref: r})
		}
	}
	return args, nil
}

// resolve returns the value of k as seen from scope sc: the provided value,
// wrapped by the decorators of sc and its ancestors (outermost first).
func (pr *planner) resolve(k key, sc *scope) (ref, error) {
	if k.group {
		return pr.group(k)
	}
	rk := resolveKey{k, sc}
	if r, ok := pr.memo[rk]; ok {
		return r, nil
	}
	if pr.busy[rk] {
		return ref{}, pr.errorf("dependency cycle through %s", k)
	}
	pr.busy[rk] = true
	defer delete(pr.busy, rk)

	var r ref
	var err error
	if sc.parent != nil {
		r, err = pr.resolve(k, sc.parent)
	} else {
		r, err = pr.raw(k)
	}
	if err != nil {
		return ref{}, err
	}
	for _, d := range sc.decs[k] {
		if err := pr.needErr(d); err != nil {
			return ref{}, err
		}
		pr.chain = append(pr.chain, d.String())
		args, err := pr.args(d, d.params[1:])
		pr.chain = pr.chain[:len(pr.chain)-1]
		if err != nil {
			return ref{}, err
		}
		v := &variable{typ: d.result}
		pr.pl.steps = append(pr.pl.steps, step{kind: sCall, fn: d, args: append([]arg{{ref: r}}, args...), out: v})
		r = ref{v: v}
	}
	pr.memo[rk] = r
	return r, nil
}

// raw returns the undecorated value of k.
func (pr *planner) raw(k key) (ref, error) {
	p, ok := pr.g.producers[k]
	if !ok {
		return ref{}, pr.missing(k)
	}
	switch p.kind {
	case pInput:
		return ref{v: p.input}, nil
	case pSupply:
		if r, ok := pr.memo[p]; ok {
			return r, nil
		}
		v := &variable{typ: p.typ}
		pr.pl.steps = append(pr.pl.steps, step{kind: sSupply, supply: p, out: v})
		pr.memo[p] = ref{v: v}
		return ref{v: v}, nil
	case pBind:
		// shortcut: the bound type is resolved at the root, so module decorators of T don't apply through I.
		return pr.resolve(p.target, pr.g.root)
	}
	v, err := pr.call(p.fn)
	if err != nil {
		return ref{}, err
	}
	return ref{v: v, field: p.field}, nil
}

func (pr *planner) group(k key) (ref, error) {
	if r, ok := pr.memo[k]; ok {
		return r, nil
	}
	var elems []ref
	for _, p := range pr.g.groups[k] {
		v, err := pr.call(p.fn)
		if err != nil {
			return ref{}, err
		}
		elems = append(elems, ref{v: v, field: p.field})
	}
	v := &variable{typ: types.NewSlice(pr.g.groupElem[k])}
	pr.pl.steps = append(pr.pl.steps, step{kind: sGroup, out: v, elems: elems})
	pr.memo[k] = ref{v: v}
	return ref{v: v}, nil
}

func (pr *planner) missing(k key) error {
	hint := ""
	alt := key{typ: "*" + k.typ, name: k.name}
	if strings.HasPrefix(k.typ, "*") {
		alt.typ = k.typ[1:]
	}
	if p, ok := pr.g.producers[alt]; ok {
		hint = fmt.Sprintf(" (%s is provided at %s)", alt, p.pos)
	}
	return pr.errorf("no provider for %s%s", k, hint)
}
