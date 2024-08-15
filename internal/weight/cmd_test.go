package weight_test

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExecuteSelf(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.bin")

	output, err := exec.Command("go", "build", "-o", out, "github.com/loov/goda").CombinedOutput()
	if err != nil {
		t.Log(string(output))
		t.Fatal(err)
	}

	output, err = exec.Command(out, "weight", out).CombinedOutput()
	t.Log(string(output))
	if err != nil {
		t.Fatal(err)
	}
}
