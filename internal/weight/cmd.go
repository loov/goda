package weight

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/google/subcommands"

	"github.com/loov/goda/internal/memory"
	"github.com/loov/goda/internal/weight/nm"
)

type Command struct {
	limit      int
	sort       Order
	cumulative bool
	humanized  bool
	minimum    int
}

type Order string

const (
	Default     Order = ""
	BySize      Order = "size"
	ByTotalSize Order = "totalsize"
	ByName      Order = "name"
)

func (mode *Order) Set(v string) error {
	switch Order(strings.ToLower(v)) {
	case Default:
		*mode = Default
	case BySize:
		*mode = BySize
	case ByTotalSize:
		*mode = ByTotalSize
	case ByName:
		*mode = ByName
	default:
		return fmt.Errorf("unsupported order %q", v)
	}
	return nil
}
func (mode *Order) String() string { return string(*mode) }

func (*Command) Name() string     { return "weight" }
func (*Command) Synopsis() string { return "Analyse binary symbols." }
func (*Command) Usage() string {
	return `weight <binary>:
	Analyse binary symbols.
`
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.IntVar(&cmd.limit, "limit", -1, "limit number of entries to print")
	f.Var(&cmd.sort, "sort", "sorting mode (size, totalsize, name)")
	f.BoolVar(&cmd.cumulative, "cum", false, "print cumulative size (deprecated, use -sort)")
	f.BoolVar(&cmd.humanized, "h", false, "humanized size output")

	f.IntVar(&cmd.minimum, "minimum", 1024, "minimum size to print")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "missing binary argument\n")
		return subcommands.ExitUsageError
	}

	syms, err := nm.ParseBinary(f.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading syms failed: %v\n", err)
		return subcommands.ExitFailure
	}

	root := NewTree("")
	for _, sym := range syms {
		root.Insert(sym, "", sym.Path)
	}

	trees := []*Tree{root}

	var recurse func(tree *Tree)
	recurse = func(tree *Tree) {
		trees = append(trees, tree.Childs...)
		for _, child := range tree.Childs {
			recurse(child)
		}
	}
	recurse(root)

	if cmd.sort == "" && cmd.cumulative {
		cmd.sort = ByTotalSize
	}

	sorter, ok := sortTreeFunc[cmd.sort]
	if !ok {
		fmt.Fprintf(os.Stderr, "invalid sorting mode %q\n", cmd.sort)
		return subcommands.ExitFailure
	}

	root.Sort(sorter, sortSymFunc[cmd.sort])
	sorter(trees)

	if cmd.limit > 0 && cmd.limit > len(trees) {
		trees = trees[:cmd.limit]
	}

	sizeToString := func(v int64) string {
		return strconv.Itoa(int(v))
	}
	if cmd.humanized {
		sizeToString = memory.ToString
	}

	for _, tree := range trees {
		if tree.TotalSize < int64(cmd.minimum) {
			continue
		}

		fmt.Fprintf(os.Stdout, "%10s %10s %v [syms %d]\n", sizeToString(tree.TotalSize), sizeToString(tree.Size), tree.Path, len(tree.Syms))
		for _, sym := range tree.Syms {
			if sym.Size < int64(cmd.minimum) {
				continue
			}
			fmt.Fprintf(os.Stdout, "%10s %10s %v %v\n", "", sizeToString(sym.Size), string(sym.Code), sym.Name)
		}
	}

	return subcommands.ExitSuccess
}

var sortTreeFunc = map[Order]func([]*Tree){
	Default: sortBySize,
	BySize:  sortBySize,
	ByTotalSize: func(trees []*Tree) {
		sort.Slice(trees, func(i, k int) bool { return trees[i].TotalSize > trees[k].TotalSize })
	},
	ByName: func(trees []*Tree) {
		sort.Slice(trees, func(i, k int) bool { return trees[i].Path < trees[k].Path })
	},
}

var sortSymFunc = map[Order]func([]*nm.Sym){
	Default:     sortBySymSize,
	BySize:      sortBySymSize,
	ByTotalSize: sortBySymSize,
	ByName: func(syms []*nm.Sym) {
		sort.Slice(syms, func(i, k int) bool { return syms[i].Name < syms[k].Name })
	},
}

func sortBySize(trees []*Tree) {
	sort.Slice(trees, func(i, k int) bool {
		if trees[i].Size == trees[k].Size {
			return trees[i].TotalSize > trees[k].TotalSize
		}
		return trees[i].Size > trees[k].Size
	})
}

func sortBySymSize(syms []*nm.Sym) {
	sort.Slice(syms, func(i, k int) bool { return syms[i].Size > syms[k].Size })
}

type Tree struct {
	Path   string
	Name   string
	Lookup map[string]*Tree
	Childs []*Tree

	TotalSize int64

	Size int64
	Syms []*nm.Sym
}

func NewTree(name string) *Tree {
	return &Tree{
		Name:   name,
		Lookup: make(map[string]*Tree),
	}
}

func (tree *Tree) Insert(sym *nm.Sym, parent string, suffix []string) {
	tree.TotalSize += sym.Size

	if len(suffix) == 0 {
		tree.Size += sym.Size
		tree.Syms = append(tree.Syms, sym)
		return
	}

	name := suffix[0]

	subtree, ok := tree.Lookup[name]
	if !ok {
		subtree = NewTree(name)
		subtree.Path = parent + "/" + name
		tree.Lookup[subtree.Name] = subtree
		tree.Childs = append(tree.Childs, subtree)
	}

	subtree.Insert(sym, subtree.Path, suffix[1:])
}

func (tree *Tree) Sort(sortfn func([]*Tree), sortSyms func([]*nm.Sym)) {
	sortfn(tree.Childs)
	for _, child := range tree.Childs {
		child.Sort(sortfn, sortSyms)
	}

	sortSyms(tree.Syms)
}
