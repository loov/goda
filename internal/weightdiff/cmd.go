package weightdiff

import (
	"context"
	"flag"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"text/tabwriter"

	"github.com/google/subcommands"

	"github.com/loov/goda/internal/memory"
	"github.com/loov/goda/internal/weight/nm"
)

type Command struct {
	humanized bool
	miss      bool
	minimum   int64
	allsyms   bool
}

func (*Command) Name() string     { return "weight-diff" }
func (*Command) Synopsis() string { return "Compare binary symbol sizes. (Experimental)" }
func (*Command) Usage() string {
	return `weight-diff <binary1> <binary2> <binary3>:
	Compare binary sizes and the differences.
	(Experimental)
`
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&cmd.humanized, "h", false, "humanized size output")
	f.BoolVar(&cmd.miss, "miss", false, "include missing entries")
	f.Int64Var(&cmd.minimum, "minimum", 1024, "minimum size difference to print")
	f.BoolVar(&cmd.allsyms, "all", false, "include all symbols (e.g. BSS symbols)")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "missing binary arguments\n")
		return subcommands.ExitUsageError
	}

	binaries := []string{}
	symnameSet := map[string]struct{}{}
	symSets := []map[string]*nm.Sym{}
	for _, binary := range f.Args() {
		syms, err := nm.ParseBinary(binary)
		if err != nil {
			fmt.Fprintf(os.Stderr, "loading syms failed: %v\n", err)
			return subcommands.ExitFailure
		}

		if !cmd.allsyms {
			syms = slices.DeleteFunc(syms, func(sym *nm.Sym) bool {
				return !sym.Code.ConsumesBinary()
			})
		}

		symset := map[string]*nm.Sym{}
		for _, sym := range syms {
			symnameSet[sym.QualifiedName] = struct{}{}
			symset[sym.QualifiedName] = sym
		}

		binaries = append(binaries, binary)
		symSets = append(symSets, symset)
	}

	symnames := []string{}
	for symname := range symnameSet {
		symnames = append(symnames, symname)
	}
	sort.Strings(symnames)

	type Row struct {
		QualifiedName string

		Diff int64
		Syms []*nm.Sym
	}

	rows := []Row{}
	for _, symname := range symnames {
		row := Row{
			QualifiedName: symname,
		}

		count := 0
		min, max := int64(0), int64(0)
		for _, xs := range symSets {
			sym := xs[symname]
			row.Syms = append(row.Syms, sym)
			if sym != nil {
				if count == 0 {
					min, max = sym.Size, sym.Size
				} else {
					if sym.Size < min {
						min = sym.Size
					}
					if sym.Size > max {
						max = sym.Size
					}
				}
				count++
			} else {
				min = 0
			}
		}

		if count == 1 {
			row.Diff = max
		} else {
			row.Diff = max - min
		}
		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, k int) bool {
		a, b := &rows[i], &rows[k]
		return abs(a.Diff) > abs(b.Diff)
	})

	sizeToString := func(v int64) string {
		return strconv.Itoa(int(v))
	}
	if cmd.humanized {
		sizeToString = memory.ToString
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)
	defer func() { _ = w.Flush() }()

	fmt.Fprintf(w, "name\tdiff")
	for _, bin := range binaries {
		fmt.Fprintf(w, "\t%v", bin)
	}
	fmt.Fprintf(w, "\n")

	for _, row := range rows {
		if abs(row.Diff) < cmd.minimum {
			continue
		}

		fmt.Fprintf(w, "%v\t%v", row.QualifiedName, sizeToString(row.Diff))
		for _, sym := range row.Syms {
			if sym == nil {
				fmt.Fprintf(w, "\t-")
			} else {
				fmt.Fprintf(w, "\t%v", sizeToString(sym.Size))
			}
		}
		fmt.Fprintf(w, "\n")
	}

	return subcommands.ExitSuccess
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
