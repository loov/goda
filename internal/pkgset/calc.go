package pkgset

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/loov/goda/internal/pkgset/ast"
)

func Parse(ctx context.Context, expr []string) (ast.Expr, error) {
	tokens, err := ast.Tokenize(strings.Join(expr, " "))
	if err != nil {
		return nil, fmt.Errorf("failed to tokenize: %v", err)
	}

	root, err := ast.Parse(tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %v", err)
	}

	return root, nil
}

func Calc(parentContext context.Context, expr []string) (Set, error) {
	if len(expr) == 0 {
		expr = []string{"."}
	}

	rootExpr, err := Parse(parentContext, expr)
	if err != nil {
		return New(), err
	}

	var eval func(*Context, ast.Expr) (Set, error)

	evalArgs := func(ctx *Context, exprs []ast.Expr) ([]Set, error) {
		args := make([]Set, len(exprs))
		var errs []error
		for i, expr := range exprs {
			var err error
			args[i], err = eval(ctx, expr)
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

	eval = func(ctx *Context, e ast.Expr) (Set, error) {
		if e == nil {
			return nil, errors.New("empty expression")
		}
		switch e := e.(type) {
		case ast.Package:
			roots, err := ctx.Load(string(e))
			return NewRoot(roots...), err

		case ast.Func:
			if e.IsContext() {
				subctx := ctx.Clone()
				key, value := KeyValue(e.Name)
				subctx.Set(key, value)
				if len(e.Args) != 1 {
					return nil, fmt.Errorf("expected 1 argument found %d", len(e.Args))
				}
				return eval(subctx, e.Args[0])
			}

			switch strings.ToLower(e.Name) {
			case "":
				args := extractLoadGroup(e)
				if len(args) > 0 {
					roots, err := ctx.Load(args...)
					return NewRoot(roots...), err
				}
				// fallback to union implementation
				fallthrough

			// binary operators
			case
				"+", "add", "or",
				"-", "subtract", "exclude",
				"shared", "intersect",
				"xor":
				args, err := evalArgs(ctx, e.Args)
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
				default:
					return nil, fmt.Errorf("unknown op %q", e.Name)
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
				args, err := evalArgs(ctx, e.Args)
				return Reach(args[0], args[1]), err

			case "incoming":
				if len(e.Args) != 2 {
					return nil, fmt.Errorf("incoming requires two arguments: %v", e)
				}
				args, err := evalArgs(ctx, e.Args)
				return Incoming(args[0], args[1]), err

			case "transitive":
				if len(e.Args) != 1 {
					return nil, fmt.Errorf("transitive requires one argument: %v", e)
				}
				args, err := evalArgs(ctx, e.Args)
				return Transitive(args[0]), err

			default:
				return nil, fmt.Errorf("unknown func %v: %v", e.Name, e)
			}

		case ast.Select:
			combineOp, selector := "", e.Selector
			combine := func(source, result Set) Set { return result }

			switch selector[0] {
			case '+':
				combine = Union
				combineOp, selector = selector[:1], selector[1:]
			case '-':
				combine = Subtract
				combineOp, selector = selector[:1], selector[1:]
			}

			switch strings.ToLower(selector) {
			case "all":
				set, err := eval(ctx, e.Expr)
				if err != nil {
					return nil, err
				}
				return combine(set, NewAll(set)), nil

			case "import", "imp":
				set, err := eval(ctx, e.Expr)
				if err != nil {
					return nil, err
				}
				return combine(set, DirectDependencies(set)), nil

			case "source":
				set, err := eval(ctx, e.Expr)
				if err != nil {
					return nil, err
				}

				return combine(set, Sources(set)), nil

			case "nosource": // Deprecated
				set, err := eval(ctx, e.Expr)
				if err != nil {
					return nil, err
				}

				return combine(set, Subtract(set, Sources(set))), nil

			case "main":
				set, err := eval(ctx, e.Expr)
				if err != nil {
					return nil, err
				}
				return combine(set, Main(set)), nil

			case "test":
				if pkg, ok := e.Expr.(ast.Package); ok {
					switch combineOp {
					case "+":
						roots, err := ctx.LoadWithTests(string(pkg))
						return NewRoot(roots...), err
					case "-":
						roots, err := ctx.LoadWithoutTests(string(pkg))
						return NewRoot(roots...), err
					case "":
						roots, err := ctx.LoadWithTests(string(pkg))
						withTests := NewRoot(roots...)
						return Test(withTests), err
					default:
						return nil, fmt.Errorf("unhandled combine op %q", combineOp)
					}
				}

				set, err := eval(ctx, e.Expr)
				if err != nil {
					return nil, err
				}

				roots, err := ctx.LoadWithTests(set.IDs()...)
				withTests := NewRoot(roots...)
				return combine(set, Test(withTests)), err

			default:
				return nil, fmt.Errorf("unknown selector %v: %v", e.Selector, e)
			}

		default:
			return nil, fmt.Errorf("unknown token %T", e)
		}
	}

	return eval(&Context{
		Context: parentContext,
		Env:     Strings(os.Environ()),
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
