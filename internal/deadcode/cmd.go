package deadcode

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/google/subcommands"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type Command struct {
	outputType string
}

func (*Command) Name() string     { return "deadcode" }
func (*Command) Synopsis() string { return "Find conservative mode dependency." }
func (*Command) Usage() string {
	return `deadcode <main package>:
	Find code that triggers conservative mode.
`
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.StringVar(&cmd.outputType, "type", "dot", "output type (dot)")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "missing main package argument\n")
		return subcommands.ExitFailure
	}

	build := exec.Command("go", "build", "-o", os.DevNull, "-ldflags=-c", f.Arg(0))
	var stdout, stderr bytes.Buffer
	build.Stdout = &stdout
	build.Stderr = &stderr
	err := build.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n", stderr.String())
		return subcommands.ExitFailure
	}

	graph := parseCallGraph(stderr.String())
	reversed := transpose(graph)
	reaches := reachableNodes(reversed, newNodes("reflect.Value.MethodByName", "reflect.Value.Method"))
	filtered := filterGraph(graph, reaches)

	printDot(filtered)

	return subcommands.ExitSuccess
}

func printDot(g graph) {
	pf := fmt.Printf
	pf("digraph G {\n")
	pf("  rankdir=LR")

	pf("  node [fontsize=10 shape=rectangle target=\"_graphviz\"];\n")
	pf("  edge [tailport=e];\n")
	defer pf("}\n")

	for _, n := range g.list() {
		for _, e := range n.edges {
			pf("  %q -> %q\n", n.name, e)
		}
	}
}

// implementation loosely based on golang.org/x/tools/cmd/digraph

type graph map[string]nodes
type nodes map[string]struct{}

type node struct {
	name  string
	edges []string
}

func newNodes(xs ...string) nodes {
	n := make(nodes)
	for _, x := range xs {
		n.add(x)
	}
	return n
}

func (g graph) edge(from, to string) {
	edges, ok := g[from]
	if !ok {
		edges = make(nodes)
		g[from] = edges
	}
	edges.add(to)
}

func (g graph) list() []node {
	xs := []node{}
	for k, v := range g {
		xs = append(xs, node{
			name:  k,
			edges: v.list(),
		})
	}
	slices.SortFunc(xs, func(a, b node) bool {
		return a.name < b.name
	})
	return xs
}

func (n nodes) add(x string) {
	n[x] = struct{}{}
}

func (n nodes) has(x string) bool {
	_, ok := n[x]
	return ok
}

func (n nodes) list() []string {
	keys := maps.Keys(n)
	sort.Strings(keys)
	return keys
}

func parseCallGraph(out string) graph {
	g := make(graph)

	for _, line := range strings.Split(out, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		from, to, ok := strings.Cut(line, " calls ")
		if !ok {
			continue
		}

		g.edge(from, to)
	}

	return g
}

func transpose(g graph) graph {
	x := make(graph)
	for from, edges := range g {
		for to := range edges {
			x.edge(to, from)
		}
	}
	return x
}

func reachableNodes(g graph, roots nodes) nodes {
	seen := make(nodes)
	var visit func(node string)
	visit = func(node string) {
		if !seen.has(node) {
			seen.add(node)
			for e := range g[node] {
				visit(e)
			}
		}
	}
	for root := range roots {
		visit(root)
	}
	return seen
}

func filterGraph(g graph, keep nodes) graph {
	x := make(graph)
	for from, edges := range g {
		if !keep.has(from) {
			continue
		}

		for to := range edges {
			if !keep.has(to) {
				continue
			}
			x.edge(from, to)
		}
	}
	return x
}
