package pkgset

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/loov/goda/pkgset/ast"
)

func Calc(ctx context.Context, expr []string) (Set, error) {
	tokens, err := ast.Tokenize(strings.Join(expr, " "))
	if err != nil {
		return New(), fmt.Errorf("failed to tokenize: %v", err)
	}

	rootExpr, err := ast.Parse(tokens)
	if err != nil {
		return New(), fmt.Errorf("failed to parse: %v", err)
	}

	var eval func(*packages.Config, ast.Expr) (Set, error)

	evalArgs := func(cfg *packages.Config, exprs []ast.Expr) ([]Set, error) {
		args := make([]Set, len(exprs))
		var errs []error
		for i, expr := range exprs {
			var err error
			args[i], err = eval(cfg, expr)
			if err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) == 1 {
			return args, errs[0]
		}
		if len(errs) > 1 {
			return args, fmt.Errorf("%v", errs)
		}

		return args, nil
	}

	eval = func(cfg *packages.Config, e ast.Expr) (Set, error) {
		if e == nil {
			return nil, errors.New("empty expression")
		}
		switch e := e.(type) {
		case ast.Package:
			roots, err := packages.Load(cfg, string(e))
			return New(roots...), err

		case ast.Func:
			switch strings.ToLower(e.Name) {
			case "":
				args := extractLoadGroup(e)
				if len(args) > 0 {
					roots, err := packages.Load(cfg, args...)
					return New(roots...), err
				}
				// fallback to union implementation
				fallthrough

			// binary operators
			case
				"+", "add", "or",
				"-", "subtract", "exclude",
				"shared", "intersect",
				"xor":
				args, err := evalArgs(cfg, e.Args)
				if len(args) == 0 {
					return New(), err
				}

				var op func(a, b Set) Set
				switch strings.ToLower(e.Name) {
				case " ", "+", "add", "or":
					op = Union
				case "-", "subtract", "exclude":
					op = Subtract
				case "shared", "intersect":
					op = Intersect
				case "xor":
					op = SymmetricDifference
				}

				base := args[0]
				for _, arg := range args[1:] {
					base = op(base, arg)
				}
				return base, nil

			case "reach":
				if len(e.Args) != 2 {
					return nil, fmt.Errorf("reach requires two arguments: %v", e)
				}
				args, err := evalArgs(cfg, e.Args)
				return Reach(args[0], args[1]), err

			default:
				return nil, fmt.Errorf("unknown func %v: %v", e.Name, e)
			}

		case ast.Select:
			switch strings.ToLower(e.Selector) {
			case "root":
				p, ok := e.Expr.(ast.Package)
				if !ok {
					return nil, fmt.Errorf(":root cannot be used with composite expressions: %v", e)
				}

				roots, err := packages.Load(cfg, string(p))
				if err != nil {
					return nil, err
				}

				return NewRoot(roots...), nil

			case "!root":
				p, ok := e.Expr.(ast.Package)
				if !ok {
					return nil, fmt.Errorf(":!root cannot be used with composite expressions: %v", e)
				}

				roots, err := packages.Load(cfg, string(p))
				if err != nil {
					return nil, err
				}

				return Subtract(New(roots...), NewRoot(roots...)), nil

			case "source":
				set, err := eval(cfg, e.Expr)
				if err != nil {
					return nil, err
				}

				return Sources(set), nil

			case "!source":
				set, err := eval(cfg, e.Expr)
				if err != nil {
					return nil, err
				}

				return Subtract(set, Sources(set)), nil

			case "deps":
				set, err := eval(cfg, e.Expr)
				if err != nil {
					return nil, err
				}
				return Dependencies(set), nil

			default:
				return nil, fmt.Errorf("unknown selector %v: %v", e.Selector, e)
			}
		}

		return nil, nil
	}

	return eval(&packages.Config{
		Context: ctx,
		Mode:    packages.LoadImports,
		Env:     os.Environ(),
	}, rootExpr)
}

func extractLoadGroup(fn ast.Func) []string {
	var pkgs []string
	for _, arg := range fn.Args {
		pkg, ok := arg.(ast.Package)
		if !ok {
			return nil
		}
		pkgs = append(pkgs, string(pkg))
	}
	return pkgs
}
