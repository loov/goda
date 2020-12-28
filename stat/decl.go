package stat

import (
	"go/ast"
	"go/token"
)

// TopDecl stats about top-level declarations.
type TopDecl struct {
	Func  int64
	Type  int64
	Const int64
	Var   int64
	Other int64
}

func (s *TopDecl) Add(b TopDecl) {
	s.Func += b.Func
	s.Type += b.Type
	s.Const += b.Const
	s.Var += b.Var
	s.Other += b.Other
}

func (s *TopDecl) Total() int64 {
	return s.Func + s.Type + s.Const + s.Var + s.Other
}

func TopDeclFromAst(f *ast.File) TopDecl {
	stat := TopDecl{}
	for _, decl := range f.Decls {
		switch decl := decl.(type) {
		case *ast.GenDecl:
			switch decl.Tok {
			case token.TYPE:
				stat.Type++
			case token.VAR:
				stat.Var++
			case token.CONST:
				stat.Const++
			default:
				stat.Other++
			}
		case *ast.FuncDecl:
			stat.Func++
		default:
			stat.Other++
		}
	}
	return stat
}
