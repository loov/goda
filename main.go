package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"runtime/pprof"

	"github.com/google/subcommands"

	"github.com/loov/goda/internal/cut"
	"github.com/loov/goda/internal/exec"
	"github.com/loov/goda/internal/graph"
	"github.com/loov/goda/internal/list"
	"github.com/loov/goda/internal/pkgset"
	"github.com/loov/goda/internal/tree"
	"github.com/loov/goda/internal/weight"
	"github.com/loov/goda/internal/weightdiff"
)

func main() {
	cpuProfile := flag.String("cpuprofile", "", "profile cpu usage")

	cmds := subcommands.NewCommander(flag.CommandLine, path.Base(os.Args[0]))
	cmds.Register(cmds.HelpCommand(), "")

	cmds.Register(&list.Command{}, "")
	cmds.Register(&tree.Command{}, "")
	cmds.Register(&exec.Command{}, "")
	cmds.Register(&weight.Command{}, "")
	cmds.Register(&weightdiff.Command{}, "")
	cmds.Register(&graph.Command{}, "")
	cmds.Register(&cut.Command{}, "")
	cmds.Register(&ExprHelp{}, "")
	cmds.Register(&FormatHelp{}, "")

	flag.Parse()

	os.Exit(func() int {
		ctx := context.Background()
		if *cpuProfile != "" {
			f, err := os.Create(*cpuProfile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "unable to create file: %v\n", err)
				return -1
			}
			defer func() { _ = f.Close() }()

			if err := pprof.StartCPUProfile(f); err != nil {
				fmt.Fprintf(os.Stderr, "could not start CPU profile: %v\n", err)
			}
			defer pprof.StopCPUProfile()
		}

		return int(cmds.Execute(ctx))
	}())
}

type ExprHelp struct{}

func (*ExprHelp) Name() string     { return "expr" }
func (*ExprHelp) Synopsis() string { return "Help about package expressions" }
func (*ExprHelp) Usage() string {
	return `Package expressions allow to specify calculations with dependencies.

The examples use X, Y and Z as placeholders for packages, packages paths or
package expressions.

# Basic operations:

	There are a few basic operations specified for manipulating sets of packages.

	X Y Z;
	X + Y + Z;  add(X, Y, Z);  or(X, Y, Z)
		all packages that match X, Y or Z

	X - Y - Z;  subtract(X, Y, Z);  exclude(X, Y, Z)
		packages that match X but not Y or Z

	shared(X, Y, Z);  intersect(X, Y, Z)
		packages that match all of X, Y and Z

	xor(X, Y);
		packages that match one of X or Y but not both

# Selectors:

	Selectors allow selecting parts of the dependency tree. They are
	applied in left to right order.

	For all selectors it's possible to use shorthands:

		X:+S is a shorthand for X + X:S
		X:-S is a shorthand for X - X:S


	X:all
		select X and all of its direct and indirect dependencies
	X:import, X:imp
		select direct imports of X
	X:import:all, X:imp:all
		select direct and indirect dependencies of X; X not included

	X:module, X:mod
		select X and all of its direct and indirect dependencies that
		belong to the modules of X

	X:source
		select packages not imported by any package in X

	X:main
		select packages in X named main

	X:test
		select test packages of X

# Functions:

	reach(X, Y);
		packages from X that can reach a package in Y

	incoming(X, Y);
		packages from X that directly import a package in Y, including Y

	transitive(X);
		a transitive reduction in package dependencies

	deadcode(X);
		packages from X that reach a dependency which disables dead code
		elimination (e.g. reflect.Value.MethodByName)

# Tags and OS:

	test=1(X);
		include tests when resolving X

	goos=linux(X):
		set goos to "linux" tests when resolving X

	purego=1(X):
		add tag "purego" for resolving X

# Example expressions:

	github.com/loov/goda:import
		all direct dependencies for the "github.com/loov/goda" package

	shared(github.com/loov/goda/pkgset:all, github.com/loov/goda/templates:all)
		packages directly or indirectly imported by both
		"github.com/loov/goda/pkgset" and "github.com/loov/goda/templates"

	github.com/loov/goda/...:all - golang.org/x/tools/...
		all of goda's dependencies excluding golang.org/x/tools packages

	reach(github.com/loov/goda/...:all, golang.org/x/tools/go/packages)
		packages in github.com/loov/goda/ that use golang.org/x/tools/go/packages
`
}
func (*ExprHelp) SetFlags(f *flag.FlagSet) {}

func (cmd *ExprHelp) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if f.NArg() == 0 {
		return subcommands.ExitUsageError
	}

	result, err := pkgset.Parse(ctx, f.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return subcommands.ExitFailure
	}
	fmt.Fprintln(os.Stdout, result.Tree(0))

	return subcommands.ExitSuccess
}

type FormatHelp struct{}

func (*FormatHelp) Name() string     { return "format" }
func (*FormatHelp) Synopsis() string { return "Help about formatting" }
func (*FormatHelp) Usage() string {
	return `Formatting allows to add useful information about packages.

Formatting uses -f flag for specifying the output of each package.
goda uses https://pkg.go.dev/text/template for templating and it allows
for extensive formatting.

Each package node in goda has information about the package itself,
and its statistics. Additionally there is a summary of downstream
and upstream statistics:

    type Node struct {
        *Package

        ImportsNodes []*Node

        Stat Stat // Stats about the current node.
        Up   Stat // Stats about upstream nodes.
        Down Stat // Stats about downstream nodes.
    }

    type Package struct {
        ID      string // ID is a unique identifier for a package,
        PkgPath string // PkgPath is the full import path of the package.
        Module  *packages.Module
    }

    type Module struct {
        Path    string // module path
        Version string // module version
        Main    bool   // is this the main module?
    }

This is not the full list of information about the node, however,
this is the most useful. To see inspect the structures in depth,
it's possible to use:

    goda list -f "{{ printf \"%#v\" .Package }}" .

Statistics for package contains the following information:

    type Stat struct {
        PackageCount int64

        AllFiles   Source
        Go         Source
        OtherFiles Source

        Decls  Decls
        Tokens Tokens
    }

The source information contains the following information:

    type Source struct {
        Files  int          // Files count in this stat.
        Binary int          // Binary file count.
        Size   memory.Bytes // Size in bytes of all files.
        Lines  int          // Count of non-empty lines.
        Blank  int          // Count of empty lines.
    }

As an example, to print total size of non-go files in a package:

    goda list -f "{{.ID}} {{.Stat.OtherFiles.Size}}" ./...:all

It's also possible to see information about the ast tokens and
declarations, which can be used as an approximation of the final
binary size.

    type Decls struct {
        Func  int64
        Type  int64
        Const int64
        Var   int64
        Other int64
    }

    type Tokens struct {
        Code    int64
        Comment int64
        Basic   int64
    }

"goda cut" command additionally contains:

    type Node struct {
        Cut stat.Stat

        Imports    []*Node
        ImportedBy []*Node

        InDegree  int // alias for len(ImportedBy)
        OutDegree int // alias for len(Imports)
        ...
    }

This contains summary of packages that would be removed when that
package would deleted from the project.
`
}
func (*FormatHelp) SetFlags(f *flag.FlagSet) {}

func (cmd *FormatHelp) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	fmt.Println("Run \"goda help format\" to see help about formatting.")
	return subcommands.ExitUsageError
}
