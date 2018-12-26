package calc

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"

	"github.com/loov/goda/pkg"
	"github.com/loov/goda/templates"
)

type Command struct {
	printStandard bool
	format        string
}

func (*Command) Name() string     { return "calc" }
func (*Command) Synopsis() string { return "Calculate with pacakge sets." }
func (*Command) Usage() string {
	return `calc <pkg> [(|+|-|@) <pkg>]*:
	Calculates with package dependency sets.
	
	a b    : returns packages that are used by either a or b
	a + b  : returns packages that are used by both a and b
	a - b  : returns packages that are used by a and not used by b
	a @    : dependencies (e.g. golang.org/x/tools/... @)

	Examples:

	calc github.com/loov/goda @
		show all dependencies for "github.com/loov/goda" package 

	calc github.com/loov/goda/... @
		show all dependencies for "github.com/loov/goda" sub-package 
	
	calc github.com/loov/goda/pkg + github.com/loov/goda/calc
		show packages shared by "github.com/loov/goda/pkg" and "github.com/loov/goda/calc"

	calc ./... @ - golang.org/x/tools/...
		show all my dependencies excluding golang.org/x/tools
  `
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&cmd.printStandard, "std", false, "print std packages")
	f.StringVar(&cmd.format, "format", "{{.ID}}", "formatting")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "missing package names\n")
		return subcommands.ExitUsageError
	}

	t, err := templates.Parse(cmd.format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid format string\n")
		return subcommands.ExitFailure
	}

	result, err := pkg.Calc(ctx, f.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return subcommands.ExitFailure
	}

	pkgs := result.Sorted()
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
