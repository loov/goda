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

// NewRoot makes a set from roots recursively
func NewRoot(roots ...*packages.Package) Set {
	set := make(Set, len(roots))
	for _, p := range roots {
		set[p.ID] = p
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
	tree.Sort()
	return tree
}

func (set Set) Walk(fn func(*packages.Package)) {
	for _, p := range set {
		fn(p)
	}
}

func (set Set) WalkDependencies(fn func(*packages.Package)) {
	seen := map[string]bool{}
	var walk func(*packages.Package)
	walk = func(p *packages.Package) {
		if seen[p.ID] {
			return
		}
		seen[p.ID] = true
		fn(p)
		for _, child := range p.Imports {
			walk(child)
		}
	}

	for _, p := range set {
		walk(p)
	}
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
		r[pid] = p // TODO: make a deep clone
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

// Reach returns packages in a that terminate in b
func Reach(a, b Set) Set {
	result := New()
	reaches := b.Clone()
	cannotReach := New()

	var checkReachability func(p *packages.Package) bool
	checkReachability = func(p *packages.Package) bool {
		if _, ok := reaches[p.ID]; ok {
			return true
		}
		if _, ok := cannotReach[p.ID]; ok {
			return false
		}

		for _, dep := range p.Imports {
			if checkReachability(dep) {
				if _, ina := a[p.ID]; ina {
					result[p.ID] = p
				}
				reaches[p.ID] = p
				return true
			}
		}

		cannotReach[p.ID] = p
		return false
	}

	for _, p := range a {
		if _, reaches := b[p.ID]; reaches {
			result[p.ID] = p
			continue
		}
		checkReachability(p)
	}

	return result
}

// Transitive returns transitive reduction.
func Transitive(a Set) Set {
	result := a.Clone()

	var includeDeps func(p *packages.Package, r map[string]struct{})
	includeDeps = func(p *packages.Package, r map[string]struct{}) {
		for _, c := range p.Imports {
			if _, visited := r[c.ID]; visited {
				continue
			}
			r[c.ID] = struct{}{}
			includeDeps(c, r)
		}
	}

	for _, p := range result {
		indirectDeps := make(map[string]struct{})
		for _, c := range p.Imports {
			includeDeps(c, indirectDeps)
		}
		for dep := range indirectDeps {
			delete(p.Imports, dep)
		}
	}

	return result
}

// Sources returns packages that don't have incoming edges
func Sources(a Set) Set {
	incoming := map[string]int{}

	a.WalkDependencies(func(p *packages.Package) {
		for _, dep := range p.Imports {
			incoming[dep.ID]++
		}
	})

	result := New()
	for _, p := range a {
		if incoming[p.ID] == 0 {
			result[p.ID] = p
		}
	}

	return result
}

// Dependencies returns packages that has removed first layer in the DAG
func Dependencies(a Set) Set {
	all := map[string]*packages.Package{}
	incoming := map[string]int{}

	a.WalkDependencies(func(p *packages.Package) {
		all[p.ID] = p
		for _, dep := range p.Imports {
			incoming[dep.ID]++
		}
	})

	result := New()
	for pid, count := range incoming {
		if count > 0 {
			result[pid] = all[pid]
		}
	}
	return result
}
