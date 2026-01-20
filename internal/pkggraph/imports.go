package pkggraph

import (
	"slices"
	"sort"

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

		var xs []string
		for _, child := range p.Imports {
			xs = includePackageID(xs, child.ID)
			for _, pkg := range fetch(child) {
				xs = includePackageID(xs, pkg)
			}
		}
		cache[p.ID] = xs

		return xs
	}

	for _, p := range pkgs {
		_ = fetch(p)
	}

	return cache
}

func includePackageID(xs []string, p string) []string {
	if !hasPackageID(xs, p) {
		xs = append(xs, p)
		sort.Strings(xs)
	}
	return xs
}

func hasPackageID(xs []string, p string) bool {
	return slices.Contains(xs, p)
}
