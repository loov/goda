package pkggraph

import (
	"maps"
	"slices"

	"golang.org/x/tools/go/packages"
)

func allImportsCache(pkgs map[string]*packages.Package) map[string][]string {
	cache := map[string][]string{}

	var fetch func(p *packages.Package) []string
	fetch = func(p *packages.Package) []string {
		if n, ok := cache[p.ID]; ok {
			return n
		}

		// prevent cycles
		cache[p.ID] = []string{}

		set := map[string]struct{}{}
		for _, child := range p.Imports {
			set[child.ID] = struct{}{}
			for _, pkg := range fetch(child) {
				set[pkg] = struct{}{}
			}
		}
		xs := slices.Sorted(maps.Keys(set))
		cache[p.ID] = xs

		return xs
	}

	for _, p := range pkgs {
		_ = fetch(p)
	}

	return cache
}
