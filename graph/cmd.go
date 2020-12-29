package graph

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/google/subcommands"

	"github.com/loov/goda/pkggraph"
	"github.com/loov/goda/pkgset"
	"github.com/loov/goda/templates"
)

type Command struct {
	printStandard bool

	docs string

	outputType  string
	labelFormat string

	nocolor bool

	clusters bool
	shortID  bool
}

func (*Command) Name() string     { return "graph" }
func (*Command) Synopsis() string { return "Print dependency graph." }
func (*Command) Usage() string {
	return `graph <expr>:
	Print dependency dot graph.

	See "help expr" for further information about expressions.
`
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&cmd.printStandard, "std", false, "print std packages")

	f.BoolVar(&cmd.nocolor, "nocolor", false, "disable coloring")

	f.StringVar(&cmd.docs, "docs", "https://pkg.go.dev/", "override the docs url to use")

	f.StringVar(&cmd.outputType, "type", "dot", "output type")
	f.StringVar(&cmd.labelFormat, "f", "{{.ID}}\\l{{ .Stat.GoFiles.Lines }} / {{ .Stat.GoFiles.Size }}\\l", "label formatting")

	f.BoolVar(&cmd.clusters, "cluster", false, "create clusters")
	f.BoolVar(&cmd.shortID, "short", false, "use short package id-s")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "missing package names\n")
		return subcommands.ExitUsageError
	}

	if cmd.outputType != "dot" {
		fmt.Fprintf(os.Stderr, "unknown output type %v\n", cmd.outputType)
		return subcommands.ExitUsageError
	}

	label, err := templates.Parse(cmd.labelFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid label format: %v\n", err)
		return subcommands.ExitFailure
	}

	result, err := pkgset.Calc(ctx, f.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return subcommands.ExitFailure
	}
	if !cmd.printStandard {
		result = pkgset.Subtract(result, pkgset.Std())
	}

	dot := &Dot{
		out:     os.Stdout,
		err:     os.Stderr,
		docs:    cmd.docs,
		nocolor: cmd.nocolor,
		shortID: cmd.shortID,
		label:   label,
	}

	graph := pkggraph.FromSet(result)
	if cmd.clusters {
		dot.WriteClusters(graph)
	} else {
		dot.WriteRegular(graph)
	}

	return subcommands.ExitSuccess
}

type Dot struct {
	out io.Writer
	err io.Writer

	docs    string
	nocolor bool
	shortID bool

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

func pkgID(p *pkggraph.Node) string {
	return escapeID(p.ID)
}

func escapeID(s string) string {
	var d []byte
	for _, r := range s {
		if 'a' <= r && r <= 'z' {
			d = append(d, byte(r))
		} else if 'A' <= r && r <= 'Z' {
			d = append(d, byte(r))
		} else if '0' <= r && r <= '9' {
			d = append(d, byte(r))
		} else {
			d = append(d, '_')
		}
	}
	return "n_" + string(d)
}
