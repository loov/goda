# goda

Goda is a Go dependency analysing toolset.

```
Usage: goda <flags> <subcommand> <subcommand args>

Subcommands:
	calc             Calculate with pacakge sets.
	exec             Run command with extended statistics.
	graph            Print dependency graph.
	nm               Analyse binary symbols.
	tree             Print dependency tree.
```

## `goda calc`

Implements calculating with dependency sets.

* `goda calc github.com/loov/goda/...`: lists all subpackages
* `goda calc github.com/loov/goda/... @`: lists all dependencies
* `goda calc github.com/loov/goda/pkg + github.com/loov/goda/calc`: lists packages used by both `github.com/loov/goda/pkg` and `github.com/loov/goda/calc`
* `goda calc github.com/loov/goda/... @ - golang.org/x/tools/...`: lists all dependencies and excludes `golang.org/x/tools` packages.