package codegen

import (
	"errors"
	"fmt"
	"go/types"
	"strings"
)

// Graph returns a Mermaid flowchart per injector in the packages matching
// patterns: one node per constructed value, edges from dependency to user.
func Graph(dir string, patterns ...string) (string, error) {
	pps, err := loadPlans(dir, patterns)
	var b strings.Builder
	for _, pp := range pps {
		for _, pl := range pp.plans {
			mermaid(&b, pl)
		}
	}
	return b.String(), errors.Join(err)
}

func mermaid(b *strings.Builder, pl *plan) {
	qual := func(p *types.Package) string { return p.Name() }
	fmt.Fprintf(b, "%%%% injector %s (%s)\nflowchart LR\n", pl.inj.decl.Name.Name, pl.inj.pos)
	ids := map[*variable]string{}
	n := 0
	node := func(label string) string {
		n++
		id := fmt.Sprintf("n%d", n)
		fmt.Fprintf(b, "  %s[\"%s\"]\n", id, strings.ReplaceAll(label, `"`, "#quot;"))
		return id
	}
	edge := func(from *variable, to string) {
		if id, ok := ids[from]; ok {
			fmt.Fprintf(b, "  %s --> %s\n", id, to)
		}
	}
	for _, p := range pl.params {
		ids[p] = node(strings.TrimSpace("input " + p.name + " " + types.TypeString(p.typ, qual)))
	}
	for _, s := range pl.steps {
		switch s.kind {
		case sApp:
			ids[s.out] = node("solder.App")
		case sSupply:
			ids[s.out] = node("supply " + types.ExprString(s.supply.expr))
		case sGroup:
			id := node("group " + types.TypeString(s.out.typ, qual))
			for _, r := range s.elems {
				edge(r.v, id)
			}
			ids[s.out] = id
		case sHook:
			fmt.Fprintf(b, "  %s -.->|hooks| %s\n", ids[s.out], ids[pl.app])
		case sCall:
			id := node(s.fn.obj.Pkg().Name() + "." + s.fn.obj.Name())
			seen := map[*variable]bool{}
			for _, a := range s.args {
				refs := []ref{a.ref}
				if a.app {
					refs = []ref{{v: pl.app}}
				}
				for _, f := range a.fields {
					refs = append(refs, f.ref)
				}
				for _, r := range refs {
					if r.v != nil && !seen[r.v] {
						seen[r.v] = true
						edge(r.v, id)
					}
				}
			}
			if s.out != nil {
				ids[s.out] = id
			}
		}
	}
	if !pl.returnsApp {
		fmt.Fprintf(b, "  %s --> result((%s))\n", ids[pl.result.v], pl.inj.decl.Name.Name)
	}
	b.WriteString("\n")
}
