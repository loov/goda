package pkg

import (
	"sync"

	"golang.org/x/tools/go/packages"
)

var stdpkgs Set
var stdonce sync.Once

func loadstd() {
	stdonce.Do(func() {
		standard, err := packages.Load(&packages.Config{
			Mode:  packages.LoadImports,
			Tests: false,
		}, "std")

		if err != nil {
			panic(err)
		}

		stdpkgs = NewSet(standard...)
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
