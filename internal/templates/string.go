package templates

import (
	"encoding/json"
	"strings"
	"text/template"
)

func stringFuncs() template.FuncMap {
	return template.FuncMap{
		"rel":    rel,
		"rename": rename,
		"json":   jsonify,
	}
}

func rel(prefix, id string) string {
	if x, ok := replace(id, prefix, "./"); ok {
		return x
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
	if after, ok := strings.CutPrefix(id, prefix); ok {
		id = after
		if replacement != "" {
			replacement = withSlash(replacement)
		}
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

func jsonify(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "unable to marshal: " + err.Error()
	}
	return string(data)
}
