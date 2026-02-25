package pkgset

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Deadcode returns packages from the set that have a dependency path to a
// package that disables dead code elimination. This is typically caused by
// reflect.Value.MethodByName or similar calls, which force the linker to
// keep all interface methods alive.
func Deadcode(ctx context.Context, pkgs Set) (Set, error) {
	ids := pkgs.IDs()
	if len(ids) == 0 {
		return New(), nil
	}

	args := []string{"build", "-ldflags=-dumpdep", "-o", "/dev/null"}
	args = append(args, ids...)

	cmd := exec.CommandContext(ctx, "go", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go build -dumpdep failed: %w\n%s", err, stderr.String())
	}

	// Expand pkgs to include all transitive dependencies, so we can
	// find ReflectMethod packages anywhere in the dependency tree.
	allPkgs := NewAll(pkgs)

	reflectMethodPkgs := make(Set)

	scanner := bufio.NewScanner(&stderr)
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.Contains(line, "<ReflectMethod>") {
			continue
		}

		// Line format: "pkg.Symbol <ReflectMethod> -> target"
		sym, _, _ := strings.Cut(line, "<ReflectMethod>")
		sym = strings.TrimSpace(sym)
		pkg := packageFromSymbol(sym)
		if pkg == "" {
			continue
		}

		if p, ok := allPkgs[pkg]; ok {
			reflectMethodPkgs[pkg] = p
		}
	}

	if len(reflectMethodPkgs) == 0 {
		return New(), nil
	}

	return Intersect(pkgs, Reach(allPkgs, reflectMethodPkgs)), nil
}

// packageFromSymbol extracts the package path from a linker symbol name.
// e.g. "text/template.(*state).evalField" -> "text/template"
func packageFromSymbol(sym string) string {
	sym = strings.TrimSpace(sym)
	if sym == "" {
		return ""
	}

	// Find the last '/' to separate the package path prefix from the final element.
	lastSlash := strings.LastIndexByte(sym, '/')
	rest := sym
	prefix := ""
	if lastSlash >= 0 {
		prefix = sym[:lastSlash+1]
		rest = sym[lastSlash+1:]
	}

	// In the final element, the first '.' separates the package name from the symbol.
	pkgName, _, ok := strings.Cut(rest, ".")
	if !ok {
		return ""
	}

	return prefix + pkgName
}
