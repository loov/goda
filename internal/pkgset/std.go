package pkgset

import (
	"sync"

	"golang.org/x/tools/go/packages"
)

var stdpkgs Set
var stdonce sync.Once

// LoadStd preloads the std package list.
func LoadStd() {
	stdonce.Do(func() {
		standard, err := packages.Load(&packages.Config{
			Mode:  packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedModule,
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
	LoadStd()

	return IsStdName(p.ID)
}

// IsStdName returns whether id corresponds to a standard package
func IsStdName(id string) bool {
	LoadStd()

	_, ok := stdpkgs[id]
	return ok
}

// Std returns the standard package set
func Std() Set {
	LoadStd()

	return stdpkgs.Clone()
}
