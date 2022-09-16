package pkggraph

import (
	"encoding/json"
	"sort"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/vcs"

	"github.com/loov/goda/internal/stat"
)

type Graph struct {
	Packages map[string]*Node
	Sorted   []*Node
	stat.Stat
}

func (g *Graph) AddNode(n *Node) {
	g.Packages[n.ID] = n
	n.Graph = g
}

type Node struct {
	*packages.Package
	Repo *vcs.RepoRoot

	ImportsNodes []*Node

	// Stats about the current node.
	stat.Stat
	// Stats about upstream nodes.
	Up stat.Stat
	// Stats about downstream nodes.
	Down stat.Stat

	Errors []error
	Graph  *Graph
}

func (n *Node) Pkg() *packages.Package { return n.Package }

// From creates a new graph from a map of packages.
func From(pkgs map[string]*packages.Package) *Graph {
	g := &Graph{Packages: map[string]*Node{}}

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

func LoadNode(p *packages.Package) *Node {
	node := &Node{}
	node.Package = p

	if repo, err := vcs.RepoRootForImportPath(p.PkgPath, false); err != nil {
		node.Errors = append(node.Errors, err)
		node.Repo = &vcs.RepoRoot{
			VCS:  &vcs.Cmd{},
			Repo: p.PkgPath, // maybe it's possible to use `PkgPath` here instead or find one from `p.Module.???`
			Root: p.PkgPath,
		}
	} else {
		node.Repo = repo
	}

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
