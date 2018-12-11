package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

func main() {
	count := flag.Int("count", 0, "maximum number to show")
	cumulative := flag.Bool("cum", false, "sort in cumulative order")
	min := flag.Int("min", 1024, "minimum size to print")

	flag.Parse()
	cmd := exec.Command("go", "tool", "nm", "-size", flag.Arg(0))

	reader, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("failed to get pipe: %v", err)
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		log.Printf("failed to load: %v", err)
		os.Exit(1)
	}

	root := NewTree("")

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		sym, err := ParseSym(scanner.Text())
		if err != nil {
			log.Printf("failed to parse: %v", err)
			os.Exit(1)
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
		log.Printf("process failed: %v", err)
		os.Exit(1)
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
	if *cumulative {
		sorter = SortByTotalSize
	} else {
		sorter = SortBySize
	}

	root.Sort(sorter)
	sorter(trees)

	if *count > 0 && *count > len(trees) {
		trees = trees[:*count]
	}
	for _, tree := range trees {
		if tree.TotalSize < uint64(*min) {
			continue
		}

		fmt.Printf("%10d %10d %v [syms %d]\n", tree.TotalSize, tree.Size, tree.Path, len(tree.Syms))
		for _, sym := range tree.Syms {
			if sym.Size < uint64(*min) {
				continue
			}
			fmt.Printf("%10s %10d %v %v\n", "", sym.Size, string(sym.Code), sym.Name)
		}
	}
}

func SortByTotalSize(trees []*Tree) {
	sort.Slice(trees, func(i, k int) bool { return trees[i].TotalSize > trees[k].TotalSize })
}

func SortBySize(trees []*Tree) {
	sort.Slice(trees, func(i, k int) bool { return trees[i].Size > trees[k].Size })
}

type Tree struct {
	Path   string
	Name   string
	Lookup map[string]*Tree
	Childs []*Tree

	TotalSize uint64

	Size uint64
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
	Size uint64
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
		sym.Size, err = strconv.ParseUint(size, 10, 64)
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
