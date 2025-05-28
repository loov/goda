package graph

import (
	"crypto/sha256"
	"fmt"
	"io"
	"regexp"
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

func (ctx *Mermaid) Label(p *pkggraph.Node) string {
	var labelText strings.Builder
	err := ctx.label.Execute(&labelText, p)
	if err != nil {
		fmt.Fprintf(ctx.err, "template error: %v\n", err)
	}
	return labelText.String()
}

var rxMermaidID = regexp.MustCompile("[^a-zA-Z0-9]+")

func (ctx *Mermaid) PkgID(p *pkggraph.Node) string {
	// Go quoting rules are similar enough to dot quoting.
	// At least enough similar to quote a Go import path.
	return rxMermaidID.ReplaceAllString(p.ID, "_")
}

func (ctx *Mermaid) Ref(p *pkggraph.Node) string {
	return ctx.docs + p.ID
}

func (ctx *Mermaid) writeGraphProperties() {
}

func (ctx *Mermaid) Write(graph *pkggraph.Graph) error {
	return ctx.WriteRegular(graph)
}

func (ctx *Mermaid) WriteRegular(graph *pkggraph.Graph) error {
	fmt.Fprintf(ctx.out, "flowchart LR\n")
	ctx.writeGraphProperties()

	for _, n := range graph.Sorted {
		nid := ctx.PkgID(n)
		fmt.Fprintf(ctx.out, "    %v[%q]\n", nid, ctx.Label(n))

		if ref := ctx.Ref(n); ref != "" {
			fmt.Fprintf(ctx.out, "    click %v %q _blank\n", nid, ref)
		}

		if color := ctx.colorOf(n); color != "" {
			fmt.Fprintf(ctx.out, "    style %v fill:%v\n", nid, color)
		}
	}

	linkIndex := 0
	for _, src := range graph.Sorted {
		srcid := ctx.PkgID(src)
		for _, dst := range src.ImportsNodes {
			dstid := ctx.PkgID(dst)
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

	hash := sha256.Sum256([]byte(p.PkgPath))
	hue := float64(uint(hash[0])<<8|uint(hash[1])) / 0xFFFF
	return hslahex(hue, 0.6, 0.7, 0.6)
}

func (ctx *Mermaid) strokeColorOf(p *pkggraph.Node) string {
	if p.Color != "" {
		return p.Color
	}
	if ctx.nocolor {
		return ""
	}

	hash := sha256.Sum256([]byte(p.PkgPath))
	hue := float64(uint(hash[0])<<8|uint(hash[1])) / 0xFFFF
	return hslahex(hue, 0.6, 0.3, 0.8)
}
