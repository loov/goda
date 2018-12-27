# Goda

Goda is a Go dependency analysis toolkit. It contains a bunch of different things to figure out what your program is using.

_Note: the exact syntax of the command line arguments has not yet been finalized. So expect some changes to it._

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

Maybe you noticed that it's using some weird symbols on the command-line while specifying packages. They allow to specify more complex scenarios.

The basic syntax is that you can specify multiple packages:

```
goda list github.com/loov/goda/... github.com/loov/qloc
```

By default it will select all the packages and dependencies of those packages. You can select only the packages with `$` and only the dependencies with `@`. For example:

```
goda list github.com/loov/goda/... @
goda list github.com/loov/goda/... $
```

You can also do basic arithmetic with these sets. For example, if you wish to ignore all `golang.org/x/tools` things in your output you can write:

```
goda list github.com/loov/goda/... - golang.org/x/tools/...
```

There's also `+` which allows to list the shared dependencies:

```
goda list github.com/loov/goda/exec + github.com/loov/goda/graph
```

All of these can of course be combined:

```
# list packages used by github.com/loov/goda
# excluding golang.org/x/tools/..., but not their dependencies
goda list github.com/loov/goda/... @ - golang.org/x/tools/... $
```

## How it differs from `go list` or `go mod`

`go list` and `go mod` are tightly integrated with Go and can answer simple queries with compatibility. It also serves as good building blocks for other tools.

`goda` is intended for more complicated queries and analysis. Some of the features can be reproduced by format flags and scripts, however this is an extra step when you want to quickly figure out different parts.

Of course, it should go without saying, they can be used together.