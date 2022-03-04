package graph

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/subcommands"

	"github.com/loov/goda/internal/pkggraph"
	"github.com/loov/goda/internal/pkgset"
	"github.com/loov/goda/internal/templates"
)

type Command struct {
	printStandard bool

	docs string

	outputType  string
	labelFormat string

	nocolor bool

	clusters bool
	shortID  bool
}

func (*Command) Name() string     { return "graph" }
func (*Command) Synopsis() string { return "Print dependency graph." }
func (*Command) Usage() string {
	return `graph <expr>:
	Print dependency dot graph.

Supported output types:

	dot - GraphViz dot format

	graphml - GraphML format

	tgf - Trivial Graph Format

	edges - format with each edge separately

	digraph - format with each node and its edges on a single line

	See "help expr" for further information about expressions.
	See "help format" for further information about formatting.
`
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&cmd.printStandard, "std", false, "print std packages")

	f.BoolVar(&cmd.nocolor, "nocolor", false, "disable coloring")

	f.StringVar(&cmd.docs, "docs", "https://pkg.go.dev/", "override the docs url to use")

	f.StringVar(&cmd.outputType, "type", "dot", "output type (dot, graphml, digraph, edges, tgf)")
	f.StringVar(&cmd.labelFormat, "f", "", "label formatting")

	f.BoolVar(&cmd.clusters, "cluster", false, "create clusters")
	f.BoolVar(&cmd.shortID, "short", false, "use short package id-s inside clusters")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if cmd.labelFormat == "" {
		switch cmd.outputType {
		case "dot":
			cmd.labelFormat = `{{.ID}}\l{{ .Stat.Go.Lines }} / {{ .Stat.Go.Size }}\l`
		default:
			cmd.labelFormat = `{{.ID}}`
		}
	}

	label, err := templates.Parse(cmd.labelFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid label format: %v\n", err)
		return subcommands.ExitFailure
	}

	var format Format
	switch strings.ToLower(cmd.outputType) {
	case "dot":
		format = &Dot{
			out:      os.Stdout,
			err:      os.Stderr,
			docs:     cmd.docs,
			clusters: cmd.clusters,
			nocolor:  cmd.nocolor,
			shortID:  cmd.shortID,
			label:    label,
		}
	case "digraph":
		format = &Digraph{
			out:   os.Stdout,
			err:   os.Stderr,
			label: label,
		}
	case "tgf":
		format = &TGF{
			out:   os.Stdout,
			err:   os.Stderr,
			label: label,
		}
	case "edges":
		format = &Edges{
			out:   os.Stdout,
			err:   os.Stderr,
			label: label,
		}
	case "graphml":
		format = &GraphML{
			out:   os.Stdout,
			err:   os.Stderr,
			label: label,
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown output type %q\n", cmd.outputType)
		return subcommands.ExitFailure
	}

	result, err := pkgset.Calc(ctx, f.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return subcommands.ExitFailure
	}
	if !cmd.printStandard {
		result = pkgset.Subtract(result, pkgset.Std())
	}

	graph := pkggraph.From(result)
	format.Write(graph)

	return subcommands.ExitSuccess
}

type Format interface {
	Write(*pkggraph.Graph)
}

func pkgID(p *pkggraph.Node) string {
	// Go quoting rules are similar enough to dot quoting.
	// At least enough similar to quote a Go import path.
	return strconv.Quote(p.ID)
}
