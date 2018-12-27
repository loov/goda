# goda

Goda is a Go dependency analysing toolset.

```
Usage: goda <flags> <subcommand> <subcommand args>

Subcommands:
	cut              Print dependencies cutting information.
	exec             Run command with extended statistics.
	expr             Help about package expressions
	graph            Print dependency graph.
	list             List packages
	nm               Analyse binary symbols.
	tree             Print dependency tree.
```

Cool things it can do:

```
# draw graph of packages in github.com/loov/goda
goda graph github.com/loov/goda/... $ | dot -Tsvg -o graph.svg

# list dependencies of github.com/loov/goda
goda list github.com/loov/goda/... @

# list packages shared by github.com/loov/goda/pkgset and github.com/loov/goda/calc
goda list github.com/loov/goda/pkgset + github.com/loov/goda/calc

# list how much memory each symbol in the final binary is taking
goda nm -h $GOPATH/bin/goda

# list how much dependencies would be removed by cutting a package
goda cut ./...

# print dependency tree of all sub-packages
goda tree ./...

# print stats while building a go program
go build -a --toolexec "goda exec" .
```
