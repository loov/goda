package nm

import (
	"bufio"
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
	Info          string

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
		line := scanner.Text()

		sym, err := parseLine(line)
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

	if err := command.Wait(); err != nil {
		return nil, fmt.Errorf("nm failed: %w: %s", err, stderr.String())
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

	addrField := ""
	sizeField := ""
	typeField := ""
	nameField := ""
	infoField := ""

	isSymType := func(s string) bool {
		return len(s) == 1 && (unicode.IsLetter(rune(s[0])) || s[0] == '_' || s[0] == '?')
	}

	switch {
	case isSymType(tokens[1]):
		// in some cases addr is not printed
		sizeField = tokens[0]
		typeField = tokens[1]
		if len(tokens) > 2 {
			nameField = tokens[2]
		}
		if len(tokens) > 3 {
			infoField = strings.Join(tokens[3:], " ")
		}
	case isSymType(tokens[2]):
		addrField = tokens[0]
		sizeField = tokens[1]
		typeField = tokens[2]
		if len(tokens) > 3 {
			nameField = tokens[3]
		}
		if len(tokens) > 4 {
			infoField = strings.Join(tokens[4:], " ")
		}
	default:
		return nil, fmt.Errorf("unable to find type in sym: %q", s)
	}

	if addrField != "" {
		sym.Addr, err = strconv.ParseUint(addrField, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid addr: %q", addrField)
		}
	}

	if sizeField != "" {
		sym.Size, err = strconv.ParseInt(sizeField, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid size %q: %q", s, sizeField)
		}

		// ignore external sym size
		if sym.Size == 4294967296 {
			sym.Size = 0
		}
	}

	if code := strings.TrimSpace(typeField); code != "" {
		tmp, _ := utf8.DecodeRuneInString(code)
		sym.Code = Code(tmp)
	}

	sym.QualifiedName = nameField
	sym.Info = infoField

	if sym.QualifiedName == "" {
		return sym, nil
	}

	braceOff := strings.IndexByte(sym.QualifiedName, '(')
	if braceOff < 0 {
		braceOff = len(sym.QualifiedName)
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
