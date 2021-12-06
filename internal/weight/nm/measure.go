package nm

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Sym struct {
	Addr uint64
	Size int64
	Code rune // nm code (T for text, D for data, and so on)

	QualifiedName string
	Info          string

	Path []string
	Name string
}

func ParseBinary(binary string) ([]*Sym, error) {
	command := exec.Command("go", "tool", "nm", "-size", binary)

	reader, err := command.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout: %w", err)
	}

	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("failed to start: %w", err)
	}

	var syms []*Sym
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		sym, err := parseLine(scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("failed to parse: %w", err)
		}
		if sym.QualifiedName == "" {
			continue
		}

		if len(sym.Path) > 0 && strings.HasPrefix(sym.Path[0], "go.itab.") {
			continue
		}
		if len(sym.Path) > 0 && strings.HasPrefix(sym.Path[0], "type..") {
			continue
		}

		syms = append(syms, sym)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning failed: %w", err)
	}

	return syms, nil
}

func parseLine(s string) (*Sym, error) {
	var err error
	sym := &Sym{}

	tokens := strings.Fields(s[8:])
	if len(tokens) < 2 {
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

	if len(tokens) >= 3 {
		sym.QualifiedName = tokens[2]
	}
	if len(tokens) >= 4 {
		sym.Info = strings.Join(tokens[3:], " ")
	}
	if sym.QualifiedName == "" {
		return sym, nil
	}

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
