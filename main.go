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

	a b    : returns packages that are used by either a or b
	a + b  : returns packages that are used by both a and b
	a - b  : returns packages that are used by a and not used by b
	a @    : only dependencies of a
	a $    : only roots of a

	Examples:

	github.com/loov/goda @
		all dependencies for "github.com/loov/goda" package 

	github.com/loov/goda/... @
		all dependencies for "github.com/loov/goda" sub-package 
	
	github.com/loov/goda/pkg + github.com/loov/goda/calc
		packages shared by "github.com/loov/goda/pkg" and "github.com/loov/goda/calc"

	./... @ - golang.org/x/tools/...
		all my dependencies excluding golang.org/x/tools
`
}
func (*ExprHelp) SetFlags(f *flag.FlagSet) {}

func (cmd *ExprHelp) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	return subcommands.ExitUsageError
}
