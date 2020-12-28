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
}

// Tree returns Node tree.
func (graph *Graph) Tree() *Tree {
	tree := NewTree(nil, "")
	for _, pkg := range graph.Sorted {
		tree.Add(pkg)
	}
	tree.Sort()
	return tree
}

func NewTree(parent *Tree, path string) *Tree {
	return &Tree{
		Path:   path,
		Child:  map[string]*Tree{},
		Parent: parent,
	}
}

func (tree *Tree) Add(pkg *Node) {
	tree.Insert([]string{}, strings.Split(pkg.PkgPath, "/"), pkg)
}

func (tree *Tree) Insert(prefix, suffix []string, pkg *Node) {
	if len(suffix) == 0 {
		tree.Package = pkg
		return
	}

	childPrefix := append(prefix, suffix[0])
	child, hasChild := tree.Child[suffix[0]]
	if !hasChild {
		child = NewTree(tree, path.Join(childPrefix...))
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
