package pkgset

import (
	"sort"

	"golang.org/x/tools/go/packages"
)

// Set is a p.ID -> *packages.Package
type Set map[string]*packages.Package

// New makes a set from roots recursively
func New(roots ...*packages.Package) Set {
	set := make(Set, len(roots))
	for _, p := range roots {
		set.IncludeRecursive(p)
	}
	return set
}

// Sorted returns packages in sorted order
func (set Set) Sorted() []*packages.Package {
	var list []*packages.Package
	for _, pkg := range set {
		list = append(list, pkg)
	}
	sort.Slice(list, func(i, k int) bool {
		return list[i].ID < list[k].ID
	})
	return list
}

// Tree returns package tree
func (set Set) Tree() *Tree {
	tree := NewTree(nil, "")
	for _, pkg := range set {
		tree.Add(pkg)
	}
	return tree
}

// IncludeRecursive adds p recursively
func (set Set) IncludeRecursive(p *packages.Package) {
	if _, added := set[p.ID]; added {
		return
	}
	set[p.ID] = p

	for _, imp := range p.Imports {
		set.IncludeRecursive(imp)
	}
}

// Clone makes a copy of the set
func (set Set) Clone() Set {
	r := make(Set, len(set))
	for pid, p := range set {
		r[pid] = p
	}
	return r
}

// Union includes packages from both sets
func Union(a, b Set) Set {
	if len(a) == 0 {
		return b.Clone()
	}

	r := a.Clone()
	for pid, p := range b {
		if _, exists := r[pid]; !exists {
			r[pid] = p
		}
	}
	return r
}

// Subtract returns packages that exist in a, but not in b
func Subtract(a, b Set) Set {
	r := a.Clone()
	for pid := range b {
		delete(r, pid)
	}
	return r
}

// Intersect returns packages that exist in both
func Intersect(a, b Set) Set {
	r := make(Set, len(a))
	for pid := range b {
		if p, ok := a[pid]; ok {
			r[pid] = p
		}
	}
	return r
}

// SymmetricDifference returns packages that are different
func SymmetricDifference(a, b Set) Set {
	r := make(Set, len(a))
	for pid, p := range a {
		if _, ok := b[pid]; !ok {
			r[pid] = p
		}
	}

	for pid, p := range b {
		if _, ok := a[pid]; !ok {
			r[pid] = p
		}
	}
	return r
}
