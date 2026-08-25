package graph

import (
	"fmt"
	"io"
	"text/template"

	"github.com/loov/goda/internal/pkggraph"
)

type Edges struct {
	out   io.Writer
	err   io.Writer
	label *template.Template
}

func (ctx *Edges) Label(p *pkggraph.Node) string { return renderLabel(ctx.label, ctx.err, p) }

func (ctx *Edges) Write(graph *pkggraph.Graph) error {
	labelCache := map[*pkggraph.Node]string{}
	for _, node := range graph.Sorted {
		labelCache[node] = ctx.Label(node)
	}
	for _, node := range graph.Sorted {
		for _, imp := range node.ImportsNodes {
			fmt.Fprintf(ctx.out, "%s %s\n", labelCache[node], labelCache[imp])
		}
	}

	return nil
}
