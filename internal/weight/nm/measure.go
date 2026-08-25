package nm

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Code represents the type of a symbol in the nm output.
type Code rune

const (
	CodeText          Code = 'T' // text (code) section
	CodeTextLocal     Code = 't' // text (code) section, local
	CodeData          Code = 'D' // initialized data section
	CodeDataLocal     Code = 'd' // initialized data section, local
	CodeBSS           Code = 'B' // uninitialized data (BSS) section
	CodeBSSLocal      Code = 'b' // uninitialized data (BSS) section, local
	CodeReadOnly      Code = 'R' // read-only data section
	CodeReadOnlyLocal Code = 'r' // read-only data section, local
	CodeUndefined     Code = 'U' // undefined symbol
	CodeCommon        Code = 'C' // common symbol (uninitialized data)
	CodeWeak          Code = 'W' // weak symbol
	CodeWeakLocal     Code = 'w' // weak symbol, local
)

// ConsumesBinary returns true if the symbol consumes binary space.
func (code Code) ConsumesBinary() bool {
	switch code {
	case CodeText, CodeTextLocal,
		CodeData, CodeDataLocal,
		CodeReadOnly, CodeReadOnlyLocal:
		return true
	default:
		return false
	}
}

type Sym struct {
	Addr uint64
	Size int64
	Code Code // nm code (T for text, D for data, and so on)

	QualifiedName string

	Path []string
	Name string
}

func (sym *Sym) MaybeSize() int64 {
	if sym == nil {
		return 0
	}
	return sym.Size
}

func ParseBinary(binary string) ([]*Sym, error) {
	command := exec.Command("go", "tool", "nm", "-size", binary)

	var stderr bytes.Buffer
	command.Stderr = &stderr

	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("nm failed: %w: %s", err, stderr.String())
	}

	var syms []*Sym
	for line := range strings.Lines(string(output)) {
		sym, err := parseLine(strings.TrimSuffix(line, "\n"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse: %w", err)
		}
		if sym.QualifiedName == "" {
			continue
		}

		if strings.HasPrefix(sym.QualifiedName, "go:itab.") || strings.HasPrefix(sym.QualifiedName, "type:.") {
			continue
		}

		syms = append(syms, sym)
	}

	return syms, nil
}

func parseLine(s string) (*Sym, error) {
	var err error
	sym := &Sym{}

	tokens := strings.Fields(s)
	if len(tokens) < 2 {
		return nil, fmt.Errorf("invalid sym text: %q", s)
	}

	isSymType := func(s string) bool {
		return len(s) == 1 && (unicode.IsLetter(rune(s[0])) || s[0] == '_' || s[0] == '?')
	}

	// "[addr] size type name", where name may contain spaces.
	var addrField, sizeField, typeField string
	rest := s
	switch {
	case isSymType(tokens[1]):
		sizeField, typeField = tokens[0], tokens[1]
		rest = skipFields(rest, 2)
	case isSymType(tokens[2]):
		addrField, sizeField, typeField = tokens[0], tokens[1], tokens[2]
		rest = skipFields(rest, 3)
	default:
		return nil, fmt.Errorf("unable to find type in sym: %q", s)
	}
	sym.QualifiedName = strings.TrimSpace(rest)

	if addrField != "" {
		sym.Addr, err = strconv.ParseUint(addrField, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid addr: %q", addrField)
		}
	}

	sym.Size, err = strconv.ParseInt(sizeField, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid size %q: %q", s, sizeField)
	}
	// ignore external sym size
	if sym.Size == 4294967296 {
		sym.Size = 0
	}

	tmp, _ := utf8.DecodeRuneInString(typeField)
	sym.Code = Code(tmp)

	if sym.QualifiedName == "" {
		return sym, nil
	}

	// package path ends before the first '(' or '[' (receiver or type args)
	braceOff := len(sym.QualifiedName)
	if i := strings.IndexAny(sym.QualifiedName, "(["); i >= 0 {
		braceOff = i
	}

	slashPos := max(strings.LastIndexByte(sym.QualifiedName[:braceOff], '/'), 0)
	pointOff := max(strings.IndexByte(sym.QualifiedName[slashPos:braceOff], '.'), 0)

	p := slashPos + pointOff
	if p > 0 {
		sym.Path = strings.Split(sym.QualifiedName[:p], "/")
		sym.Name = sym.QualifiedName[p+1:]
	} else {
		sym.Name = sym.QualifiedName
	}

	return sym, nil
}

// skipFields drops the first n whitespace separated fields from s.
func skipFields(s string, n int) string {
	for range n {
		s = strings.TrimLeftFunc(s, unicode.IsSpace)
		if i := strings.IndexFunc(s, unicode.IsSpace); i >= 0 {
			s = s[i:]
		} else {
			s = ""
		}
	}
	return s
}
