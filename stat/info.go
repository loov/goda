package stat

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"

	"golang.org/x/tools/go/packages"
)

type Stat struct {
	Go    Source
	Other Source

	Decls  Decls
	Tokens Tokens
}

func (info *Stat) AllFiles() Source {
	var c Source
	c.Add(info.Go)
	c.Add(info.Other)
	return c
}

func (s *Stat) Add(b Stat) {
	s.Go.Add(b.Go)
	s.Other.Add(b.Other)
	s.Decls.Add(b.Decls)
	s.Tokens.Add(b.Tokens)
}

func Package(p *packages.Package) (Stat, []error) {
	var info Stat
	var errs []error

	fset := token.NewFileSet()

	for _, filename := range p.GoFiles {
		src, err := ioutil.ReadFile(filename)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to read %q: %w", filename, err))
			continue
		}

		count := SourceFromBytes(src)
		info.Go.Add(count)

		f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse %q: %w", filename, err))
			continue
		}

		info.Decls.Add(DeclsFromAst(f))
		info.Tokens.Add(TokensFromAst(f))
	}

	for _, filename := range p.OtherFiles {
		count, err := SourceFromPath(filename)
		info.Other.Add(count)
		if err != nil {
			if !errors.Is(err, ErrEmptyFile) {
				errs = append(errs, err)
			}
			continue
		}
	}

	return info, errs
}
