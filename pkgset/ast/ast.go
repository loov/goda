package ast

import (
	"errors"
	"fmt"
	"strings"
)

type Expr interface {
	String() string
	Tree(ident int) string
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

func (p Package) Tree(ident int) string { return strings.Repeat("  ", ident) + string(p) + "\n" }

func (s Select) Tree(ident int) string {
	return strings.Repeat("  ", ident) + "select " + s.Selector + "\n" + s.Expr.Tree(ident+1)
}

func (f Func) Tree(ident int) string {
	result := strings.Repeat("  ", ident) + f.Name + "{\n"
	for _, arg := range f.Args {
		result += arg.Tree(ident + 1)
	}
	return result
}

func (f Func) IsContext() bool {
	return strings.IndexByte(f.Name, '=') >= 0
}

func Parse(tokens []Token) (Expr, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	p, expr, err := parseCombine(0, tokens, false)
	if err != nil {
		return expr, err
	}
	if p != len(tokens) {
		panic("failed to parse")
	}
	return expr, nil
}

func parseCombine(p int, tokens []Token, lookingForOperator bool) (int, Expr, error) {
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
				p, arg, err = parseCombine(p, tokens, false)
				if err != nil {
					return p, combine(exprs), err
				}
				if arg == nil {
					return p, combine(exprs), errors.New("empty expression")
				}
				funcexpr.Args = append(funcexpr.Args, arg)
				if tokens[p-1].Kind != TComma {
					break
				}
			}
			expr = funcexpr

		case TOp:
			p++
			if lookingForOperator {
				return p, combine(exprs), nil
			}

			op := tok.Text
			left := combine(exprs)
			for {
				var right Expr
				p, right, err = parseCombine(p, tokens, true)
				if err != nil {
					return p, combine(exprs), err
				}
				if right == nil {
					return p, combine(exprs), errors.New("empty expression")
				}
				left = Func{op, []Expr{left, right}}
				if p == len(tokens) && tokens[p-1].Kind != TOp {
					break
				}
				if tokens[p-1].Kind != TOp {
					return p, left, fmt.Errorf("unexpected token %q %#v", tokens[p-1].Kind, tokens[p-1])
				}
				op = tokens[p-1].Text
			}

			return p, left, nil

		case TSelector:
			return p, nil, fmt.Errorf("unexpected selector %q %#v", tokens[p].Kind, tokens[p])

		case TRightParen, TComma:
			p++
			return p, combine(exprs), nil

		default:
			return p, nil, fmt.Errorf("unhandled token %#v", tokens[p])
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
