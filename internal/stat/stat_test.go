package stat

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestSourceFromBytes(t *testing.T) {
	tests := []struct {
		in   string
		want Source
	}{
		{"", Source{Files: 1}},
		{"a\n\nb", Source{Files: 1, Size: 4, Lines: 2, Blank: 1}},
		{"a\n \t\r\n", Source{Files: 1, Size: 6, Lines: 1, Blank: 1}},
		{"x\x00y", Source{Files: 0, Binary: 1, Size: 3}},
	}
	for _, test := range tests {
		if got := SourceFromBytes([]byte(test.in)); got != test.want {
			t.Errorf("%q: got %+v, want %+v", test.in, got, test.want)
		}
	}
}

func TestTokensFromAst(t *testing.T) {
	const src = `// Package doc.
package x

// F doc.
func F() int { return 1 /* inline */ }
`
	f, err := parser.ParseFile(token.NewFileSet(), "x.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	got := TokensFromAst(f)
	if got.Comment != 3 {
		t.Errorf("comment groups: got %d, want 3", got.Comment)
	}
	if got.Basic != 1 {
		t.Errorf("basic literals: got %d, want 1", got.Basic)
	}
	if got.Code == 0 {
		t.Errorf("expected code tokens")
	}
}
