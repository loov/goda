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
		"std - (std - unsafe:all)",
		"-(std, -(std, unsafe:all))",
		[]Token{
			{TPackage, "std"},
			{TOp, "-"},
			{TLeftParen, "("},
			{TPackage, "std"},
			{TOp, "-"},
			{TPackage, "unsafe"},
			{TSelector, "all"},
			{TRightParen, ")"},
		},
	}, {
		"  github.com/loov/goda:all - golang.org/x/tools/...  ",
		"-(github.com/loov/goda:all, golang.org/x/tools/...)",
		[]Token{
			{TPackage, "github.com/loov/goda"},
			{TSelector, "all"},
			{TOp, "-"},
			{TPackage, "golang.org/x/tools/..."},
		},
	}, {
		"Reaches(github.com/loov/goda +   github.com/loov/qloc, golang.org/x/tools/...:all)",
		"Reaches(+(github.com/loov/goda, github.com/loov/qloc), golang.org/x/tools/...:all)",
		[]Token{
			{TFunc, "Reaches"},
			{TLeftParen, "("},
			{TPackage, "github.com/loov/goda"},
			{TOp, "+"},
			{TPackage, "github.com/loov/qloc"},
			{TComma, ","},
			{TPackage, "golang.org/x/tools/..."},
			{TSelector, "all"},
			{TRightParen, ")"},
		},
	}, {
		"Reaches(github.com/loov/goda, golang.org/x/tools/...:all):deps:all",
		"Reaches(github.com/loov/goda, golang.org/x/tools/...:all):deps:all",
		[]Token{
			{TFunc, "Reaches"},
			{TLeftParen, "("},
			{TPackage, "github.com/loov/goda"},
			{TComma, ","},
			{TPackage, "golang.org/x/tools/..."},
			{TSelector, "all"},
			{TRightParen, ")"},
			{TSelector, "deps"},
			{TSelector, "all"},
		},
	}, {
		"test=1(github.com/loov/goda)",
		"test=1(github.com/loov/goda)",
		[]Token{
			{TFunc, "test=1"},
			{TLeftParen, "("},
			{TPackage, "github.com/loov/goda"},
			{TRightParen, ")"},
		},
	}, {
		"test=1(github.com/loov/goda) - test=0(github.com/loov/goda)",
		"-(test=1(github.com/loov/goda), test=0(github.com/loov/goda))",
		[]Token{
			{TFunc, "test=1"},
			{TLeftParen, "("},
			{TPackage, "github.com/loov/goda"},
			{TRightParen, ")"},
			{TOp, "-"},
			{TFunc, "test=0"},
			{TLeftParen, "("},
			{TPackage, "github.com/loov/goda"},
			{TRightParen, ")"},
		},
	}}

	for _, test := range tests {
		tokens, err := Tokenize(test.input)
		if err != nil {
			t.Errorf("\nlex %q\n\tgot:%v\n\terr:%v", test.input, tokens, err)
			continue
		}
		if len(tokens) == 0 {
			tokens = nil
		}

		if !reflect.DeepEqual(tokens, test.tokens) {
			t.Errorf("\nlex %q\n\texp:%v\n\tgot:%v", test.input, test.tokens, tokens)
			continue
		}

		expr, err := Parse(tokens)
		if err != nil {
			t.Errorf("\nparse %q\n\terr:%v", test.input, err)
			continue
		}
		if expr == nil {
			continue
		}

		clean := expr.String()
		if clean != test.clean {
			t.Errorf("\nparse %q\n\texp:%v\n\tgot:%v", test.input, test.clean, clean)
			t.Log("\nTREE\n", expr.Tree(0))
			continue
		}
	}
}
