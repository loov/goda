package graph

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/loov/goda/internal/pkggraph"
)

type Mermaid struct {
	out io.Writer
	err io.Writer

	docs    string
	nocolor bool
	shortID bool

	label *template.Template
}

func (ctx *Mermaid) Label(p *pkggraph.Node) string { return renderLabel(ctx.label, ctx.err, p) }

// mermaidLabel quotes a label for mermaid, which has no backslash escapes.
func mermaidLabel(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, "#quot;") + `"`
}

func (ctx *Mermaid) Ref(p *pkggraph.Node) string {
	return ctx.docs + p.ID
}

func (ctx *Mermaid) Write(graph *pkggraph.Graph) error {
	fmt.Fprintf(ctx.out, "flowchart LR\n")

	// sanitized paths can collide (a-b vs a_b), so use positional ids.
	ids := map[*pkggraph.Node]string{}
	for i, n := range graph.Sorted {
		ids[n] = fmt.Sprintf("n%d", i)
	}

	for _, n := range graph.Sorted {
		nid := ids[n]
		fmt.Fprintf(ctx.out, "    %v[%v]\n", nid, mermaidLabel(ctx.Label(n)))

		fmt.Fprintf(ctx.out, "    click %v %q _blank\n", nid, ctx.Ref(n))

		if color := ctx.colorOf(n); color != "" {
			fmt.Fprintf(ctx.out, "    style %v fill:%v\n", nid, color)
		}
	}

	linkIndex := 0
	for _, src := range graph.Sorted {
		srcid := ids[src]
		for _, dst := range src.ImportsNodes {
			dstid := ids[dst]
			fmt.Fprintf(ctx.out, "    %v --> %v\n", srcid, dstid)
			if color := ctx.strokeColorOf(dst); color != "" {
				fmt.Fprintf(ctx.out, "    linkStyle %v stroke:%v\n", linkIndex, color)
			}
			linkIndex++
		}
	}

	return nil
}

func (ctx *Mermaid) colorOf(p *pkggraph.Node) string {
	if p.Color != "" {
		return p.Color
	}
	if ctx.nocolor {
		return ""
	}

	hue := hueOf(p)
	return hslahex(hue, 0.6, 0.7, 0.6)
}

func (ctx *Mermaid) strokeColorOf(p *pkggraph.Node) string {
	if p.Color != "" {
		return p.Color
	}
	if ctx.nocolor {
		return ""
	}

	hue := hueOf(p)
	return hslahex(hue, 0.6, 0.3, 0.8)
}
