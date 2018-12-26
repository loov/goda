# Goda

Goda is a Go dependency analysing toolset.

Subcommands:

* `goda calc`: calculate with dependency sets
* `goda tree`: print dependency tree
* `goda exec`: execute commands with more information


## `goda calc`

Implements calculating with dependency sets.

* `goda calc github.com/loov/goda/...`: lists all subpackages
* `goda calc github.com/loov/goda/... @`: lists all dependencies
* `goda calc github.com/loov/goda/pkg + github.com/loov/goda/calc`: lists packages used by both `github.com/loov/goda/pkg` and `github.com/loov/goda/calc`
* `goda calc github.com/loov/goda/... @ - golang.org/x/tools/...`: lists all dependencies and excludes `golang.org/x/tools` packages.