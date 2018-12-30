package ast

import (
	"reflect"
	"testing"
)

func TestParsing(t *testing.T) {
	tests := []struct {
		input  string
		clean  string
		tokens []Token
	}{{
		"", "", nil,
	}, {
		"golang.org/x/tools/...",
		"golang.org/x/tools/...",
		[]Token{
			{TPackage, "golang.org/x/tools/..."},
		},
	}, {
		"  github.com/loov/goda    golang.org/x/tools/...  ",
		"(github.com/loov/goda, golang.org/x/tools/...)",
		[]Token{
			{TPackage, "github.com/loov/goda"},
			{TPackage, "golang.org/x/tools/..."},
		},
	}, {
		"  github.com/loov/goda  +  golang.org/x/tools/...  ",
		"+(github.com/loov/goda, golang.org/x/tools/...)",
		[]Token{
			{TPackage, "github.com/loov/goda"},
			{TOp, "+"},
			{TPackage, "golang.org/x/tools/..."},
		},
	}, {
		"  github.com/loov/goda:root - golang.org/x/tools/...  ",
		"-(github.com/loov/goda:root, golang.org/x/tools/...)",
		[]Token{
			{TPackage, "github.com/loov/goda"},
			{TSelector, "root"},
			{TOp, "-"},
			{TPackage, "golang.org/x/tools/..."},
		},
	}, {
		"Reaches(github.com/loov/goda +   github.com/loov/qloc, golang.org/x/tools/...:root)",
		"Reaches(+(github.com/loov/goda, github.com/loov/qloc), golang.org/x/tools/...:root)",
		[]Token{
			{TFunc, "Reaches"},
			{TLeftParen, "("},
			{TPackage, "github.com/loov/goda"},
			{TOp, "+"},
			{TPackage, "github.com/loov/qloc"},
			{TComma, ","},
			{TPackage, "golang.org/x/tools/..."},
			{TSelector, "root"},
			{TRightParen, ")"},
		},
	}, {
		"Reaches(github.com/loov/goda, golang.org/x/tools/...:root):deps:root",
		"Reaches(github.com/loov/goda, golang.org/x/tools/...:root):deps:root",
		[]Token{
			{TFunc, "Reaches"},
			{TLeftParen, "("},
			{TPackage, "github.com/loov/goda"},
			{TComma, ","},
			{TPackage, "golang.org/x/tools/..."},
			{TSelector, "root"},
			{TRightParen, ")"},
			{TSelector, "deps"},
			{TSelector, "root"},
		},
	}}

	for _, test := range tests {
		tokens, err := Tokenize(test.input)
		if err != nil {
			t.Errorf("\nlex %q\n\t%v\n\t%v", test.input, tokens, err)
			continue
		}
		if len(tokens) == 0 {
			tokens = nil
		}

		if !reflect.DeepEqual(tokens, test.tokens) {
			t.Errorf("\nlex %q\n\t%v\n\t%v", test.input, test.tokens, tokens)
			continue
		}

		expr, err := Parse(tokens)
		if expr == nil {
			continue
		}
		if err != nil {
			t.Errorf("\nparse %q\n\t%v", test.input, err)
			continue
		}

		clean := expr.String()
		if clean != test.clean {
			t.Errorf("\nparse %q\n\t%v\n\t%v", test.input, test.clean, clean)
			continue
		}
	}
}
