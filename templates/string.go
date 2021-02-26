package templates

import (
	"strings"
	"text/template"
)

func stringFuncs() template.FuncMap {
	return template.FuncMap{
		"rel":    rel,
		"rename": rename,
	}
}

func rel(args ...string) string {
	if len(args) == 0 {
		return ""
	}
	id, prefixes := args[len(args)-1], args[:len(args)-1]

	for _, prefix := range prefixes {
		if x, ok := replace(id, prefix, "./"); ok {
			return x
		}
	}
	return id
}

func rename(args ...string) string {
	if len(args) == 0 {
		return ""
	}
	id, replacements := args[len(args)-1], args[:len(args)-1]

	for i := 0; i < len(replacements); i += 2 {
		prefix := replacements[i]
		replacement := "./"
		if i+1 < len(replacements) {
			replacement = replacements[i+1]
		}
		if x, ok := replace(id, prefix, replacement); ok {
			return x
		}
	}
	return id
}

func replace(id string, prefix, replacement string) (string, bool) {
	if id == prefix {
		return replacement, true
	}

	prefix = withSlash(prefix)
	if strings.HasPrefix(id, prefix) {
		id = strings.TrimPrefix(id, prefix)
		replacement = withSlash(replacement)
		return replacement + id, true
	}
	return id, false
}

func withSlash(prefix string) string {
	if !strings.HasSuffix(prefix, "/") {
		return prefix + "/"
	}
	return prefix
}
