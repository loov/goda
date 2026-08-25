package templates

import (
	"regexp"
	"text/template"
)

func Parse(t string) (*template.Template, error) {
	return template.New("").Funcs(numericFuncs()).Funcs(stringFuncs()).Parse(t)
}

var rxHeader = regexp.MustCompile(`(\{\{\s*\.?|\s*\}\})`)

// Header derives a table header from a format string,
// e.g. "{{.ID}}\t{{.Cut.Go.Lines}}" -> "ID\tCut.Go.Lines".
func Header(format string) string {
	return rxHeader.ReplaceAllString(format, "")
}
