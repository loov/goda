package graph

import (
	"fmt"
	"io"
	"text/template"

	"github.com/loov/goda/internal/pkggraph"
)

type TGF struct {
	out   io.Writer
	err   io.Writer
	label *template.Template
}

func (ctx *TGF) Label(p *pkggraph.Node) string { return renderLabel(ctx.label, ctx.err, p) }

func (ctx *TGF) Write(graph *pkggraph.Graph) error {
	indexCache := map[*pkggraph.Node]int{}
	for i, node := range graph.Sorted {
		label := ctx.Label(node)
		indexCache[node] = i + 1
		fmt.Fprintf(ctx.out, "%d %s\n", i+1, label)
	}

	fmt.Fprintf(ctx.out, "#\n")

	for _, node := range graph.Sorted {
		for _, imp := range node.ImportsNodes {
			fmt.Fprintf(ctx.out, "%d %d\n", indexCache[node], indexCache[imp])
		}
	}

	return nil
}
