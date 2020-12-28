package cut

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/google/subcommands"
	"golang.org/x/tools/go/packages"

	"github.com/loov/goda/memory"
	"github.com/loov/goda/pkgset"
	"github.com/loov/goda/templates"
)

type Command struct {
	printStandard bool
	format        string
	exclude       string
}

func (*Command) Name() string     { return "cut" }
func (*Command) Synopsis() string { return "Analyse indirect-dependencies." }
func (*Command) Usage() string {
	return `cut <expr>:
	Print information about indirect-dependencies.
	It shows packages whose removal would remove the most indirect dependencies.

	See "help expr" for further information about expressions.
`
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&cmd.printStandard, "std", false, "print std packages")
	f.StringVar(&cmd.format, "f", "{{.ID}}\tin:{{.InDegree}}\tpkgs:{{.Cut.Packages}}\tsize:{{.Cut.SourceSize}}\tloc:{{.Cut.Lines}}", "info formatting")
	f.StringVar(&cmd.exclude, "exclude", "", "package expr to exclude from output")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "missing package names\n")
		return subcommands.ExitUsageError
	}

	t, err := templates.Parse(cmd.format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid label string: %v\n", err)
		return subcommands.ExitFailure
	}

	result, err := pkgset.Calc(ctx, f.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return subcommands.ExitFailure
	}

	excluded := pkgset.New()
	if cmd.exclude != "" {
		excluded, err = pkgset.Calc(ctx, strings.Fields(cmd.exclude))
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return subcommands.ExitFailure
		}
	}

	if !cmd.printStandard {
		result = pkgset.Subtract(result, pkgset.Std())
	}

	stats := map[string]*Stat{}
	statlist := []*Stat{}

	var include func(parent *Stat, p *packages.Package)
	include = func(parent *Stat, p *packages.Package) {
		if p, ok := stats[p.ID]; ok {
			parent.Import(p)
			return
		}

		stat := &Stat{
			Package: p,
		}
		stats[p.ID] = stat
		if _, analyse := result[p.ID]; analyse {
			statlist = append(statlist, stat)
		}

		parent.Import(stat)
		for _, child := range p.Imports {
			include(stat, child)
		}
	}

	for _, p := range result {
		include(nil, p)
	}

	for _, p := range stats {
		if !cmd.printStandard && pkgset.IsStd(p.Package) {
			continue
		}

		p.Info.Packages = 1
		p.Info.Lines = templates.LineCount(p.Package)
		p.Info.SourceSize = templates.SourceSize(p.Package)
	}

	for _, stat := range statlist {
		Reset(stats)
		stat.Cut = Erase(stat)
	}

	sort.Slice(statlist, func(i, k int) bool {
		if statlist[i].InDegree() == statlist[k].InDegree() {
			return statlist[i].Cut.Packages > statlist[k].Cut.Packages
		}
		return statlist[i].InDegree() < statlist[k].InDegree()
	})

	for _, stat := range statlist {
		if _, exclude := excluded[stat.ID]; exclude {
			continue
		}

		err := t.Execute(os.Stdout, stat)
		fmt.Fprintln(os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "template error: %v\n", err)
		}
	}

	return subcommands.ExitSuccess
}

func Reset(stats map[string]*Stat) {
	for _, stat := range stats {
		stat.indegree = len(stat.ImportedBy)
	}
}

func Erase(stat *Stat) Info {
	cut := stat.Info
	for _, imp := range stat.Imports {
		imp.indegree--
		if imp.indegree == 0 {
			cut = cut.Add(Erase(imp))
		}
	}
	return cut
}

type Info struct {
	Packages   int
	Lines      int64
	SourceSize memory.Bytes
}

func (a Info) Add(b Info) Info {
	return Info{
		Packages:   a.Packages + b.Packages,
		Lines:      a.Lines + b.Lines,
		SourceSize: a.SourceSize + b.SourceSize,
	}
}

type Stat struct {
	*packages.Package

	Info Info
	Cut  Info

	Imports    []*Stat
	ImportedBy []*Stat

	indegree int
}

func (parent *Stat) Pkg() *packages.Package { return parent.Package }
func (parent *Stat) InDegree() int          { return len(parent.ImportedBy) }
func (parent *Stat) OutDegree() int         { return len(parent.Imports) }

func (parent *Stat) Import(child *Stat) {
	if parent == nil {
		return
	}

	if !hasPackage(parent.Imports, child) {
		child.indegree++
		child.ImportedBy = append(child.ImportedBy, parent)

		parent.Imports = append(parent.Imports, child)
	}
}

func hasPackage(xs []*Stat, p *Stat) bool {
	for _, x := range xs {
		if x == p {
			return true
		}
	}
	return false
}
