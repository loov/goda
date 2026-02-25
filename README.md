# Goda

Goda is a Go dependency analysis toolkit. It contains tools to figure out what your program is using.

_Note: the exact syntax of the command line arguments has not yet been finalized. So expect some changes to it._

### Installation

To install, you'll need a recent version of Go and then you can install via:

```
go install github.com/loov/goda@latest
```

The commands assume that your GOBIN is reachable on path.

The graph visualizations require [GraphViz](https://graphviz.org/) for rendering the graph.

## Cool things it can do

```
# All of the commands should be run in the cloned repository.
git clone https://github.com/loov/goda && cd goda

# draw a graph of packages in github.com/loov/goda
goda graph "github.com/loov/goda/..." | dot -Tsvg -o graph.svg

# draw a graph of command github.com/loov/goda, within the module
goda graph "github.com/loov/goda:mod" | dot -Tsvg -o graph.svg

# draw a dependency graph of github.com/loov/goda and dependencies
goda graph -cluster -short "github.com/loov/goda:all" | dot -Tsvg -o graph.svg

# list direct dependencies of github.com/loov/goda
goda list "github.com/loov/goda/...:import"

# list dependency graph that reaches flag package, including std
goda graph -std "reach(github.com/loov/goda/...:all, flag)" | dot -Tsvg -o graph.svg

# list packages shared by github.com/loov/goda/pkgset and github.com/loov/goda/cut
goda list "shared(github.com/loov/goda/pkgset:all, github.com/loov/goda/cut:all)"

# list packages that are only imported for tests
goda list "github.com/loov/goda/...:+test:all - github.com/loov/goda/...:all"

# list packages that are imported with `purego` tag
goda list -std "purego=1(github.com/loov/goda/...:all)"

# list packages that are imported for windows and not linux
goda list "goos=windows(github.com/loov/goda/...:all) - goos=linux(github.com/loov/goda/...:all)"

# list how much memory each symbol in the final binary is taking
goda weight -h $GOPATH/bin/goda

# show the impact of cutting a package
goda cut ./...:all

# print dependency tree of all sub-packages
goda tree ./...:all

# print stats while building a go program
go build -a --toolexec "goda exec" .

# list dependency graph in same format as "go mod graph"
goda graph -type edges -f '{{.ID}}{{if .Module}}{{with .Module.Version}}@{{.}}{{end}}{{end}}' ./...:all
```

Maybe you noticed that it's using some weird symbols on the command-line while specifying packages. They allow for more complex scenarios.

The basic syntax is that you can specify multiple packages:

```
goda list github.com/loov/goda/... github.com/loov/qloc
```

By default it will select all the specific packages. You can select the package's direct dependencies with `:import`, direct and indirect dependencies with `:import:all`, the package and all of its direct and indirect dependencies with `:all`:

```
goda list github.com/loov/goda/...:import
goda list github.com/loov/goda/...:import:all
goda list github.com/loov/goda/...:all
```

You can also do basic arithmetic with these sets. For example, if you wish to ignore all `golang.org/x/tools` dependencies:

```
goda list github.com/loov/goda/...:all - golang.org/x/tools/...
```

To get more help about expressions or formatting:

```
goda help expr
goda help format
```

### Reducing binary sizes

One of the main purposes of `goda` is to help reduce binary sizes and dependencies.
Most of the tooling has been designed with that goal in mind.

A few commands deserve special mention, beyond graph inspection:

To find packages with the fewest dependents and the largest size impact:

```
goda cut ./...:all
```

To inspect binary size by summing up symbol sizes per package:

```
goda weight <binary>
```

To find packages that prevent dead code elimination (e.g. due to `reflect.Value.MethodByName`):

```
goda graph -std "deadcode(github.com/loov/goda/...)"
```

To find out why binary size changed between multiple versions:

```
goda weight-diff <1> <2> <3>
```

_Note: Go 1.26 and older Go versions may list some symbols that are not present in the
binary size, due to [Issue #77301](https://github.com/golang/go/issues/77301)._

## Graph example

Here's an example output for:

```
git clone https://github.com/loov/goda && cd goda
goda graph github.com/loov/goda:mod | dot -Tsvg -o graph.svg
```

![github.com/loov/goda dependency graph](./graph.svg)

## How it differs from `go list` or `go mod`

`go list` and `go mod` are tightly integrated with Go and can answer simple queries with compatibility. They also serves as good building blocks for other tools.

`goda` is intended for more complicated queries and analysis. Some of the features can be reproduced by format flags and scripts. However, this library aims to make even complicated analysis fast.

Also, `goda` can be used together with `go list` and `go mod`.
