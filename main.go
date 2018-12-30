package main

import (
	"context"
	"flag"
	"os"
	"path"

	"github.com/google/subcommands"

	"github.com/loov/goda/cut"
	"github.com/loov/goda/exec"
	"github.com/loov/goda/graph"
	"github.com/loov/goda/list"
	"github.com/loov/goda/nm"
	"github.com/loov/goda/tree"
)

func main() {
	cmds := subcommands.NewCommander(flag.CommandLine, path.Base(os.Args[0]))
	cmds.Register(cmds.HelpCommand(), "")

	cmds.Register(&list.Command{}, "")
	cmds.Register(&tree.Command{}, "")
	cmds.Register(&exec.Command{}, "")
	cmds.Register(&nm.Command{}, "")
	cmds.Register(&graph.Command{}, "")
	cmds.Register(&cut.Command{}, "")
	cmds.Register(&ExprHelp{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(cmds.Execute(ctx)))
}

type ExprHelp struct{}

func (*ExprHelp) Name() string     { return "expr" }
func (*ExprHelp) Synopsis() string { return "Help about package expressions" }
func (*ExprHelp) Usage() string {
	return `Package expressions allow to specify calculations with dependencies:

# Basic operations
	There are a few basic oprations specified for manipulating sets of packages.

	a b c;  
	a + b + c;  add(a, b, c);  or(a, b, c)
		returns packages that are used by a or b

	a - b - c;  subtract(a, b, c);  exclude(a, b, c)
		returns packages that are used by a and not used by b

	shared(a, b, c);  intersect(a, b, c)
		returns packages that are used by both a and b

	xor(a, b);
		returns packages that are different between a and b

# Selectors
	Selectors allow selecting parts of the dependency tree.

	a:root
		keeps packages that are explictly included by a (excluding dependencies)
	a:!root
		selects excluding roots, shorthand for (a - a:root)

	a:source
		keeps packages that have no dependents
	a:!source
		selects excluding sources, shorthand for (a - a:source)

	a:deps
		dependencies of a (a not included)

# Functions:

	reach(a, b);
		lists packages from a that can reach a package in b

# Example expressions

	github.com/loov/goda:deps
		all dependencies for "github.com/loov/goda" package 

	github.com/loov/goda/...:!root
		all dependencies for "github.com/loov/goda" sub-package 

	shared(github.com/loov/goda/pkgset, github.com/loov/goda/templates)
		packages shared by "github.com/loov/goda/pkgset" and "github.com/loov/goda/templates"

	github.com/loov/goda/...:!root - golang.org/x/tools/...
		all dependencies excluding golang.org/x/tools
	
	reach(github.com/loov/goda/...:root, golang.org/x/tools/go/packages:root)
		packages in github.com/loov/goda/ that use golang.org/x/tools/go/packages
`
}
func (*ExprHelp) SetFlags(f *flag.FlagSet) {}

func (cmd *ExprHelp) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	return subcommands.ExitUsageError
}
