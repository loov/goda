package templates

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"text/template"

	"github.com/loov/goda/memory"
	"golang.org/x/tools/go/packages"
)

func Parse(t string) (*template.Template, error) {
	return template.New("").Funcs(template.FuncMap{
		"LineCount":  LineCount,
		"SourceSize": SourceSize,
		"AllFiles":   AllFiles,
		"DeclCount":  CountDecls,
	}).Parse(t)
}

func LineCount(vs ...interface{}) int64 {
	var count int64

	for _, filename := range filesFromInterface(vs...) {
		r, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v open failed: %v\n", filename, err)
			continue
		}
		count += countLines(r)

		if err := r.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "%v close failed: %v\n", filename, err)
			continue
		}
	}

	return count
}

func SourceSize(vs ...interface{}) memory.Bytes {
	var size int64

	for _, filename := range filesFromInterface(vs...) {
		stat, err := os.Stat(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v stat failed: %v", filename, err)
			continue
		}
		size += stat.Size()
	}

	return memory.Bytes(size)
}

func AllFiles(vs ...interface{}) (files []string) {
	for _, v := range vs {
		switch v := v.(type) {
		case []string:
			files = append(files, v...)
		case *packages.Package:
			files = append(files, allFiles(v)...)
		case interface{ Pkg() *packages.Package }:
			files = append(files, allFiles(v.Pkg())...)
		}
	}
	return files
}

type DeclCount struct {
	Func  int64
	Type  int64
	Const int64
	Var   int64
	Other int64
}

func (decl DeclCount) Total() int64 {
	return decl.Func + decl.Type + decl.Const + decl.Var + decl.Other
}

func CountDecls(vs ...interface{}) DeclCount {
	var count DeclCount

	for _, v := range vs {
		fset := token.NewFileSet() // positions are relative to fset

		for _, filename := range filesFromInterface(v) {
			src, err := ioutil.ReadFile(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%q reading failed: %v\n", filename, err)
				continue
			}

			f, err := parser.ParseFile(fset, filename, src, 0)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%q parsing failed: %v\n", filename, err)
				continue
			}

			for _, decl := range f.Decls {
				switch decl := decl.(type) {
				case *ast.GenDecl:
					switch decl.Tok {
					case token.TYPE:
						count.Type++
					case token.VAR:
						count.Var++
					case token.CONST:
						count.Const++
					default:
						count.Other++
					}
				case *ast.FuncDecl:
					count.Func++
				default:
					count.Other++
				}
			}
		}
	}

	return count
}

func filesFromInterface(vs ...interface{}) []string {
	var files []string
	for _, v := range vs {
		switch v := v.(type) {
		// assume we want the list of files
		case []string:
			files = append(files, v...)
		// assume we want the all files in package directories
		case *packages.Package:
			files = append(files, v.GoFiles...)
		// assume we want the all files in package directories
		case interface{ Pkg() *packages.Package }:
			files = append(files, v.Pkg().GoFiles...)
		}
	}
	return files
}

func allFiles(p *packages.Package) []string {
	files := map[string]bool{}
	for _, filename := range p.GoFiles {
		files[filename] = true
	}
	for _, filename := range p.OtherFiles {
		files[filename] = true
	}

	var list []string
	for file := range files {
		list = append(list, file)
	}
	sort.Strings(list)

	return list
}

func countLines(r io.Reader) int64 {
	var count int64
	var buffer [1 << 20]byte
	for {
		n, err := r.Read(buffer[:])

		for _, r := range buffer[:n] {
			if r == 0 { // probably a binary file
				return 0
			}
			if r == '\n' {
				count++
			}
		}

		if err != nil || n == 0 {
			return count
		}
	}
}
