package pkgset

import (
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Set is a p.ID -> *packages.Package
type Set map[string]*packages.Package

// New makes a set from roots recursively
func New(roots ...*packages.Package) Set {
	set := make(Set)
	for _, p := range roots {
		set.IncludeRecursive(p)
	}
	return set
}

// NewRoot makes a set from roots recursively
func NewRoot(roots ...*packages.Package) Set {
	set := make(Set)
	for _, p := range roots {
		set[p.ID] = p
	}
	return set
}

// NewAll includes all the dependencies from the graph.
func NewAll(src Set) Set {
	set := make(Set)
	for _, p := range src {
		set.IncludeRecursive(p)
	}
	return set
}

// List returns packages in unsorted order.
func (set Set) List() []*packages.Package {
	dst := make([]*packages.Package, 0, len(set))
	for _, p := range set {
		dst = append(dst, p)
	}
	return dst
}

// IDs returns package ID-s in sorted order.
func (set Set) IDs() []string {
	rs := []string{}
	for id := range set {
		rs = append(rs, id)
	}
	sort.Strings(rs)
	return rs
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

// UnionAll includes packages from both sets
func UnionAll(xs ...Set) Set {
	if len(xs) == 0 {
		return Set{}
	}

	r := xs[0].Clone()
	for _, b := range xs[1:] {
		for pid, p := range b {
			if _, exists := r[pid]; !exists {
				r[pid] = p
			}
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

// Incoming returns packages from a that directly import packages in b.
func Incoming(a, b Set) Set {
	result := b.Clone()
next:
	for _, x := range a {
		for _, imp := range x.Imports {
			if _, ok := b[imp.ID]; ok {
				result[x.ID] = x
				continue next
			}
		}
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

// DirectDependencies returns packages that are direct dependencies of a, `a` not included.
func DirectDependencies(a Set) Set {
	rs := map[string]*packages.Package{}
	for _, p := range a {
		for _, dep := range p.Imports {
			if _, ok := a[dep.ID]; ok {
				continue
			}
			rs[dep.ID] = dep
		}
	}
	return rs
}

// ModuleDependencies returns packages that are direct or indirect dependencies of a,
// which are part of modules of package a.
func ModuleDependencies(a Set) Set {
	modules := map[string]struct{}{}
	for _, p := range a {
		if p.Module != nil {
			modules[p.Module.Path] = struct{}{}
		}
	}

	rs := a.Clone()
	a.WalkDependencies(func(p *packages.Package) {
		if p.Module != nil {
			if _, ok := modules[p.Module.Path]; ok {
				rs[p.ID] = p
			}
		}
	})

	return rs
}

// Main returns main pacakges.
func Main(a Set) Set {
	rs := Set{}
	for pid, pkg := range a {
		if pkg.Name == "main" {
			rs[pid] = pkg
		}
	}
	return rs
}

// Test returns test packages from set.
func Test(a Set) Set {
	rs := Set{}
	for pid, pkg := range a {
		if IsTestPkg(pkg) {
			rs[pid] = pkg
		}
	}
	return rs
}

func IsTestPkg(pkg *packages.Package) bool {
	return strings.HasSuffix(pkg.ID, ".test") ||
		strings.HasSuffix(pkg.ID, "_test") ||
		strings.HasSuffix(pkg.ID, ".test]")
}
