package nm

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/subcommands"
	"github.com/loov/goda/memory"
)

type Command struct {
	limit      int
	cumulative bool
	humanized  bool
	minimum    int
}

func (*Command) Name() string     { return "nm" }
func (*Command) Synopsis() string { return "Analyse binary symbols." }
func (*Command) Usage() string {
	return `nm <binary>:
	Analyse binary symbols.
`
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.IntVar(&cmd.limit, "limit", -1, "limit number of entries to print")
	f.BoolVar(&cmd.cumulative, "cum", false, "print cumulative size")
	f.BoolVar(&cmd.humanized, "h", false, "humanized size output")

	f.IntVar(&cmd.minimum, "minimum", 1024, "minimum size to print")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "missing binary argument\n")
		return subcommands.ExitUsageError
	}

	command := exec.Command("go", "tool", "nm", "-size", f.Arg(0))

	reader, err := command.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get stdout: %v\n", err)
		return subcommands.ExitFailure
	}

	if err := command.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start: %v\n", err)
		return subcommands.ExitFailure
	}

	root := NewTree("")

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		sym, err := ParseSym(scanner.Text())
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse: %v\n", err)
			return subcommands.ExitFailure
		}

		if len(sym.Path) > 0 && strings.HasPrefix(sym.Path[0], "go.itab.") {
			continue
		}
		if len(sym.Path) > 0 && strings.HasPrefix(sym.Path[0], "type..") {
			continue
		}

		root.Insert(sym, "", sym.Path)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "process failed: %v\n", err)
		return subcommands.ExitFailure
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

	var sorter func([]*Tree)
	if cmd.cumulative {
		sorter = sortByTotalSize
	} else {
		sorter = sortBySize
	}

	root.Sort(sorter)
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

		fmt.Fprintf(os.Stderr, "%10s %10s %v [syms %d]\n", sizeToString(tree.TotalSize), sizeToString(tree.Size), tree.Path, len(tree.Syms))
		for _, sym := range tree.Syms {
			if sym.Size < int64(cmd.minimum) {
				continue
			}
			fmt.Fprintf(os.Stderr, "%10s %10s %v %v\n", "", sizeToString(sym.Size), string(sym.Code), sym.Name)
		}
	}

	return subcommands.ExitSuccess
}

func sortByTotalSize(trees []*Tree) {
	sort.Slice(trees, func(i, k int) bool { return trees[i].TotalSize > trees[k].TotalSize })
}

func sortBySize(trees []*Tree) {
	sort.Slice(trees, func(i, k int) bool { return trees[i].Size > trees[k].Size })
}

type Tree struct {
	Path   string
	Name   string
	Lookup map[string]*Tree
	Childs []*Tree

	TotalSize int64

	Size int64
	Syms []*Sym
}

func NewTree(name string) *Tree {
	return &Tree{
		Name:   name,
		Lookup: make(map[string]*Tree),
	}
}

func (tree *Tree) Insert(sym *Sym, parent string, suffix []string) {
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

func (tree *Tree) Sort(sortfn func([]*Tree)) {
	sortfn(tree.Childs)
	for _, child := range tree.Childs {
		child.Sort(sortfn)
	}

	sort.Slice(tree.Syms, func(i, k int) bool { return tree.Syms[i].Size > tree.Syms[k].Size })
}

type Sym struct {
	Addr uint64
	Size int64
	Code rune // nm code (T for text, D for data, and so on)

	QualifiedName string
	Info          string

	Path []string
	Name string
}

func ParseSym(s string) (*Sym, error) {
	var err error
	sym := &Sym{}

	tokens := strings.Fields(s[8:])
	if len(tokens) < 3 {
		return nil, fmt.Errorf("invalid sym text: %q", s)
	}

	if addr := strings.TrimSpace(s[:8]); addr != "" {
		sym.Addr, err = strconv.ParseUint(addr, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid addr: %q", addr)
		}
	}

	if size := strings.TrimSpace(tokens[0]); size != "" {
		sym.Size, err = strconv.ParseInt(size, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid size: %q", size)
		}
	}

	if code := strings.TrimSpace(tokens[1]); code != "" {
		sym.Code, _ = utf8.DecodeRuneInString(code)
	}

	sym.QualifiedName = tokens[2]
	sym.Info = strings.Join(tokens[3:], " ")

	braceOff := strings.IndexByte(sym.QualifiedName, '(')
	if braceOff < 0 {
		braceOff = len(sym.QualifiedName)
	}

	slashPos := strings.LastIndexByte(sym.QualifiedName[:braceOff], '/')
	if slashPos < 0 {
		slashPos = 0
	}

	pointOff := strings.IndexByte(sym.QualifiedName[slashPos:braceOff], '.')
	if pointOff < 0 {
		pointOff = 0
	}

	p := slashPos + pointOff
	if p > 0 {
		sym.Path = strings.Split(sym.QualifiedName[:p], "/")
		sym.Name = sym.QualifiedName[p+1:]
	} else {
		sym.Name = sym.QualifiedName
	}

	return sym, nil
}
