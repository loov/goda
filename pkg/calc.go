package pkg

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/tools/go/packages"
)

func Calc(ctx context.Context, expr []string) (Set, error) {
	left := NewSet()
	operation := ""

	for len(expr) > 0 {
		nextOperation := findOp(expr)
		load := expr[:nextOperation]

		roots, err := packages.Load(&packages.Config{
			Context: ctx,
			Mode:    packages.LoadImports,
			Env:     os.Environ(),
		}, load...)

		if err != nil {
			return left, fmt.Errorf("failed to load %v: %v", load, err)
		}

		right := NewSet(roots...)
		if nextOperation < len(expr) && expr[nextOperation] == "@" {
			for _, root := range roots {
				delete(right, root.ID)
			}
			nextOperation++
		}

		switch operation {
		case "":
			left = Union(left, right)
		case "-":
			left = Subtract(left, right)
		case "+":
			left = Intersect(left, right)
		}

		if nextOperation >= len(expr) {
			break
		}

		operation, expr = expr[nextOperation], expr[nextOperation+1:]
	}

	return left, nil
}

func isOp(arg string) bool {
	return arg == "+" || arg == "-" || arg == "@"
}

func findOp(stack []string) int {
	for i := 0; i < len(stack); i++ {
		if isOp(stack[i]) {
			return i
		}
	}

	return len(stack)
}
