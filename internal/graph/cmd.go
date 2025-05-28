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
	colors  exprColors

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

	mermaid - mermaid flowchart

	See "help expr" for further information about expressions.
	See "help format" for further information about formatting.
`
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&cmd.printStandard, "std", false, "print std packages")

	f.BoolVar(&cmd.nocolor, "nocolor", false, "disable coloring")
	f.Var(&cmd.colors, "color", "specify a color for packages in a given expr (e.g. `-color red=./...`)")

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
	case "mermaid":
		format = &Mermaid{
			out:     os.Stdout,
			err:     os.Stderr,
			docs:    cmd.docs,
			nocolor: cmd.nocolor,
			shortID: cmd.shortID,
			label:   label,
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
			out:     os.Stdout,
			err:     os.Stderr,
			label:   label,
			nocolor: cmd.nocolor,
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown output type %q\n", cmd.outputType)
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

	graph := pkggraph.From(result)
	for _, color := range cmd.colors {
		target, err := pkgset.Calc(ctx, []string{color.Expr})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to evaluate color expression %q: %v", color.Expr, err)
			continue
		}
		for id := range target {
			if n, ok := graph.Packages[id]; ok {
				n.Color = color.Color
			}
		}
	}

	if err := format.Write(graph); err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

type Format interface {
	Write(*pkggraph.Graph) error
}

func pkgID(p *pkggraph.Node) string {
	// Go quoting rules are similar enough to dot quoting.
	// At least enough similar to quote a Go import path.
	return strconv.Quote(p.ID)
}

// exprColors allows to define coloring for the given package set.
type exprColors []exprColor

// exprColor defines a color for an expression.
type exprColor struct {
	Color string
	Expr  string
}

// Set implements flag.Value.
func (c *exprColors) Set(v string) error {
	for _, v := range strings.Split(v, ";") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		var x exprColor
		if err := x.Set(v); err != nil {
			return err
		}
		*c = append(*c, x)
	}
	return nil
}

// String implements flag.Value.
func (c *exprColors) String() string {
	var xs []string
	for _, x := range *c {
		xs = append(xs, x.String())
	}
	return strings.Join(xs, ";")
}

// Set implements flag.Value.
func (c *exprColor) Set(v string) error {
	color, expr, ok := strings.Cut(v, "=")
	if !ok {
		return fmt.Errorf("invalid expression coloring %q", v)
	}
	c.Color, c.Expr = color, expr
	return nil
}

// String implements flag.Value.
func (c *exprColor) String() string {
	return c.Color + "=" + c.Expr
}
