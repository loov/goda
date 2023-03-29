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

type Sequence struct {
	Exprs []Expr
}

type Assignment struct {
	Name Package
	Expr Expr
}

type Select struct {
	Expr     Expr
	Selector string
}

type Func struct {
	Name string
	Args []Expr
}

func (v Sequence) String() string {
	var exprs []string
	for _, x := range v.Exprs {
		exprs = append(exprs, x.String())
	}
	return strings.Join(exprs, "; ")
}

func (v Assignment) String() string { return v.Name.String() + " := " + v.Expr.String() }

func (p Package) String() string { return string(p) }

func (s Select) String() string { return s.Expr.String() + ":" + s.Selector }

func (f Func) String() string {
	var args []string
	for _, arg := range f.Args {
		args = append(args, arg.String())
	}
	return f.Name + "(" + strings.Join(args, ", ") + ")"
}

func (v Sequence) Tree(ident int) string {
	var exprs []string
	for _, x := range v.Exprs {
		exprs = append(exprs, x.Tree(ident+1))
	}
	return strings.Join(exprs, "\n")
}

func (v Assignment) Tree(ident int) string {
	return v.Name.String() + " := " + v.Expr.Tree(ident+1)
}

func (p Package) Tree(ident int) string { return strings.Repeat("  ", ident) + string(p) + "\n" }

func (s Select) Tree(ident int) string {
	return strings.Repeat("  ", ident) + "select " + s.Selector + "\n" + s.Expr.Tree(ident+1)
}

func (f Func) Tree(ident int) string {
	name := f.Name
	if name == "" {
		name = "+"
	}
	result := strings.Repeat("  ", ident) + name + "{\n"
	for _, arg := range f.Args {
		result += arg.Tree(ident + 1)
	}
	result += strings.Repeat("  ", ident) + "}"
	return result
}

func (f Func) IsContext() bool {
	return strings.IndexByte(f.Name, '=') >= 0
}

func Parse(tokens []Token) (Expr, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	var seq Sequence

	p := 0
	for p < len(tokens) {
		var expr Expr
		var err error
		p, expr, err = parseCombine(p, tokens, false)
		if expr != nil {
			seq.Exprs = append(seq.Exprs, expr)
		}
		if err != nil {
			return seq, err
		}
	}

	if p != len(tokens) {
		panic("failed to parse")
	}

	if len(seq.Exprs) == 1 {
		return seq.Exprs[0], nil
	}
	return seq, nil
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

			if p < len(tokens) && tokens[p].Kind == TAssign {
				p++
				if len(exprs) != 0 {
					return p, combine(exprs), errors.New("expected \"<package> := <expr>;\"")
				}

				assign := Assignment{
					Name: Package(tok.Text),
				}

				var arg Expr
				p, arg, err = parseCombine(p, tokens, false)
				if err != nil {
					return p, arg, err
				}

				assign.Expr = arg
				return p, assign, nil
			}

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

			if tok.Kind == TLeftParen {
				if len(funcexpr.Args) != 1 {
					return p, combine(exprs), errors.New("comma delimited values between parens")
				}
				expr = funcexpr.Args[0]
			} else {
				expr = funcexpr
			}

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
				// finished parsing
				if p == len(tokens) && tokens[p-1].Kind != TOp {
					break
				}
				// finished parsing an expression
				if tokens[p-1].Kind != TOp {
					return p, left, nil
				}
				op = tokens[p-1].Text
			}

			return p, left, nil

		case TSelector:
			return p, nil, fmt.Errorf("unexpected selector %q %#v", tokens[p].Kind, tokens[p])

		case TRightParen, TComma:
			p++
			return p, combine(exprs), nil

		case TSemicolon:
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
