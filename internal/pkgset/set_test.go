package pkgset

import (
	"errors"
	"slices"
	"strings"
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

func TestBroadPatterns(t *testing.T) {
	got := broadPatterns([]string{"./...", "github.com/...", "golang.org/x/...", "fmt", "example.com/m/..."})
	want := []string{"github.com/...", "golang.org/x/...", "example.com/m/..."}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExplainChecksumError(t *testing.T) {
	tidy := errors.New("go: updates to go.mod needed; to update it:\n\tgo mod tidy")
	if got := explainChecksumError(tidy, []string{"github.com/..."}); !strings.Contains(got.Error(), `pattern "github.com/..."`) {
		t.Errorf("broad pattern hint missing: %v", got)
	}
	if got := explainChecksumError(tidy, []string{"github.com/x/y"}); !strings.Contains(got.Error(), "outside this module's dependency graph") {
		t.Errorf("explicit package hint missing: %v", got)
	}
	sum := errors.New("pattern periph.io/x/...: periph.io/x/d2xx@v0.1.1: missing go.sum entry")
	if got := explainChecksumError(sum, []string{"periph.io/x/..."}); !strings.Contains(got.Error(), `pattern "periph.io/x/..."`) {
		t.Errorf("go.sum hint missing: %v", got)
	}
	other := errors.New("package nope/x is not in std")
	if got := explainChecksumError(other, []string{"nope/x"}); got != other {
		t.Errorf("unrelated error was wrapped: %v", got)
	}
	if got := explainChecksumError(nil, nil); got != nil {
		t.Errorf("nil error became %v", got)
	}
}
