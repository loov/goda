package graph

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"golang.org/x/tools/go/packages"

	"github.com/loov/goda/internal/pkggraph"
	"github.com/loov/goda/internal/pkgset"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden(t *testing.T) {
	roots, err := packages.Load(&packages.Config{
		Dir:  filepath.Join("..", "testdata", "alpha.test"),
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedModule,
	}, "./...")
	if err != nil {
		t.Fatal(err)
	}
	graph := pkggraph.From(pkgset.NewRoot(roots...))
	label := template.Must(template.New("").Parse("{{.ID}}"))

	var out, errs bytes.Buffer
	formats := map[string]Format{
		"dot":         &Dot{out: &out, err: &errs, docs: "https://pkg.go.dev/", label: label},
		"dot-cluster": &Dot{out: &out, err: &errs, docs: "https://pkg.go.dev/", label: label, clusters: true, shortID: true},
		"mermaid":     &Mermaid{out: &out, err: &errs, docs: "https://pkg.go.dev/", label: label},
		"graphml":     &GraphML{out: &out, err: &errs, label: label},
		"tgf":         &TGF{out: &out, err: &errs, label: label},
		"edges":       &Edges{out: &out, err: &errs, label: label},
		"digraph":     &Digraph{out: &out, err: &errs, label: label},
	}

	for name, format := range formats {
		t.Run(name, func(t *testing.T) {
			out.Reset()
			errs.Reset()
			if err := format.Write(graph); err != nil {
				t.Fatal(err)
			}
			if errs.Len() > 0 {
				t.Fatalf("unexpected stderr: %s", errs.String())
			}

			golden := filepath.Join("testdata", name+".golden")
			if *update {
				if err := os.WriteFile(golden, out.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(out.Bytes(), want) {
				t.Errorf("output differs from %s (run with -update to accept):\n%s", golden, out.String())
			}
		})
	}
}
