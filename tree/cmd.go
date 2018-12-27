package tree

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/google/subcommands"
	"golang.org/x/tools/go/packages"

	"github.com/loov/goda/pkgset"
	"github.com/loov/goda/templates"
)

type Command struct {
	printStandard bool
	format        string
}

func (*Command) Name() string     { return "tree" }
func (*Command) Synopsis() string { return "Print dependency tree." }
func (*Command) Usage() string {
	return `tree <pkg>+:
	Print dependency tree of packages.
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

	roots, err := packages.Load(&packages.Config{
		Context: ctx,
		Mode:    packages.LoadImports,
		Env:     os.Environ(),
	}, f.Args()...)

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load %v: %v\n", f.Args(), err)
		return subcommands.ExitFailure
	}

	printed := map[string]bool{}

	var visit func(int, *packages.Package, bool)
	visit = func(ident int, p *packages.Package, last bool) {
		if last {
			fmt.Fprint(os.Stdout, strings.Repeat("  ", ident), "  └ ")
		} else {
			fmt.Fprint(os.Stdout, strings.Repeat("  ", ident), "  ├ ")
		}

		err := t.Execute(os.Stdout, p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "template error: %v\n", err)
		}

		if printed[p.ID] || pkgset.IsStd(p) {
			fmt.Fprintln(os.Stdout, " ~")
			return
		}
		fmt.Fprintln(os.Stdout)

		printed[p.ID] = true
		keys := []string{}
		for id, dep := range p.Imports {
			if !cmd.printStandard && pkgset.IsStd(dep) {
				continue
			}
			keys = append(keys, id)
		}

		sort.Strings(keys)
		for i, id := range keys {
			dep := p.Imports[id]
			visit(ident+1, dep, i == len(keys)-1)
		}
	}

	for _, root := range roots {
		visit(0, root, false)
	}

	return subcommands.ExitSuccess
}
