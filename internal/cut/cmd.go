package cut

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/google/subcommands"
	"golang.org/x/tools/go/packages"

	"github.com/loov/goda/internal/pkggraph"
	"github.com/loov/goda/internal/pkgset"
	"github.com/loov/goda/internal/stat"
	"github.com/loov/goda/internal/templates"
)

type Command struct {
	printStandard bool
	exclude       string

	noAlign bool
	header  string
	format  string
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
	f.StringVar(&cmd.exclude, "exclude", "", "package expr to exclude from output")

	f.BoolVar(&cmd.noAlign, "noalign", false, "disable aligning tabs")
	f.StringVar(&cmd.header, "header", "", "header for the table\nautomatically derives from format, when empty, use \"-\" to skip")
	f.StringVar(&cmd.format, "f", "{{.ID}}\t{{.InDegree}}\t{{.Cut.PackageCount}}\t{{.Cut.AllFiles.Size}}\t{{.Cut.Go.Lines}}", "info formatting")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	t, err := templates.Parse(cmd.format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid label string: %v\n", err)
		return subcommands.ExitFailure
	}

	if !cmd.printStandard {
		go pkgset.LoadStd()
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

	if cmd.print(os.Stdout, t, result, excluded) {
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

// print writes the cut table for result to w, returning whether a template failed.
func (cmd *Command) print(w io.Writer, t *template.Template, result, excluded pkgset.Set) (failed bool) {
	graph := pkggraph.From(result)

	// pkggraph.From only links nodes inside the graph, so a flat two-pass build suffices.
	nodes := map[string]*Node{}
	nodelist := []*Node{}
	for _, n := range graph.Sorted {
		node := &Node{Node: n}
		nodes[n.ID] = node
		nodelist = append(nodelist, node)
	}
	for _, node := range nodelist {
		for _, child := range node.ImportsNodes {
			node.Import(nodes[child.ID])
		}
	}

	for _, node := range nodelist {
		node.Cut = Erase(node)
	}

	slices.SortFunc(nodelist, func(a, b *Node) int {
		return cmp.Or(
			cmp.Compare(a.InDegree(), b.InDegree()),
			cmp.Compare(b.Cut.PackageCount, a.Cut.PackageCount),
			cmp.Compare(a.ID, b.ID),
		)
	})

	if !cmd.noAlign {
		w = tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	}
	if cmd.header != "-" {
		if cmd.header == "" {
			cmd.header = templates.Header(cmd.format)
		}
		fmt.Fprintln(w, cmd.header)
	}
	for _, node := range nodelist {
		if _, exclude := excluded[node.ID]; exclude {
			continue
		}

		if err := t.Execute(w, node); err != nil {
			fmt.Fprintf(os.Stderr, "template error: %v\n", err)
			failed = true
		}
		fmt.Fprintln(w)
	}
	if w, ok := w.(interface{ Flush() error }); ok {
		w.Flush()
	}
	return failed
}

// Erase returns the stats of packages that become unreachable when root is removed.
func Erase(root *Node) stat.Stat {
	var touched []*Node
	var erase func(n *Node) stat.Stat
	erase = func(n *Node) stat.Stat {
		cut := n.Stat
		for _, imp := range n.Imports {
			if imp.indegree == len(imp.ImportedBy) {
				touched = append(touched, imp)
			}
			imp.indegree--
			if imp.indegree == 0 {
				cut.Add(erase(imp))
			}
		}
		return cut
	}
	cut := erase(root)
	// restore only what we touched instead of resetting every node
	for _, n := range touched {
		n.indegree = len(n.ImportedBy)
	}
	return cut
}

type Node struct {
	*pkggraph.Node

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

	if !slices.Contains(parent.Imports, child) {
		child.indegree++
		child.ImportedBy = append(child.ImportedBy, parent)

		parent.Imports = append(parent.Imports, child)
	}
}
