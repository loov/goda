package tree

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/template"

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

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
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
	cmd.print(os.Stdout, t, result)
	return subcommands.ExitSuccess
}

// print writes the dependency tree of result to w.
func (cmd *Command) print(w io.Writer, t *template.Template, result pkgset.Set) {
	roots := pkgset.Sources(result)

	lineNr := 0
	printed := map[string]int{}

	var visit func(int, string, *packages.Package, bool)
	visit = func(ident int, parentID string, p *packages.Package, last bool) {
		lineNr++
		if last {
			fmt.Fprintf(w, "%-4d%s  └ ", lineNr, strings.Repeat("  ", ident))
		} else {
			fmt.Fprintf(w, "%-4d%s  ├ ", lineNr, strings.Repeat("  ", ident))
		}

		type packageWithImporter struct {
			ParentID string
			*packages.Package
		}
		err := t.Execute(w, packageWithImporter{
			ParentID: parentID,
			Package:  p,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "template error: %v\n", err)
		}

		if line, ok := printed[p.ID]; ok {
			fmt.Fprintf(w, " @%d\n", line)
			return
		}
		if pkgset.IsStd(p) {
			fmt.Fprintln(w, " ~")
			return
		}
		fmt.Fprintln(w)

		printed[p.ID] = lineNr
		deps := []*packages.Package{}
		for _, dep := range p.Imports {
			if _, ok := result[dep.ID]; !ok {
				continue
			}
			deps = append(deps, dep)
		}

		sort.Slice(deps, func(i, k int) bool { return deps[i].ID < deps[k].ID })
		for i, dep := range deps {
			visit(ident+1, p.ID, dep, i == len(deps)-1)
		}
	}

	sorted := roots.Sorted()
	for i, root := range sorted {
		visit(0, "\x00", root, i == len(sorted)-1)
	}
}
