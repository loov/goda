package weightdiff

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/google/subcommands"

	"github.com/loov/goda/internal/memory"
	"github.com/loov/goda/internal/weight/nm"
)

type Command struct {
	humanized  bool
	miss       bool
	minimum    int64
	allsyms    bool
	color      bool
	cumulative bool
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
	f.Int64Var(&cmd.minimum, "minimum", 1024, "minimum abs(total delta) difference to print")
	f.BoolVar(&cmd.allsyms, "all", false, "include all symbols (e.g. BSS symbols)")
	f.BoolVar(&cmd.color, "color", false, "color delta based on sign")
	f.BoolVar(&cmd.cumulative, "cum", false, "include cumulative total of deltas")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "missing binary arguments\n")
		return subcommands.ExitUsageError
	}

	binaries := []string{}
	aliases := []string{}
	symnameSet := map[string]struct{}{}
	symSets := []map[string]*nm.Sym{}
	for _, binary := range f.Args() {
		alias := binary
		if a, b, ok := strings.Cut(binary, "="); ok {
			alias = a
			binary = b
		}

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
		aliases = append(aliases, alias)
		symSets = append(symSets, symset)
	}

	symnames := []string{}
	for symname := range symnameSet {
		symnames = append(symnames, symname)
	}
	sort.Strings(symnames)

	type Cell struct {
		*nm.Sym
		Delta int64 // pre + delta = cur
	}
	type Row struct {
		QualifiedName string

		Cells []Cell
		Delta []int64

		TotalDelta  int64
		MaxAbsDelta int64
	}

	rows := []Row{}
	for _, symname := range symnames {
		row := Row{
			QualifiedName: symname,
		}

		lastSize := int64(0)
		for _, xs := range symSets {
			sym := xs[symname]
			size := sym.MaybeSize()
			row.Cells = append(row.Cells, Cell{
				Sym:   sym,
				Delta: size - lastSize,
			})
			lastSize = size
		}

		for _, cell := range row.Cells[1:] {
			if cell.Sym != nil {
				row.MaxAbsDelta = max(row.MaxAbsDelta, abs(cell.Delta))
			}
		}
		row.TotalDelta = row.Cells[len(row.Cells)-1].MaybeSize() - row.Cells[0].MaybeSize()

		rows = append(rows, row)
	}

	slices.SortFunc(rows, func(a, b Row) int {
		r := cmp.Compare(abs(a.TotalDelta), abs(b.TotalDelta))
		if r != 0 {
			return -r
		}
		return cmp.Compare(a.QualifiedName, b.QualifiedName)
	})

	sizeToString := func(v int64) string {
		return strconv.Itoa(int(v))
	}
	if cmd.humanized {
		sizeToString = memory.ToStringShort
	}

	symSizeToString := func(sym *nm.Sym) string {
		if sym == nil {
			return "-"
		}
		return sizeToString(sym.Size)
	}

	deltaToString := func(v int64) string {
		if v == 0 {
			return "~"
		} else if v > 0 {
			if cmd.color {
				return "\x1b[7m" + "+" + sizeToString(v) + "\x1b[0m"
			}
			return "+" + sizeToString(v)
		} else {
			return sizeToString(v)
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)
	defer func() { _ = w.Flush() }()

	fmt.Fprintf(w, "name")
	for i, bin := range aliases {
		if i == 0 {
			fmt.Fprintf(w, "\t%7v", bin)
		} else {
			fmt.Fprintf(w, "\t%7v\t  delta", bin)
		}
	}
	if len(binaries) > 2 {
		fmt.Fprintf(w, "\ttotal ∆")
	}
	if cmd.cumulative {
		fmt.Fprintf(w, "\tcum. ∆")
	}

	fmt.Fprintf(w, "\n")

	cumulative := int64(0)
	for _, row := range rows {
		if abs(row.TotalDelta) < cmd.minimum {
			continue
		}
		cumulative += row.TotalDelta

		fmt.Fprintf(w, "%v", row.QualifiedName)
		for i, cell := range row.Cells {
			if i == 0 {
				fmt.Fprintf(w, "\t%7v", symSizeToString(cell.Sym))
			} else {
				fmt.Fprintf(w, "\t%7v\t%8v", symSizeToString(cell.Sym), deltaToString(cell.Delta))
			}
		}
		if len(binaries) > 2 {
			fmt.Fprintf(w, "\t%8v", deltaToString(row.TotalDelta))
		}
		if cmd.cumulative {
			fmt.Fprintf(w, "\t%8v", deltaToString(cumulative))
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
