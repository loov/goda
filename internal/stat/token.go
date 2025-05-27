package stat

import "go/ast"

type Tokens struct {
	Code    int64
	Comment int64
	Basic   int64
}

func (stat *Tokens) Add(b Tokens) {
	stat.Code += b.Code
	stat.Comment += b.Comment
	stat.Basic += b.Basic
}

func (stat *Tokens) Sub(b Tokens) {
	stat.Code -= b.Code
	stat.Comment -= b.Comment
	stat.Basic -= b.Basic
}

func TokensFromAst(f *ast.File) Tokens {
	stat := Tokens{}

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		switch n.(type) {
		default:
			stat.Code++
		case *ast.BasicLit:
			stat.Basic++
		case *ast.CommentGroup, *ast.Comment:
			stat.Comment++
			return false
		}

		return true
	})

	return stat
}
