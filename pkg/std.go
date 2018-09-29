package pkg

import (
	"fmt"

	"golang.org/x/tools/go/packages"
)

var stdpkgs Set

func init() {
	standard, err := packages.Load(&packages.Config{
		Mode:  packages.LoadImports,
		Tests: false,
	}, "std")

	if err != nil {
		panic(err)
	}

	stdpkgs = NewSet(standard...)

	for pkg := range stdpkgs {
		fmt.Println(pkg)
	}
}

func IsStd(p *packages.Package) bool {
	return IsStdName(p.ID)
}

func IsStdName(id string) bool {
	_, ok := stdpkgs[id]
	return ok
}
