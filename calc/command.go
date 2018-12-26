package calc

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/template"

	"github.com/google/subcommands"
	"golang.org/x/tools/go/packages"

	"github.com/loov/goda/pkg"
)

type Command struct {
	format        string
	printStandard bool
}

func (*Command) Name() string     { return "calc" }
func (*Command) Synopsis() string { return "Calculate with pacakge sets." }
func (*Command) Usage() string {
	return `calc <pkg> [(+|-|++) <pkg>]*:
	Calculates with package dependency sets.
	
	a b    : returns packages that are used in either a or b
	a - b  : returns packages that are needed used in a and not used in b
	a + b  : returns packages that are used in both a and b
	a ^    : dependencies (e.g. golang.org/x/tools/... ^)

	Examples:

	calc github.com/loov/goda ^
		show all dependencies for "github.com/loov/goda" package 

	calc github.com/loov/goda/... ^
		show all dependencies for "github.com/loov/goda" sub-package 
	
	calc github.com/loov/goda/pkg + github.com/loov/goda/calc
		show packages shared by "github.com/loov/goda/pkg" and "github.com/loov/goda/calc"

	calc ./... ^ - golang.org/x/tools/...
		show all my dependencies excluding golang.org/x/tools
  `
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.StringVar(&cmd.format, "format", "{{.ID}}", "formatting")
	f.BoolVar(&cmd.printStandard, "std", false, "print std packages")
}

func isOp(arg string) bool {
	return arg == "+" || arg == "-" || arg == "^"
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

	t, err := template.New("").Parse(cmd.format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid format string\n")
		return subcommands.ExitFailure
	}

	stack := f.Args()

	left := pkg.NewSet()
	operation := ""

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
		if nextOperation < len(stack) && stack[nextOperation] == "^" {
			for _, root := range roots {
				delete(right, root.ID)
			}
			nextOperation++
		}

		switch operation {
		case "":
			left = pkg.Union(left, right)
		case "-":
			left = pkg.Subtract(left, right)
		case "+":
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
		err := t.Execute(os.Stdout, p)
		fmt.Fprintln(os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "template error: %v\n", err)
		}
	}

	return subcommands.ExitSuccess
}
