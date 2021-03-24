package templates

import (
	"text/template"
)

func Parse(t string) (*template.Template, error) {
	return template.New("").Funcs(numericFuncs()).Funcs(stringFuncs()).Parse(t)
}
