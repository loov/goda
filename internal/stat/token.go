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

	// comment groups are reachable both via Doc fields and f.Comments,
	// so count them once from f.Comments and skip them during inspection.
	stat.Comment = int64(len(f.Comments))

	ast.Inspect(f, func(n ast.Node) bool {
		switch n.(type) {
		case nil:
		case *ast.BasicLit:
			stat.Basic++
		case *ast.CommentGroup:
			return false
		default:
			stat.Code++
		}
		return true
	})

	return stat
}
