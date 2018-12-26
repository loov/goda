package calc

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
	"golang.org/x/tools/go/packages"

	"github.com/loov/ago/pkg"
)

type Command struct {
	printStandard bool
}

func (*Command) Name() string     { return "calc" }
func (*Command) Synopsis() string { return "Calculate with pacakge sets." }
func (*Command) Usage() string {
	return `calc <pkg> [(+|-|++) <pkg>]*:
	Calculates with package dependency sets.
	
	a - b: returns packages that are needed used in a and not used in b
	a + b: returns packages that are used in either a or b
	a ++ b: returns packages that are used in both a and b
  `
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&cmd.printStandard, "std", false, "print std packages")
}

func isOp(arg string) bool {
	return arg == "+" || arg == "-" || arg == "++"
}

func findOp(stack []string) int {
	for i := 0; i < len(stack); i++ {
		if isOp(stack[i]) {
			return i
		}
	}

	return len(stack)
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "missing package names\n")
		return subcommands.ExitUsageError
	}

	stack := f.Args()

	left := pkg.NewSet()
	operation := "+"

	for len(stack) > 0 {
		nextOperation := findOp(stack)
		load := stack[:nextOperation]

		roots, err := packages.Load(&packages.Config{
			Context: ctx,
			Mode:    packages.LoadImports,
			Env:     os.Environ(),
		}, load...)

		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load %v: %v\n", load, err)
			return subcommands.ExitFailure
		}

		right := pkg.NewSet(roots...)
		switch operation {
		case "+":
			left = pkg.Union(left, right)
		case "-":
			left = pkg.Subtract(left, right)
		case "++":
			left = pkg.Intersect(left, right)
		}

		if nextOperation >= len(stack) {
			break
		}

		operation, stack = stack[nextOperation], stack[nextOperation+1:]
	}

	pkgs := left.Sorted()
	for _, p := range pkgs {
		if !cmd.printStandard && pkg.IsStd(p) {
			continue
		}
		fmt.Fprintf(os.Stdout, "%v\n", p.ID)
	}

	return subcommands.ExitSuccess
}
