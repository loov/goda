package cut

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/loov/goda/internal/pkgset"
	"github.com/loov/goda/internal/templates"
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
	result := pkgset.NewRoot(roots...)

	var cmd Command
	cmd.SetFlags(flag.NewFlagSet("cut", flag.ContinueOnError))
	cmd.format = "{{.ID}}\t{{.InDegree}}\t{{.Cut.PackageCount}}\t{{.Cut.Go.Lines}}"

	var out bytes.Buffer
	if cmd.print(&out, templates.MustParse(cmd.format), result, pkgset.New()) {
		t.Fatal("template failed")
	}

	golden := filepath.Join("testdata", "cut.golden")
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
}
