package graph

import (
	"fmt"
	"io"
	"text/template"

	"github.com/loov/goda/internal/pkggraph"
)

type Digraph struct {
	out   io.Writer
	err   io.Writer
	label *template.Template
}

func (ctx *Digraph) Label(p *pkggraph.Node) string { return renderLabel(ctx.label, ctx.err, p) }

func (ctx *Digraph) Write(graph *pkggraph.Graph) error {
	labelCache := map[*pkggraph.Node]string{}
	for _, node := range graph.Sorted {
		labelCache[node] = ctx.Label(node)
	}
	for _, node := range graph.Sorted {
		fmt.Fprintf(ctx.out, "%s", labelCache[node])
		for _, imp := range node.ImportsNodes {
			fmt.Fprintf(ctx.out, " %s", labelCache[imp])
		}
		fmt.Fprintf(ctx.out, "\n")
	}

	return nil
}
