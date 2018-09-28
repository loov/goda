package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	std := flag.Bool("std", false, "include std lib in dependencies")
	flag.Parse()

	pkgNames := flag.Args()
	if len(pkgNames) == 0 {
		pkgNames = []string{"."}
	}

	roots, err := packages.Load(&packages.Config{
		Mode: packages.LoadImports,
		Env:  os.Environ(),
	}, pkgNames...)

	if err != nil {
		panic(err)
	}

	seen := map[*packages.Package]bool{}

	var visit func(int, *packages.Package, bool)
	visit = func(ident int, p *packages.Package, last bool) {
		if last {
			fmt.Print(strings.Repeat("  ", ident), "  └ ", p.ID)
		} else {
			fmt.Print(strings.Repeat("  ", ident), "  ├ ", p.ID)
		}

		if seen[p] || isStd(p) {
			fmt.Println(" ~")
			return
		}
		fmt.Println()

		seen[p] = true
		keys := []string{}
		for id, pkg := range p.Imports {
			if !*std && isStd(pkg) {
				continue
			}
			keys = append(keys, id)
		}

		sort.Strings(keys)
		for i, id := range keys {
			pkg := p.Imports[id]
			visit(ident+1, pkg, i == len(keys)-1)
		}
	}

	for _, pkg := range roots {
		visit(0, pkg, false)
	}
}

var root = runtime.GOROOT()
var stdlib = map[*packages.Package]bool{}

func isStd(p *packages.Package) bool {
	if isstd, ok := stdlib[p]; ok {
		return isstd
	}

	if len(p.GoFiles) == 0 {
		stdlib[p] = true
		return true
	}
	if filepath.HasPrefix(p.GoFiles[0], root) {
		stdlib[p] = true
		return true
	}

	stdlib[p] = false
	return false
}
