package pkggraph

import (
	"path"
	"sort"
	"strings"
)

type Tree struct {
	Path    string
	Package *Node

	Child map[string]*Tree

	Parent   *Tree
	Children []*Tree

	strat Strategy
}

type Strategy interface {
	Layers(pkg *Node) []string
}

type PathStrategy struct{}

func (s PathStrategy) Layers(pkg *Node) []string {
	return strings.Split(pkg.PkgPath, "/")
}

type RepoStrategy struct{}

func (s RepoStrategy) Layers(pkg *Node) []string {
	if pkg.Repo == nil {
		return strings.Split(pkg.PkgPath, "/")
	}

	ls := []string{pkg.Repo.Root}

	suffix := strings.TrimPrefix(strings.TrimPrefix(pkg.PkgPath, pkg.Repo.Root), "/")
	if suffix != "" {
		ls = append(ls, suffix)
	}

	return ls
}

// Tree returns Node tree.
func (graph *Graph) Tree(strat Strategy) *Tree {
	tree := NewTree(nil, "", strat)
	for _, pkg := range graph.Sorted {
		tree.Add(pkg)
	}
	tree.Sort()
	return tree
}

func NewTree(parent *Tree, path string, strat Strategy) *Tree {
	return &Tree{
		Path:   path,
		Child:  map[string]*Tree{},
		Parent: parent,
		strat:  strat,
	}
}

func (tree *Tree) Add(pkg *Node) {
	layers := tree.strat.Layers(pkg)
	tree.Insert([]string{}, layers, pkg)
}

func (tree *Tree) Insert(prefix, suffix []string, pkg *Node) {
	if len(suffix) == 0 {
		tree.Package = pkg
		return
	}

	childPrefix := append(prefix, suffix[0])
	child, hasChild := tree.Child[suffix[0]]
	if !hasChild {
		child = NewTree(tree, path.Join(childPrefix...), tree.strat)
		tree.Child[suffix[0]] = child
		tree.Children = append(tree.Children, child)
	}

	child.Insert(childPrefix, suffix[1:], pkg)
}

func (tree *Tree) Walk(fn func(tree *Tree)) {
	fn(tree)
	for _, child := range tree.Children {
		child.Walk(fn)
	}
}

func (tree *Tree) HasParent(parent *Tree) bool {
	return strings.HasPrefix(tree.Path, parent.Path+"/")
}

func (tree *Tree) LookupTable() map[*Node]*Tree {
	table := map[*Node]*Tree{}
	tree.Walk(func(x *Tree) {
		if x.Package != nil {
			table[x.Package] = x
		}
	})
	return table
}

func (tree *Tree) Sort() {
	tree.Walk(func(x *Tree) {
		sort.Slice(x.Children, func(i, k int) bool {
			left, right := x.Children[i], x.Children[k]
			return left.Path < right.Path
		})
	})
}
