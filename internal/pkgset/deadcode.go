package pkgset

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Deadcode returns packages from the set that directly or indirectly call
// a function annotated with <ReflectMethod> in the linker dependency graph.
// This is typically caused by reflect.Value.MethodByName or similar calls,
// which force the linker to keep all interface methods alive.
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

	callers, reflectMethodSyms := parseDumpDep(&stderr)

	// Walk backwards from <ReflectMethod> symbols to find all transitive callers.
	reachable := map[string]bool{}
	var walk func(sym string)
	walk = func(sym string) {
		if reachable[sym] {
			return
		}
		reachable[sym] = true
		for _, caller := range callers[sym] {
			walk(caller)
		}
	}
	for sym := range reflectMethodSyms {
		walk(sym)
	}

	// Extract packages from reachable symbols and intersect with input set.
	result := make(Set)
	for sym := range reachable {
		pkg := packageFromSymbol(sym)
		if pkg == "" {
			continue
		}
		if p, ok := pkgs[pkg]; ok {
			result[pkg] = p
		}
	}

	return result, nil
}

// parseDumpDep parses the output of `go build -ldflags="-dumpdep"`.
// It returns a reverse dependency graph (callers) mapping each symbol to its
// callers, and a set of symbols annotated with <ReflectMethod>.
func parseDumpDep(r io.Reader) (callers map[string][]string, reflectMethodSyms map[string]bool) {
	callers = map[string][]string{}
	reflectMethodSyms = map[string]bool{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse lines in the format:
		//   "source -> target"
		//   "source <ReflectMethod> -> target"
		//   "source -> target <ReflectMethod>"
		source, target, ok := strings.Cut(line, " -> ")
		if !ok {
			continue
		}
		source = strings.TrimSpace(source)
		target = strings.TrimSpace(target)

		// Check for <ReflectMethod> annotation on either side.
		if strings.Contains(target, " <ReflectMethod>") {
			target, _, _ = strings.Cut(target, " <")
			reflectMethodSyms[target] = true
		}
		if strings.Contains(source, " <ReflectMethod>") {
			source, _, _ = strings.Cut(source, " <")
			reflectMethodSyms[source] = true
		}

		// Strip any other annotations.
		if idx := strings.Index(source, " <"); idx >= 0 {
			source = source[:idx]
		}
		if idx := strings.Index(target, " <"); idx >= 0 {
			target = target[:idx]
		}

		callers[target] = append(callers[target], source)
	}

	return callers, reflectMethodSyms
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
