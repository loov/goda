package main

import (
	"context"
	"flag"
	"os"
	"path"

	"github.com/google/subcommands"

	"github.com/loov/goda/calc"
	"github.com/loov/goda/exec"
	"github.com/loov/goda/nm"
	"github.com/loov/goda/tree"
)

func main() {
	cmds := subcommands.NewCommander(flag.CommandLine, path.Base(os.Args[0]))
	cmds.Register(cmds.HelpCommand(), "")

	cmds.Register(&calc.Command{}, "")
	cmds.Register(&tree.Command{}, "")
	cmds.Register(&exec.Command{}, "")
	cmds.Register(&nm.Command{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(cmds.Execute(ctx)))
}
