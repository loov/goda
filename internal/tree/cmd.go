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

	"github.com/loov/goda/internal/pkgset"
	"github.com/loov/goda/internal/templates"
)

type Command struct {
	printStandard bool
	format        string
}

func (*Command) Name() string     { return "tree" }
func (*Command) Synopsis() string { return "Print dependency tree." }
func (*Command) Usage() string {
	return `tree <expr>:
	Print dependency tree of packages.
`
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&cmd.printStandard, "std", false, "print std packages")
	f.StringVar(&cmd.format, "f", "{{.ID}}", "formatting")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	t, err := templates.Parse(cmd.format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid format string: %v\n", err)
		return subcommands.ExitFailure
	}

	if !cmd.printStandard {
		go pkgset.LoadStd()
	}
	result, err := pkgset.Calc(ctx, f.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return subcommands.ExitFailure
	}
	if !cmd.printStandard {
		result = pkgset.Subtract(result, pkgset.Std())
	}
	roots := pkgset.DisjointSources(result)

	printed := map[string]bool{}

	var visit func(int, string, *packages.Package, bool)
	visit = func(ident int, parentID string, p *packages.Package, last bool) {
		if last {
			fmt.Fprint(os.Stdout, strings.Repeat("  ", ident), "  └ ")
		} else {
			fmt.Fprint(os.Stdout, strings.Repeat("  ", ident), "  ├ ")
		}

		type packageWithImporter struct {
			ParentID string
			*packages.Package
		}
		err := t.Execute(os.Stdout, packageWithImporter{
			ParentID: parentID,
			Package:  p,
		})
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
		for id := range p.Imports {
			if _, ok := result[id]; !ok {
				continue
			}
			keys = append(keys, id)
		}

		sort.Strings(keys)
		for i, id := range keys {
			dep := p.Imports[id]
			visit(ident+1, p.ID, dep, i == len(keys)-1)
		}
	}

	for _, root := range roots {
		visit(0, "\x00", root, false)
	}

	return subcommands.ExitSuccess
}
