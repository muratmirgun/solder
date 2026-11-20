// Package codegen turns solder injectors into plain Go.
//
// Pipeline per injector: parse (markers → graph), plan (graph → ordered
// steps), emit (steps → source). See docs/adr for the semantics.
package codegen

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

const solderPath = "github.com/muratmirgun/solder"

// File is a generated file.
type File struct {
	Path    string
	Content []byte
}

// Generate loads the packages matching patterns, relative to dir, and returns
// a solder_gen.go for every package that has injectors. Packages that fail
// are reported in the error; the files of the others are still returned.
func Generate(dir string, patterns ...string) ([]File, error) {
	pps, err := loadPlans(dir, patterns)
	var files []File
	errs := []error{err}
	for _, pp := range pps {
		src, err := emit(pp.pkg, pp.plans)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		files = append(files, File{Path: filepath.Join(pp.outDir, "solder_gen.go"), Content: src})
	}
	return files, errors.Join(errs...)
}

// loadPlans plans the injectors of every package matching patterns. Packages
// that fail are reported in the error; the plans of the others are returned.
func loadPlans(dir string, patterns []string) ([]*pkgPlans, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real // go list reports resolved paths; keep positions relative
	}
	ld := &loader{dir: dir, pkgs: map[string]*packages.Package{}}
	roots, err := ld.load(patterns...)
	if err != nil {
		return nil, err
	}
	var out []*pkgPlans
	var errs []error
	for _, pkg := range roots {
		pp, err := planPackage(ld, pkg)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if pp != nil {
			out = append(out, pp)
		}
	}
	return out, errors.Join(errs...)
}

type loader struct {
	dir  string
	pkgs map[string]*packages.Package
}

const loadMode = packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
	packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports

func (l *loader) load(patterns ...string) ([]*packages.Package, error) {
	cfg := &packages.Config{Mode: loadMode, Dir: l.dir, BuildFlags: []string{"-tags=solderinject"}}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages match %s (is go.mod up to date? try go mod tidy)", strings.Join(patterns, " "))
	}
	var errs []error
	for _, p := range pkgs {
		for _, e := range p.Errors {
			errs = append(errs, e)
		}
		l.pkgs[p.PkgPath] = p
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return pkgs, nil
}

// syntax returns the package at path with syntax and type info, loading it
// on first use. Used to read module variables declared in other packages.
// Objects from separate loads are not identical, which is why graph keys are
// type strings, not types.Type.
func (l *loader) syntax(path string) (*packages.Package, error) {
	if p, ok := l.pkgs[path]; ok && p.Syntax != nil {
		return p, nil
	}
	pkgs, err := l.load(path)
	if err != nil {
		return nil, err
	}
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("loading %s: got %d packages", path, len(pkgs))
	}
	return pkgs[0], nil
}

// pos formats p relative to the working directory when possible.
func (l *loader) pos(fset *token.FileSet, p token.Pos) string {
	ps := fset.Position(p)
	if rel, err := filepath.Rel(l.dir, ps.Filename); err == nil && !strings.HasPrefix(rel, "..") {
		ps.Filename = rel
	}
	return ps.String()
}

type injector struct {
	decl  *ast.FuncDecl
	sig   *types.Signature
	build *ast.CallExpr
	pos   string
}

// pkgPlans is the planned injectors of one package.
type pkgPlans struct {
	pkg    *packages.Package
	outDir string
	plans  []*plan
}

// planPackage plans every injector in pkg; nil if it has none.
func planPackage(ld *loader, pkg *packages.Package) (*pkgPlans, error) {
	var injs []*injector
	var outDir string
	for _, f := range pkg.Syntax {
		var found []*injector
		var other ast.Decl
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok {
				if call := buildCall(pkg.TypesInfo, fd); call != nil {
					obj := pkg.TypesInfo.Defs[fd.Name].(*types.Func)
					found = append(found, &injector{decl: fd, sig: obj.Signature(), build: call, pos: ld.pos(pkg.Fset, fd.Name.Pos())})
					continue
				}
			}
			if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
				continue
			}
			if other == nil {
				other = d
			}
		}
		if len(found) == 0 {
			continue
		}
		if other != nil {
			return nil, fmt.Errorf("%s: injector files may only contain injectors and imports; move this declaration to a file without the solderinject build tag", ld.pos(pkg.Fset, other.Pos()))
		}
		injs = append(injs, found...)
		outDir = filepath.Dir(pkg.Fset.Position(f.Pos()).Filename)
	}
	if len(injs) == 0 {
		return nil, nil
	}
	var plans []*plan
	var errs []error
	for _, inj := range injs {
		pl, err := planInjector(ld, pkg, inj)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		plans = append(plans, pl)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return &pkgPlans{pkg: pkg, outDir: outDir, plans: plans}, nil
}

// buildCall returns the solder.Build call if fd's body is exactly
// panic(solder.Build(...)).
func buildCall(info *types.Info, fd *ast.FuncDecl) *ast.CallExpr {
	if fd.Recv != nil || fd.Body == nil || len(fd.Body.List) != 1 {
		return nil
	}
	es, ok := fd.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return nil
	}
	p, ok := es.X.(*ast.CallExpr)
	if !ok || len(p.Args) != 1 {
		return nil
	}
	if id, ok := ast.Unparen(p.Fun).(*ast.Ident); !ok || info.Uses[id] != types.Universe.Lookup("panic") {
		return nil
	}
	call, ok := ast.Unparen(p.Args[0]).(*ast.CallExpr)
	if !ok {
		return nil
	}
	if name, _ := markerName(info, call); name != "Build" {
		return nil
	}
	return call
}

// markerName reports which solder function call calls, or "".
func markerName(info *types.Info, call *ast.CallExpr) (string, *ast.Ident) {
	fun := ast.Unparen(call.Fun)
	switch x := fun.(type) {
	case *ast.IndexExpr:
		fun = x.X
	case *ast.IndexListExpr:
		fun = x.X
	}
	id := identOf(fun)
	if id == nil {
		return "", nil
	}
	f, ok := info.Uses[id].(*types.Func)
	if !ok || f.Pkg() == nil || f.Pkg().Path() != solderPath {
		return "", nil
	}
	return f.Name(), id
}

// identOf returns the identifier naming e (x or pkg.x), or nil.
func identOf(e ast.Expr) *ast.Ident {
	switch x := ast.Unparen(e).(type) {
	case *ast.Ident:
		return x
	case *ast.SelectorExpr:
		return x.Sel
	}
	return nil
}
