package pkgset

import (
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestSources(t *testing.T) {
	pkg := func(id string, imports ...*packages.Package) *packages.Package {
		p := &packages.Package{
			ID:      id,
			PkgPath: id,
			Imports: map[string]*packages.Package{},
		}
		for _, dep := range imports {
			p.Imports[dep.PkgPath] = dep
		}
		return p
	}

	c := pkg("c")
	b := pkg("b", c)
	a := pkg("a", b)
	// x imports b, but is not part of the set below.
	_ = pkg("x", b)

	set := Set{"a": a, "b": b, "c": c}

	sources := Sources(set)
	if len(sources) != 1 {
		t.Errorf("expected 1 source, got %v", sources.IDs())
	}
	if _, ok := sources["a"]; !ok {
		t.Errorf("expected %q as source, got %v", "a", sources.IDs())
	}
}
