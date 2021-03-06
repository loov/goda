package graph

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/loov/goda/pkggraph"
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

func (ctx *Dot) ClusterLabel(tree *pkggraph.Tree, parentPrinted bool) string {
	var suffix = ""
	if parentPrinted && tree.Parent != nil && tree.Parent.Path != "" {
		suffix = "./" + strings.TrimPrefix(tree.Path, tree.Parent.Path+"/")
	}

	if parentPrinted && suffix != "" && ctx.shortID {
		return suffix
	}
	return tree.Path
}

func (ctx *Dot) TreeLabel(tree *pkggraph.Tree, parentPrinted bool) string {
	var suffix = ""
	if parentPrinted && tree.Parent != nil && tree.Parent.Path != "" {
		suffix = strings.TrimPrefix(tree.Path, tree.Parent.Path+"/")
	}

	if tree.Package == nil {
		if parentPrinted && suffix != "" && ctx.shortID {
			return suffix
		}
		return tree.Path
	}

	if suffix != "" && ctx.shortID {
		defer func(previousID string) { tree.Package.ID = previousID }(tree.Package.ID)
		tree.Package.ID = suffix
	}

	var labelText strings.Builder
	err := ctx.label.Execute(&labelText, tree.Package)
	if err != nil {
		fmt.Fprintf(ctx.err, "template error: %v\n", err)
	}
	return labelText.String()
}

func (ctx *Dot) Ref(p *pkggraph.Node) string {
	return fmt.Sprintf(`href=%q `, ctx.docs+p.ID)
}

func (ctx *Dot) TreeRef(tree *pkggraph.Tree) string {
	return fmt.Sprintf(`href=%q `, ctx.docs+tree.Path)
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

func (ctx *Dot) Write(graph *pkggraph.Graph) {
	if ctx.clusters {
		ctx.WriteClusters(graph)
	} else {
		ctx.WriteRegular(graph)
	}
}

func (ctx *Dot) WriteRegular(graph *pkggraph.Graph) {
	fmt.Fprintf(ctx.out, "digraph G {\n")
	ctx.writeGraphProperties()
	defer fmt.Fprintf(ctx.out, "}\n")

	for _, n := range graph.Sorted {
		fmt.Fprintf(ctx.out, "    %v [label=\"%v\" %v %v];\n", pkgID(n), ctx.Label(n), ctx.Ref(n), ctx.colorOf(n))
	}

	for _, src := range graph.Sorted {
		for _, dst := range src.ImportsNodes {
			fmt.Fprintf(ctx.out, "    %v -> %v [%v];\n", pkgID(src), pkgID(dst), ctx.colorOf(dst))
		}
	}
}

func (ctx *Dot) WriteClusters(graph *pkggraph.Graph) {
	fmt.Fprintf(ctx.out, "digraph G {\n")
	ctx.writeGraphProperties()
	defer fmt.Fprintf(ctx.out, "}\n")

	var walk func(bool, *pkggraph.Tree)
	root := graph.Tree()
	lookup := root.LookupTable()
	isCluster := map[*pkggraph.Node]bool{}

	walk = func(parentPrinted bool, tree *pkggraph.Tree) {
		p := tree.Package
		if len(tree.Children) == 0 {
			label := ctx.TreeLabel(tree, parentPrinted)
			href := ctx.TreeRef(tree)
			fmt.Fprintf(ctx.out, "    %v [label=\"%v\" tooltip=\"%v\" %v %v];\n", pkgID(p), label, tree.Path, href, ctx.colorOf(p))
			return
		}

		print := p != nil
		if p != nil {
			print = true
		}

		childPackageCount := 0
		for _, child := range tree.Children {
			if child.Package != nil {
				childPackageCount++
			}
		}
		if childPackageCount > 1 {
			print = true
		}

		if tree.Path == "" {
			print = false
		}

		if print {
			fmt.Fprintf(ctx.out, "subgraph cluster_%v {\n", escapeID(tree.Path))
			if tree.Package != nil {
				isCluster[tree.Package] = true
				fmt.Fprintf(ctx.out, "    %v [label=\"\" tooltip=\"%v\" shape=circle %v rank=0];\n", pkgID(p), tree.Path, ctx.colorOf(p))
			}
			fmt.Fprintf(ctx.out, "    label=\"%v\"\n", ctx.ClusterLabel(tree, parentPrinted))
			fmt.Fprintf(ctx.out, "    tooltip=\"%v\"\n", tree.Path)
			fmt.Fprintf(ctx.out, "    %v\n", ctx.TreeRef(tree))
			defer fmt.Fprintf(ctx.out, "}\n")
		}

		for _, child := range tree.Children {
			walk(print, child)
		}
	}
	walk(false, root)

	for _, src := range graph.Sorted {
		srctree := lookup[src]
		for _, dst := range src.ImportsNodes {
			dstid := pkgID(dst)
			dsttree := lookup[dst]
			tooltip := src.ID + " -> " + dst.ID

			if isCluster[dst] && !srctree.HasParent(dsttree) {
				fmt.Fprintf(ctx.out, "    %v -> %v [tooltip=\"%v\" lhead=cluster_%v %v];\n", pkgID(src), dstid, tooltip, dstid, ctx.colorOf(dst))
			} else {
				fmt.Fprintf(ctx.out, "    %v -> %v [tooltip=\"%v\" %v];\n", pkgID(src), dstid, tooltip, ctx.colorOf(dst))
			}
		}
	}
}

func (ctx *Dot) colorOf(p *pkggraph.Node) string {
	if ctx.nocolor {
		return ""
	}

	hash := sha256.Sum256([]byte(p.PkgPath))
	hue := float64(uint(hash[0])<<8|uint(hash[1])) / 0xFFFF
	return "color=" + hslahex(hue, 0.9, 0.3, 0.7)
}
