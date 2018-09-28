package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"text/tabwriter"

	"golang.org/x/tools/go/packages"
)

func main() {
	flag.Parse()
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.LoadImports,
		Env:  os.Environ(),
	}, flag.Args()...)

	if err != nil {
		panic(err)
	}

	ctx := NewContext()
	ctx.VisitRoots(pkgs)

	for _, pkg := range ctx.List {
		for _, imp := range pkg.Imports {
			imp.InDegree++
		}
	}

	roots := append([]*Package{}, ctx.List...)
	for _, root := range roots {
		Reset(ctx.List)
		root.DirectCost = Erase(root)
	}

	sort.Slice(roots, func(i, k int) bool {
		if roots[i].InDegree == roots[k].InDegree {
			return roots[i].DirectCost > roots[k].DirectCost
		}
		return roots[i].InDegree < roots[k].InDegree
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	defer w.Flush()
	for _, root := range roots {
		if root.DirectCost == 0 {
			continue
		}
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", root.ID, root.InDegree, root.DirectCost, names(root.ImportedBy))
	}

	fmt.Println("Dependencies: ", len(ctx.List))
}

func Reset(pkgs []*Package) {
	for _, pkg := range pkgs {
		pkg.InErased = pkg.InDegree
	}
}

func Erase(pkg *Package) int {
	totalRemoved := 0
	for _, imp := range pkg.Imports {
		imp.InErased--
		if imp.InErased == 0 {
			totalRemoved += 1
			totalRemoved += Erase(imp)
		}
	}
	return totalRemoved
}

func names(pkgs []*Package) []string {
	xs := []string{}
	for _, pkg := range pkgs {
		xs = append(xs, pkg.ID)
	}
	return xs
}

type Context struct {
	GOROOT string

	Ignored map[*packages.Package]bool
	Seen    map[*packages.Package]bool
	List    []*Package

	ByID map[string]*Package
}

type Package struct {
	ID string

	InDegree int
	InErased int

	DirectCost int

	ImportedBy []*Package
	Imports    []*Package
}

func NewContext() *Context {
	return &Context{
		GOROOT:  runtime.GOROOT(),
		Ignored: make(map[*packages.Package]bool),
		Seen:    make(map[*packages.Package]bool),

		ByID: make(map[string]*Package),
	}
}

func (ctx *Context) VisitRoots(roots []*packages.Package) {
	for _, root := range roots {
		ctx.Visit(nil, root)
	}
}

func (ctx *Context) Visit(parent *Package, p *packages.Package) {
	pkg := ctx.Find(p.ID)
	if parent != nil {
		parent.Imports = append(parent.Imports, pkg)
		pkg.ImportedBy = append(pkg.ImportedBy, parent)
	}
	if ctx.Seen[p] {
		return
	}
	ctx.Seen[p] = true

	// recurse
	for _, child := range p.Imports {
		if ctx.Ignore(child) {
			continue
		}
		ctx.Visit(pkg, child)
	}
}

func (ctx *Context) Find(id string) *Package {
	if pkg, ok := ctx.ByID[id]; ok {
		return pkg
	}

	pkg := &Package{ID: id}
	ctx.ByID[id] = pkg

	i := sort.Search(len(ctx.List), func(k int) bool {
		return ctx.List[k].ID > id
	})
	ctx.List = append(ctx.List, pkg)
	copy(ctx.List[i+1:], ctx.List[i:])
	ctx.List[i] = pkg

	return pkg
}

func (ctx *Context) Ignore(p *packages.Package) bool {
	if ctx.Ignored[p] {
		return true
	}

	// ignore standard library
	if len(p.GoFiles) == 0 {
		ctx.Ignored[p] = true
		return true
	}
	if filepath.HasPrefix(p.GoFiles[0], ctx.GOROOT) {
		ctx.Ignored[p] = true
		return true
	}

	// ignore golang.org/x
	if strings.HasPrefix(p.ID, "golang.org/x") {
		ctx.Ignored[p] = true
		return true
	}

	return false
}
