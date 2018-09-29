package pkg

import "golang.org/x/tools/go/packages"

// Set is a p.ID -> *packages.Package
type Set map[string]*packages.Package

// NewSet makes a set from roots recursively
func NewSet(roots ...*packages.Package) Set {
	set := make(Set, len(roots))
	for _, p := range roots {
		set.IncludeRecursive(p)
	}
	return set
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
	for pid, _ := range b {
		delete(r, pid)
	}
	return r
}

// Interesct returns packages that exist in both
func Intersect(a, b Set) Set {
	r := make(Set, len(a))
	for pid, _ := range b {
		if p, ok := a[pid]; ok {
			r[pid] = p
		}
	}
	return r
}
