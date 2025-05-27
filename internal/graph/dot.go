package graph

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
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

func (ctx *Dot) Label(p *pkggraph.Node) string {
	var labelText strings.Builder
	err := ctx.label.Execute(&labelText, p)
	if err != nil {
		fmt.Fprintf(ctx.err, "template error: %v\n", err)
	}
	return labelText.String()
}

func (ctx *Dot) GroupLabel(g *pkggraph.Group) string {
	var labelText strings.Builder
	err := ctx.label.Execute(&labelText, g)
	if err != nil {
		fmt.Fprintf(ctx.err, "template error: %v\n", err)
	}
	return labelText.String()
}

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
	parentPath := tp.Parent.Path()
	if parentPrinted && tp.Parent != nil && parentPath != "" {
		suffix = strings.TrimPrefix(tp.Path(), parentPath+"/")
	}

	if suffix != "" && ctx.shortID {
		defer func(previousID string) { tp.GraphNode.ID = previousID }(tp.GraphNode.ID)
		tp.GraphNode.ID = suffix
	}

	var labelText strings.Builder
	err := ctx.label.Execute(&labelText, tp.GraphNode)
	if err != nil {
		fmt.Fprintf(ctx.err, "template error: %v\n", err)
	}
	return labelText.String()
}

func (ctx *Dot) RepoRef(repo *pkgtree.Repo) string {
	return fmt.Sprintf(`href=%q`, ctx.docs+repo.Path())
}

func (ctx *Dot) ModuleRef(mod *pkgtree.Module) string {
	return fmt.Sprintf(`href=%q`, ctx.docs+mod.Path()+"@"+mod.Mod.Version)
}

func (ctx *Dot) GroupRef(g *pkggraph.Group) string {
	mod := g.FirstModule()
	if mod == nil {
		return ""
	}
	return fmt.Sprintf(`href=%q`, ctx.docs+mod.Path+"@"+mod.Version)
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

	printNode := func(n *pkggraph.Node) {
		fmt.Fprintf(ctx.out, "    %v [label=\"%v\" %v %v];\n", pkgID(n), ctx.Label(n), ctx.Ref(n), ctx.colorOf(n))
	}

	for _, group := range graph.SortedGroups {
		if len(group.Nodes) == 0 {
			continue
		}

		if group.Collapsed {
			// TODO: make it different from regular nodes
			fmt.Fprintf(ctx.out, "    %v [label=\"%v\" %v %v];\n", groupID(group), ctx.GroupLabel(group), ctx.GroupRef(group), ctx.colorOfGroup(group))
		} else {
			func() {
				fmt.Fprintf(ctx.out, "subgraph %q {\n", "cluster_"+group.ID)
				fmt.Fprintf(ctx.out, "    label=\"%v\"\n", ctx.GroupLabel(group))
				fmt.Fprintf(ctx.out, "    %v\n", ctx.GroupRef(group))
				defer fmt.Fprintf(ctx.out, "}\n")

				for _, n := range group.Nodes {
					printNode(n)
				}
			}()
		}
	}

	for _, n := range graph.Sorted {
		if n.Group == nil {
			printNode(n)
		}
	}

	for _, src := range graph.SortedGroups {
		if !src.Collapsed {
			continue
		}

		linked := map[*pkggraph.Group]bool{}
		for _, dst := range src.ImportsNodes() {

			gid := ""
			if dst.Group != nil {
				gid = dst.Group.ID
			}
			fmt.Fprintf(os.Stderr, "link %v -> %v [%v]\n", src.ID, dst.ID, gid)

			if dst.Group == nil || !dst.Group.Collapsed {
				fmt.Fprintf(ctx.out, "    %v -> %v [%v];\n", groupID(src), pkgID(dst), ctx.colorOf(dst))
				fmt.Fprintf(os.Stderr, "  skip because collapsed or no group id\n")
				continue
			}
			if linked[dst.Group] {
				fmt.Fprintf(os.Stderr, "  skip because already linked\n")
				continue
			}
			if src == dst.Group {
				fmt.Fprintf(os.Stderr, "  skip because self link\n")
				continue
			}

			// import something from a collapsed group
			linked[dst.Group] = true
			fmt.Fprintf(ctx.out, "    %v -> %v [%v];\n", groupID(src), groupID(dst.Group), ctx.colorOfGroup(dst.Group))
		}
	}

	for _, src := range graph.Sorted {
		if src.Group != nil && src.Group.Collapsed {
			continue
		}

		linked := map[*pkggraph.Group]bool{}
		for _, dst := range src.ImportsNodes {
			if dst.Group == nil || !dst.Group.Collapsed {
				fmt.Fprintf(ctx.out, "    %v -> %v [%v];\n", pkgID(src), pkgID(dst), ctx.colorOf(dst))
			} else {
				if linked[dst.Group] {
					continue
				}
				linked[dst.Group] = true
				fmt.Fprintf(ctx.out, "    %v -> %v [%v];\n", pkgID(src), groupID(dst.Group), ctx.colorOfGroup(dst.Group))
			}
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
			fmt.Fprintf(ctx.out, "    label=\"%v\"\n", label)
			fmt.Fprintf(ctx.out, "    tooltip=\"%v\"\n", label)
			fmt.Fprintf(ctx.out, "    %v\n", ctx.ModuleRef(tn))
			defer fmt.Fprintf(ctx.out, "}\n")

		case *pkgtree.Package:
			printed[tn] = true
			gn := tn.GraphNode
			if tn.Path() == tn.Parent.Path() {
				isCluster[tn.GraphNode] = true
				shape := "circle"
				if tn.OnlyChild() {
					shape = "point"
				}
				fmt.Fprintf(ctx.out, "    %v [label=\"\" tooltip=\"%v\" shape=%v %v rank=0];\n", pkgID(gn), tn.Path(), shape, ctx.colorOf(gn))
			} else {
				label := ctx.TreePackageLabel(tn, printed[tn.Parent])
				href := ctx.TreePackageRef(tn)
				fmt.Fprintf(ctx.out, "    %v [label=\"%v\" tooltip=\"%v\" %v %v];\n", pkgID(gn), label, tn.Path(), href, ctx.colorOf(gn))
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
			tooltip := src.ID + " -> " + dst.ID

			if isCluster[dst] && srctree.Parent != dstTree {
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

	hash := sha256.Sum256([]byte(p.PkgPath))
	hue := float64(uint(hash[0])<<8|uint(hash[1])) / 0xFFFF
	return "color=" + hslahex(hue, 0.9, 0.3, 0.7)
}

func (ctx *Dot) colorOfGroup(p *pkggraph.Group) string {
	if p.Color != "" {
		return "color=" + strconv.Quote(p.Color)
	}
	if ctx.nocolor {
		return ""
	}

	hash := sha256.Sum256([]byte(p.ID))
	hue := float64(uint(hash[0])<<8|uint(hash[1])) / 0xFFFF
	return "color=" + hslahex(hue, 0.9, 0.3, 0.7)
}
