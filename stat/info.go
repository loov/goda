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
	GoFiles    Source
	OtherFiles Source

	DeclCount  TopDecl
	TokenCount Tokens
}

func (info *Stat) AllFiles() Source {
	var c Source
	c.Add(info.GoFiles)
	c.Add(info.OtherFiles)
	return c
}

func (s *Stat) Add(b Stat) {
	s.GoFiles.Add(b.GoFiles)
	s.OtherFiles.Add(b.OtherFiles)
	s.DeclCount.Add(b.DeclCount)
	s.TokenCount.Add(b.TokenCount)
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
		info.GoFiles.Add(count)

		f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse %q: %w", filename, err))
			continue
		}

		info.DeclCount.Add(TopDeclFromAst(f))
		info.TokenCount.Add(TokensFromAst(f))
	}

	for _, filename := range p.OtherFiles {
		count, err := SourceFromPath(filename)
		info.OtherFiles.Add(count)
		if err != nil {
			if !errors.Is(err, ErrEmptyFile) {
				errs = append(errs, err)
			}
			continue
		}
	}

	return info, errs
}
