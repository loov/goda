package graph

import (
	"bytes"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"text/template"

	"golang.org/x/image/colornames"

	"github.com/loov/goda/internal/graph/graphml"
	"github.com/loov/goda/internal/pkggraph"
)

type GraphML struct {
	out   io.Writer
	err   io.Writer
	label *template.Template

	nocolor bool
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
		{For: "edge", ID: "yedgelabel", YFilesType: "edgegraphics"},
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

		ctx.addYedLabelAttr(&outnode.Attrs, "ynodelabel", label, node)
		out.Node = append(out.Node, outnode)

		for _, imp := range node.ImportsNodes {
			edge := graphml.Edge{
				Source: node.ID,
				Target: imp.ID,
			}
			ctx.addYedEdgeAttr(&edge.Attrs, "yedgelabel", label, node)
			out.Edge = append(out.Edge, edge)
		}
	}

	return out
}

func (ctx *GraphML) addYedLabelAttr(attrs *graphml.Attrs, key, value string, node *pkggraph.Node) {
	if value == "" {
		return
	}
	var buf bytes.Buffer
	buf.WriteString(`<y:ShapeNode>`)
	fmt.Fprintf(&buf, `<y:Fill color="%v" transparent="false" />`, ctx.colorOf(node))
	buf.WriteString(`<y:NodeLabel>`)
	if err := xml.EscapeText(&buf, []byte(value)); err != nil {
		// this shouldn't ever happen
		panic(err)
	}
	buf.WriteString(`</y:NodeLabel>`)
	buf.WriteString(`</y:ShapeNode>`)
	*attrs = append(*attrs, graphml.Attr{Key: key, Value: buf.Bytes()})
}

func (ctx *GraphML) addYedEdgeAttr(attrs *graphml.Attrs, key, value string, node *pkggraph.Node) {
	if value == "" {
		return
	}
	var buf bytes.Buffer
	buf.WriteString(`<y:PolyLineEdge>`)
	fmt.Fprintf(&buf, `<y:LineStyle color="%v" type="line" width="1.0" />`, ctx.colorOf(node))
	buf.WriteString(`</y:PolyLineEdge>`)
	*attrs = append(*attrs, graphml.Attr{Key: key, Value: buf.Bytes()})
}

func (ctx *GraphML) colorOf(p *pkggraph.Node) string {
	if p.Color != "" {
		c, ok := colornames.Map[strings.ToLower(p.Color)]
		if ok {
			return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
		}
		return p.Color
	}
	if ctx.nocolor {
		return ""
	}

	hash := sha256.Sum256([]byte(p.PkgPath))
	hue := float64(uint(hash[0])<<8|uint(hash[1])) / 0xFFFF
	return hslhex(hue, 0.6, 0.6)
}
