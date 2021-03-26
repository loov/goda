package pkgset

import (
	"sync"

	"golang.org/x/tools/go/packages"
)

var stdpkgs Set
var stdonce sync.Once

func loadstd() {
	stdonce.Do(func() {
		standard, err := packages.Load(&packages.Config{
			Mode:  packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports,
			Tests: true,
		}, "std")

		if err != nil {
			panic(err)
		}

		stdpkgs = New(standard...)
	})
}

// IsStd returns whether *packages.Package is a std package
func IsStd(p *packages.Package) bool {
	loadstd()

	return IsStdName(p.ID)
}

// IsStdName returns whether id corresponds to a standard package
func IsStdName(id string) bool {
	loadstd()

	_, ok := stdpkgs[id]
	return ok
}

// Std returns the standard package set
func Std() Set {
	loadstd()

	return stdpkgs.Clone()
}
