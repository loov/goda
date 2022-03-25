package cut

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/google/subcommands"
	"golang.org/x/tools/go/packages"

	"github.com/loov/goda/internal/pkggraph"
	"github.com/loov/goda/internal/pkgset"
	"github.com/loov/goda/internal/stat"
	"github.com/loov/goda/internal/templates"
)

type Command struct {
	printStandard bool
	noAlign       bool
	format        string
	exclude       string
}

func (*Command) Name() string     { return "cut" }
func (*Command) Synopsis() string { return "Analyse indirect-dependencies." }
func (*Command) Usage() string {
	return `cut <expr>:
	Print information about indirect-dependencies.
	It shows packages whose removal would remove the most indirect dependencies.

	See "help expr" for further information about expressions.
	See "help format" for further information about formatting.
`
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&cmd.printStandard, "std", false, "print std packages")
	f.BoolVar(&cmd.noAlign, "noalign", false, "disable aligning tabs")
	f.StringVar(&cmd.format, "f", "{{.ID}}\tin:{{.InDegree}}\tpkgs:{{.Cut.PackageCount}}\tsize:{{.Cut.AllFiles.Size}}\tloc:{{.Cut.Go.Lines}}", "info formatting")
	f.StringVar(&cmd.exclude, "exclude", "", "package expr to exclude from output")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	t, err := templates.Parse(cmd.format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid label string: %v\n", err)
		return subcommands.ExitFailure
	}

	result, err := pkgset.Calc(ctx, f.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return subcommands.ExitFailure
	}

	excluded := pkgset.New()
	if cmd.exclude != "" {
		excluded, err = pkgset.Calc(ctx, strings.Fields(cmd.exclude))
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return subcommands.ExitFailure
		}
	}

	if !cmd.printStandard {
		result = pkgset.Subtract(result, pkgset.Std())
	}

	graph := pkggraph.From(result)

	nodes := map[string]*Node{}
	nodelist := []*Node{}

	var include func(parent *Node, n *pkggraph.GraphNode)
	include = func(parent *Node, n *pkggraph.GraphNode) {
		if n, ok := nodes[n.ID]; ok {
			parent.Import(n)
			return
		}

		node := &Node{
			GraphNode: n,
		}
		nodes[n.ID] = node
		if _, analyse := graph.Packages[n.ID]; analyse {
			nodelist = append(nodelist, node)
		}

		parent.Import(node)
		for _, child := range n.ImportsNodes {
			include(node, child)
		}
	}

	for _, n := range graph.Sorted {
		include(nil, n)
	}

	for _, p := range nodes {
		if !cmd.printStandard && pkgset.IsStd(p.Package) {
			continue
		}
	}

	for _, node := range nodelist {
		Reset(nodes)
		node.Cut = Erase(node)
	}

	sort.Slice(nodelist, func(i, k int) bool {
		if nodelist[i].InDegree() == nodelist[k].InDegree() {
			return nodelist[i].Cut.PackageCount > nodelist[k].Cut.PackageCount
		}
		return nodelist[i].InDegree() < nodelist[k].InDegree()
	})

	var w io.Writer = os.Stdout
	if !cmd.noAlign {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	}
	for _, node := range nodelist {
		if _, exclude := excluded[node.ID]; exclude {
			continue
		}

		err := t.Execute(w, node)
		fmt.Fprintln(w)
		if err != nil {
			fmt.Fprintf(os.Stderr, "template error: %v\n", err)
		}
	}
	if w, ok := w.(interface{ Flush() error }); ok {
		w.Flush()
	}

	return subcommands.ExitSuccess
}

func Reset(stats map[string]*Node) {
	for _, stat := range stats {
		stat.indegree = len(stat.ImportedBy)
	}
}

func Erase(stat *Node) stat.Stat {
	cut := stat.Stat
	for _, imp := range stat.Imports {
		imp.indegree--
		if imp.indegree == 0 {
			cut.Add(Erase(imp))
		}
	}
	return cut
}

type Node struct {
	*pkggraph.GraphNode

	Cut stat.Stat

	Imports    []*Node
	ImportedBy []*Node

	indegree int
}

func (parent *Node) Pkg() *packages.Package { return parent.Package }

func (parent *Node) InDegree() int  { return len(parent.ImportedBy) }
func (parent *Node) OutDegree() int { return len(parent.Imports) }

func (parent *Node) Import(child *Node) {
	if parent == nil {
		return
	}

	if !hasPackage(parent.Imports, child) {
		child.indegree++
		child.ImportedBy = append(child.ImportedBy, parent)

		parent.Imports = append(parent.Imports, child)
	}
}

func hasPackage(xs []*Node, p *Node) bool {
	for _, x := range xs {
		if x == p {
			return true
		}
	}
	return false
}
