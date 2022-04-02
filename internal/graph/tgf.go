package graph

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/loov/goda/internal/pkggraph"
)

type TGF struct {
	out   io.Writer
	err   io.Writer
	label *template.Template
}

func (ctx *TGF) Label(p *pkggraph.Node) string {
	var labelText strings.Builder
	err := ctx.label.Execute(&labelText, p)
	if err != nil {
		fmt.Fprintf(ctx.err, "template error: %v\n", err)
	}
	return labelText.String()
}

func (ctx *TGF) Write(graph *pkggraph.Graph) error {
	indexCache := map[*pkggraph.Node]int64{}
	for i, node := range graph.Sorted {
		label := ctx.Label(node)
		indexCache[node] = int64(i + 1)
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
