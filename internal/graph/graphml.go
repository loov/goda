package graph

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/loov/goda/internal/graph/graphml"
	"github.com/loov/goda/internal/pkggraph"
)

type GraphML struct {
	out   io.Writer
	err   io.Writer
	label *template.Template
}

func (ctx *GraphML) Label(p *pkggraph.Node) string {
	var labelText strings.Builder
	err := ctx.label.Execute(&labelText, p)
	if err != nil {
		fmt.Fprintf(ctx.err, "template error: %v\n", err)
	}
	return labelText.String()
}

func (ctx *GraphML) Write(graph *pkggraph.Graph) error {
	file := graphml.NewFile()
	file.Graphs = append(file.Graphs, ctx.ConvertGraph(graph))

	file.Key = []graphml.Key{
		{For: "node", ID: "label", AttrName: "label", AttrType: "string"},
		{For: "node", ID: "module", AttrName: "module", AttrType: "string"},
		{For: "node", ID: "ynodelabel", YFilesType: "nodegraphics"},
	}

	enc := xml.NewEncoder(ctx.out)
	enc.Indent("", "\t")
	err := enc.Encode(file)
	if err != nil {
		fmt.Fprintf(ctx.err, "failed to output: %v\n", err)
	}

	return nil
}

func (ctx *GraphML) ConvertGraph(graph *pkggraph.Graph) *graphml.Graph {
	out := &graphml.Graph{}
	out.EdgeDefault = graphml.Directed

	for _, node := range graph.Sorted {
		outnode := graphml.Node{}
		outnode.ID = node.ID
		label := ctx.Label(node)

		outnode.Attrs.AddNonEmpty("label", label)
		if node.Package != nil {
			if node.Package.Module != nil {
				outnode.Attrs.AddNonEmpty("module", node.Package.Module.Path)
			}
		}

		addYedLabelAttr(&outnode.Attrs, "ynodelabel", label)
		out.Node = append(out.Node, outnode)

		for _, imp := range node.ImportsNodes {
			out.Edge = append(out.Edge, graphml.Edge{
				Source: node.ID,
				Target: imp.ID,
			})
		}
	}

	return out
}

func addYedLabelAttr(attrs *graphml.Attrs, key, value string) {
	if value == "" {
		return
	}
	var buf bytes.Buffer
	buf.WriteString(`<y:ShapeNode><y:NodeLabel>`)
	if err := xml.EscapeText(&buf, []byte(value)); err != nil {
		// this shouldn't ever happen
		panic(err)
	}
	buf.WriteString(`</y:NodeLabel></y:ShapeNode>`)
	*attrs = append(*attrs, graphml.Attr{Key: key, Value: buf.Bytes()})
}
