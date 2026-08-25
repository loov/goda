package graph

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/template"

	"github.com/loov/goda/internal/pkggraph"
	"github.com/loov/goda/internal/pkgtree"
)

type Dot struct {
	out io.Writer
	err io.Writer

	docs     string
	clusters bool
	nocolor  bool
	shortID  bool

	label *template.Template
}

func (ctx *Dot) Label(p *pkggraph.Node) string { return renderLabel(ctx.label, ctx.err, p) }

func (ctx *Dot) ModuleLabel(mod *pkgtree.Module) string {
	lbl := mod.Mod.Path
	if mod.Mod.Version != "" {
		lbl += "@" + mod.Mod.Version
	}
	if mod.Local {
		lbl += " (local)"
	}
	if rep := mod.Mod.Replace; rep != nil {
		lbl += " =>\\n" + rep.Path
		if rep.Version != "" {
			lbl += "@" + rep.Version
		}
	}
	return lbl
}

func (ctx *Dot) TreePackageLabel(tp *pkgtree.Package, parentPrinted bool) string {
	suffix := ""
	if parentPath := tp.Parent.Path(); parentPrinted && parentPath != "" {
		suffix = strings.TrimPrefix(tp.Path(), parentPath+"/")
	}

	if suffix != "" && ctx.shortID {
		defer func(previousID string) { tp.GraphNode.ID = previousID }(tp.GraphNode.ID)
		tp.GraphNode.ID = suffix
	}

	return ctx.Label(tp.GraphNode)
}

// dotQuote escapes s for use inside a double-quoted DOT string,
// keeping template escapes such as \l and \n intact.
func dotQuote(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

func (ctx *Dot) RepoRef(repo *pkgtree.Repo) string {
	return fmt.Sprintf(`href=%q`, ctx.docs+repo.Path())
}

func (ctx *Dot) ModuleRef(mod *pkgtree.Module) string {
	ref := ctx.docs + mod.Path()
	if mod.Mod.Version != "" {
		ref += "@" + mod.Mod.Version
	}
	return fmt.Sprintf(`href=%q`, ref)
}

func (ctx *Dot) TreePackageRef(tp *pkgtree.Package) string {
	return fmt.Sprintf(`href=%q`, ctx.docs+tp.Path())
}

func (ctx *Dot) Ref(p *pkggraph.Node) string {
	return fmt.Sprintf(`href=%q`, ctx.docs+p.ID)
}

func (ctx *Dot) writeGraphProperties() {
	if ctx.nocolor {
		fmt.Fprintf(ctx.out, "    node [fontsize=10 shape=rectangle target=\"_graphviz\"];\n")
		fmt.Fprintf(ctx.out, "    edge [tailport=e];\n")
	} else {
		fmt.Fprintf(ctx.out, "    node [penwidth=2 fontsize=10 shape=rectangle target=\"_graphviz\"];\n")
		fmt.Fprintf(ctx.out, "    edge [tailport=e penwidth=2];\n")
	}
	fmt.Fprintf(ctx.out, "    compound=true;\n")

	fmt.Fprintf(ctx.out, "    rankdir=LR;\n")
	fmt.Fprintf(ctx.out, "    newrank=true;\n")
	fmt.Fprintf(ctx.out, "    ranksep=\"1.5\";\n")
	fmt.Fprintf(ctx.out, "    quantum=\"0.5\";\n")
}

func (ctx *Dot) Write(graph *pkggraph.Graph) error {
	if ctx.clusters {
		return ctx.WriteClusters(graph)
	} else {
		return ctx.WriteRegular(graph)
	}
}

func (ctx *Dot) WriteRegular(graph *pkggraph.Graph) error {
	fmt.Fprintf(ctx.out, "digraph G {\n")
	ctx.writeGraphProperties()
	defer fmt.Fprintf(ctx.out, "}\n")

	for _, n := range graph.Sorted {
		fmt.Fprintf(ctx.out, "    %v [label=\"%v\" %v %v];\n", pkgID(n), dotQuote(ctx.Label(n)), ctx.Ref(n), ctx.colorOf(n))
	}

	for _, src := range graph.Sorted {
		for _, dst := range src.ImportsNodes {
			fmt.Fprintf(ctx.out, "    %v -> %v [%v];\n", pkgID(src), pkgID(dst), ctx.colorOf(dst))
		}
	}

	return nil
}

func (ctx *Dot) WriteClusters(graph *pkggraph.Graph) error {
	root, err := pkgtree.From(graph)
	if err != nil {
		return fmt.Errorf("failed to construct cluster tree: %v", err)
	}
	lookup := root.LookupTable()
	isCluster := map[*pkggraph.Node]bool{}

	fmt.Fprintf(ctx.out, "digraph G {\n")
	ctx.writeGraphProperties()
	defer fmt.Fprintf(ctx.out, "}\n")

	printed := make(map[pkgtree.Node]bool)

	var visit func(tn pkgtree.Node)
	visit = func(tn pkgtree.Node) {
		switch tn := tn.(type) {
		case *pkgtree.Repo:
			if tn.SameAsOnlyModule() {
				break
			}
			printed[tn] = true
			fmt.Fprintf(ctx.out, "subgraph %q {\n", "cluster_"+tn.Path())
			fmt.Fprintf(ctx.out, "    label=\"%v\"\n", tn.Path())
			fmt.Fprintf(ctx.out, "    tooltip=\"%v\"\n", tn.Path())
			fmt.Fprintf(ctx.out, "    %v\n", ctx.RepoRef(tn))
			defer fmt.Fprintf(ctx.out, "}\n")

		case *pkgtree.Module:
			printed[tn] = true
			label := ctx.ModuleLabel(tn)
			fmt.Fprintf(ctx.out, "subgraph %q {\n", "cluster_"+tn.Path())
			fmt.Fprintf(ctx.out, "    label=\"%v\"\n", dotQuote(label))
			fmt.Fprintf(ctx.out, "    tooltip=\"%v\"\n", dotQuote(label))
			fmt.Fprintf(ctx.out, "    %v\n", ctx.ModuleRef(tn))
			defer fmt.Fprintf(ctx.out, "}\n")

		case *pkgtree.Package:
			printed[tn] = true
			gn := tn.GraphNode
			if gn.ID == tn.Parent.Path() { // the module's root package, not its test variant
				isCluster[tn.GraphNode] = true
				shape := "circle"
				if tn.OnlyChild() {
					shape = "point"
				}
				fmt.Fprintf(ctx.out, "    %v [label=\"\" tooltip=\"%v\" shape=%v %v rank=0];\n", pkgID(gn), tn.Path(), shape, ctx.colorOf(gn))
			} else {
				label := dotQuote(ctx.TreePackageLabel(tn, printed[tn.Parent]))
				href := ctx.TreePackageRef(tn)
				fmt.Fprintf(ctx.out, "    %v [label=\"%v\" tooltip=\"%v\" %v %v];\n", pkgID(gn), label, dotQuote(gn.ID), href, ctx.colorOf(gn))
			}
		}

		tn.VisitChildren(visit)
	}
	root.VisitChildren(visit)

	for _, src := range graph.Sorted {
		srctree := lookup[src]
		for _, dst := range src.ImportsNodes {
			dstID := pkgID(dst)
			dstTree := lookup[dst]
			tooltip := dotQuote(src.ID + " -> " + dst.ID)

			if isCluster[dst] && srctree.Parent != dstTree.Parent {
				fmt.Fprintf(ctx.out, "    %v -> %v [tooltip=\"%v\" lhead=%q %v];\n", pkgID(src), dstID, tooltip, "cluster_"+dst.ID, ctx.colorOf(dst))
			} else {
				fmt.Fprintf(ctx.out, "    %v -> %v [tooltip=\"%v\" %v];\n", pkgID(src), dstID, tooltip, ctx.colorOf(dst))
			}
		}
	}

	return nil
}

func (ctx *Dot) colorOf(p *pkggraph.Node) string {
	if p.Color != "" {
		return "color=" + strconv.Quote(p.Color)
	}
	if ctx.nocolor {
		return ""
	}

	hue := hueOf(p)
	return "color=\"" + hslahex(hue, 0.9, 0.3, 0.7) + "\""
}
