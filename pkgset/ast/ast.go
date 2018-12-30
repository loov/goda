package ast

import (
	"errors"
	"strings"
)

type Expr interface {
	String() string
}

type Package string

type Select struct {
	Expr     Expr
	Selector string
}

type Func struct {
	Name string
	Args []Expr
}

func (p Package) String() string { return string(p) }

func (s Select) String() string { return s.Expr.String() + ":" + s.Selector }

func (f Func) String() string {
	var args []string
	for _, arg := range f.Args {
		args = append(args, arg.String())
	}
	return f.Name + "(" + strings.Join(args, ", ") + ")"
}

func Parse(tokens []Token) (Expr, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	p, expr, err := parseCombine(0, tokens)
	if err != nil {
		return expr, err
	}
	if p != len(tokens) {
		panic("failed to parse")
	}
	return expr, nil
}

func parseCombine(p int, tokens []Token) (int, Expr, error) {
	var err error
	if len(tokens) == 0 {
		return p, nil, nil
	}

	var exprs []Expr

	for p < len(tokens) {
		tok := tokens[p]

		var expr Expr
		switch tok.Kind {
		case TPackage:
			p++
			expr = Package(tok.Text)

		case TFunc, TLeftParen:
			if tok.Kind == TFunc { // position to the left paren
				p++
			}

			if p >= len(tokens) || tokens[p].Kind != TLeftParen {
				panic("unexpected func location")
			}
			p++ // skip the left paren

			funcexpr := Func{tok.Text, nil}
			if tok.Kind == TLeftParen {
				funcexpr.Name = ""
			}

			for {
				var arg Expr
				p, arg, err = parseCombine(p, tokens)
				if err != nil {
					return p, combine(exprs), err
				}
				funcexpr.Args = append(funcexpr.Args, arg)
				if tokens[p-1].Kind != TComma {
					break
				}
			}
			expr = funcexpr

		case TOp:
			var right Expr
			p, right, err = parseCombine(p+1, tokens)
			if err != nil {
				return p, combine(exprs), err
			}
			if right == nil {
				return p, combine(exprs), errors.New("expected expression for operator")
			}
			return p, Func{tok.Text, []Expr{
				combine(exprs),
				right,
			}}, err

		case TSelector:
			p++
			return p, nil, errors.New("unexpected selector")

		case TRightParen, TComma:
			p++
			return p, combine(exprs), nil
		}

		for p < len(tokens) && tokens[p].Kind == TSelector {
			expr = Select{expr, tokens[p].Text}
			p++
		}

		exprs = append(exprs, expr)
	}

	return p, combine(exprs), nil
}

func combine(exprs []Expr) Expr {
	if len(exprs) == 0 {
		return nil
	}
	if len(exprs) == 1 {
		return exprs[0]
	}
	return Func{"", exprs}
}
