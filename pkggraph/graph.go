package pkggraph

import (
	"sort"

	"golang.org/x/tools/go/packages"

	"github.com/loov/goda/pkgset"
	"github.com/loov/goda/stat"
)

type Graph struct {
	Packages map[string]*Node
	Sorted   []*Node
	stat.Stat
}

func (g *Graph) AddNode(n *Node) { g.Packages[n.ID] = n }

type Node struct {
	ImportsNodes    []*Node
	ImportedByNodes []*Node

	stat.Stat
	Imports    stat.Stat
	ImportedBy stat.Stat

	Errors []error
	Graph  *Graph

	*packages.Package
}

func (n *Node) Pkg() *packages.Package { return n.Package }

func FromSet(pkgs pkgset.Set) *Graph {
	return From(map[string]*packages.Package(pkgs))
}

// From creates a new graph from a map of packages.
func From(pkgs map[string]*packages.Package) *Graph {
	g := &Graph{Packages: map[string]*Node{}}

	for _, p := range pkgs {
		n := LoadNode(p)
		g.Sorted = append(g.Sorted, n)
		g.AddNode(n)
		g.Stat.Add(n.Stat)
	}
	SortNodes(g.Sorted)

	// TODO: find ways to improve performance.

	cache := allImportsCache(pkgs)

	for _, n := range g.Packages {
		importsIDs := cache[n.ID]
		n.ImportsNodes = make([]*Node, 0, len(importsIDs))
		for _, id := range importsIDs {
			imported, ok := g.Packages[id]
			if !ok {
				// we may not want to print info about every package
				continue
			}

			n.ImportsNodes = append(n.ImportsNodes, imported)
			imported.ImportedByNodes = append(imported.ImportedByNodes, n)

			n.Imports.Add(imported.Stat)
			imported.ImportedBy.Add(n.Stat)
		}
	}

	for _, n := range g.Packages {
		SortNodes(n.ImportsNodes)
		SortNodes(n.ImportedByNodes)
	}

	return g
}

func LoadNode(p *packages.Package) *Node {
	node := &Node{}
	node.Package = p

	stat, errs := stat.Package(p)
	node.Errors = append(node.Errors, errs...)
	node.Stat = stat

	return node
}

func SortNodes(xs []*Node) {
	sort.Slice(xs, func(i, k int) bool { return xs[i].ID < xs[k].ID })
}
