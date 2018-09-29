package main

import (
	"fmt"
	"os"

	"github.com/loov/ago/pkg"
	"golang.org/x/tools/go/packages"
)

var commands = map[string]func(*State, ...string){
	// tree prints dependency tree
	"tree": nil,
	// analyze size impact for each imported package
	"size": nil,
	// calculate with package sets
	"calc": nil,
	// time commands (cross-platform time)
	"time": nil,
}

type State struct{}

func main() {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.LoadImports,
		Env:  append(os.Environ(), "GOOS=windows"),
	}, os.Args[1])
	if err != nil {
		panic(err)
	}

	pkg.NewSet()

	/*
		func(fset *token.FileSet, filename string) (*ast.File, error) {
			const mode = parser.AllErrors | parser.ParseComments
			return parser.ParseFile(fset, filename, nil, mode)
		}
	*/

	for _, p := range pkgs {
		fmt.Println(p.ID)
		for _, file := range p.GoFiles {
			fmt.Println(file)
		}
	}
}
