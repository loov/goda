package pkggraph

import (
	"encoding/json"
	"maps"
	"slices"
	"sort"

	"golang.org/x/tools/go/packages"

	"github.com/loov/goda/internal/stat"
)

type Graph struct {
	Groups       map[string]*Group
	SortedGroups []*Group

	Packages map[string]*Node
	Sorted   []*Node
	stat.Stat
}

func (g *Graph) AddNode(n *Node) {
	g.Packages[n.ID] = n
	n.Graph = g
}

type Group struct {
	ID        string
	Collapsed bool
	Color     string
	Nodes     []*Node

	stat.Stat
}

func (g *Group) FirstModule() *packages.Module {
	for _, n := range g.Nodes {
		if n.Module != nil {
			return n.Module
		}
	}
	return nil
}

func (g *Group) AddNode(n *Node) {
	if n.Group != nil {
		n.Group.RemoveNode(n)
	}

	g.Nodes = append(g.Nodes, n)
	n.Group = g
	g.Stat.Add(n.Stat)
}

func (g *Group) RemoveNode(n *Node) {
	if n.Group != g {
		return
	}
	g.Stat.Sub(n.Stat)
	n.Group = nil

	i := slices.Index(g.Nodes, n)
	if i < 0 {
		return
	}
	g.Nodes = slices.Delete(g.Nodes, i, i+1)
}

func (g *Group) ImportsNodes() []*Node {
	xs := map[*Node]struct{}{}
	for _, n := range g.Nodes {
		for _, dep := range n.ImportsNodes {
			xs[dep] = struct{}{}
		}
	}
	nodes := slices.Collect(maps.Keys(xs))
	SortNodes(nodes)
	return nodes
}

type Node struct {
	*packages.Package
	Color string

	ImportsNodes []*Node

	// Stats about the current node.
	stat.Stat
	// Stats about upstream nodes.
	Up stat.Stat
	// Stats about downstream nodes.
	Down stat.Stat

	Errors []error
	Graph  *Graph
	Group  *Group // optional
}

func (n *Node) Pkg() *packages.Package { return n.Package }

// From creates a new graph from a map of packages.
func From(pkgs map[string]*packages.Package) *Graph {
	g := &Graph{
		Groups:   map[string]*Group{},
		Packages: map[string]*Node{},
	}

	// Create the graph nodes.
	for _, p := range pkgs {
		n := LoadNode(p)
		g.Sorted = append(g.Sorted, n)
		g.AddNode(n)
		g.Stat.Add(n.Stat)
	}
	SortNodes(g.Sorted)

	// TODO: find ways to improve performance.

	cache := allImportsCache(pkgs)

	// Populate the graph's Up and Down stats.
	for _, n := range g.Packages {
		importsIDs := cache[n.ID]
		for _, id := range importsIDs {
			imported, ok := g.Packages[id]
			if !ok {
				// we may not want to print info about every package
				continue
			}

			n.Down.Add(imported.Stat)
			imported.Up.Add(n.Stat)
		}
	}

	// Build node imports from package imports.
	for _, n := range g.Packages {
		for id := range n.Package.Imports {
			direct, ok := g.Packages[id]
			if !ok {
				// TODO:
				//  should we include dependencies where Y is hidden?
				//  X -> [Y] -> Z
				continue
			}

			n.ImportsNodes = append(n.ImportsNodes, direct)
		}
	}

	for _, n := range g.Packages {
		SortNodes(n.ImportsNodes)
	}

	return g
}

func (g *Graph) EnsureGroup(id string) *Group {
	k, ok := g.Groups[id]
	if !ok {
		k = &Group{
			ID: id,
		}
		g.Groups[id] = k
		g.SortedGroups = append(g.SortedGroups, k)
		sort.Slice(g.SortedGroups, func(i, k int) bool { return g.SortedGroups[i].ID < g.SortedGroups[k].ID })
	}
	return k
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

type flatNode struct {
	Package struct {
		ID              string
		Name            string            `json:",omitempty"`
		PkgPath         string            `json:",omitempty"`
		Errors          []packages.Error  `json:",omitempty"`
		GoFiles         []string          `json:",omitempty"`
		CompiledGoFiles []string          `json:",omitempty"`
		OtherFiles      []string          `json:",omitempty"`
		IgnoredFiles    []string          `json:",omitempty"`
		ExportFile      string            `json:",omitempty"`
		Imports         map[string]string `json:",omitempty"`
	}

	ImportsNodes []string `json:",omitempty"`

	Stat stat.Stat
	Up   stat.Stat
	Down stat.Stat

	Errors []error `json:",omitempty"`
}

func (p *Node) MarshalJSON() ([]byte, error) {
	flat := flatNode{
		Stat:   p.Stat,
		Up:     p.Up,
		Down:   p.Down,
		Errors: p.Errors,
	}

	flat.Package.ID = p.Package.ID
	flat.Package.Name = p.Package.Name
	flat.Package.PkgPath = p.Package.PkgPath
	flat.Package.GoFiles = p.Package.GoFiles
	flat.Package.CompiledGoFiles = p.Package.CompiledGoFiles
	flat.Package.OtherFiles = p.Package.OtherFiles
	flat.Package.IgnoredFiles = p.Package.IgnoredFiles
	flat.Package.ExportFile = p.Package.ExportFile

	for _, n := range p.ImportsNodes {
		flat.ImportsNodes = append(flat.ImportsNodes, n.ID)
	}
	if len(p.Package.Imports) > 0 {
		flat.Package.Imports = make(map[string]string, len(p.Imports))
		for path, ipkg := range p.Imports {
			flat.Package.Imports[path] = ipkg.ID
		}
	}

	return json.Marshal(flat)
}
