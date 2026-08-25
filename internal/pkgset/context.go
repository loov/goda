package pkgset

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/tools/go/packages"
)

var envvars = map[string]struct{}{
	"GOOS":        {},
	"GOARCH":      {},
	"GOENV":       {},
	"GOFLAGS":     {},
	"GOROOT":      {},
	"CGO_ENABLED": {},
}

var packageAliases = map[string]string{
	"C": "runtime/cgo",
}

func replaceAliases(patterns ...string) []string {
	xs := append([]string{}, patterns...)
	for i, x := range xs {
		if alias, ok := packageAliases[x]; ok {
			xs[i] = alias
		}
	}
	return xs
}

type Context struct {
	Context context.Context
	Tags    Strings
	Env     Strings

	Variables map[string]Set
}

func (ctx Context) Clone() *Context {
	return &Context{
		Context:   ctx.Context,
		Tags:      ctx.Tags.Clone(),
		Env:       ctx.Env.Clone(),
		Variables: ctx.Variables,
	}
}

func (ctx Context) Load(patterns ...string) ([]*packages.Package, error) {
	return load(ctx.Config(), patterns...)
}

func (ctx Context) LoadWithTests(patterns ...string) ([]*packages.Package, error) {
	config := ctx.Config()
	config.Tests = true
	return load(config, patterns...)
}

func (ctx Context) LoadWithoutTests(patterns ...string) ([]*packages.Package, error) {
	config := ctx.Config()
	config.Tests = false
	return load(config, patterns...)
}

// load wraps packages.Load and reports patterns that failed to resolve,
// which packages.Load only records as placeholder packages with a ListError.
func load(config *packages.Config, patterns ...string) ([]*packages.Package, error) {
	roots, err := packages.Load(config, replaceAliases(patterns...)...)
	if err != nil {
		return roots, err
	}
	var errs []error
	for _, p := range roots {
		for _, e := range p.Errors {
			if e.Kind == packages.ListError {
				errs = append(errs, fmt.Errorf("%v: %v", p.ID, e.Msg))
			}
		}
	}
	return roots, errors.Join(errs...)
}

func (ctx *Context) Set(key, value string) {
	if _, ok := envvars[strings.ToUpper(key)]; ok {
		ctx.Env.Set(strings.ToUpper(key), value)
		return
	}
	ctx.Tags.Set(key, value)
}

func (ctx Context) Config() *packages.Config {
	config := &packages.Config{
		Context: ctx.Context,
		Mode:    packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedModule,
		Env:     ctx.Env,
		Tests:   ctx.Tags.ValueOf("test") == "1",
	}

	tags := []string{}
	for _, tag := range ctx.Tags {
		key, value := KeyValue(tag)
		if strings.EqualFold("test", key) {
			continue
		}
		if value == "1" {
			tags = append(tags, key)
		}
	}
	if len(tags) > 0 {
		config.BuildFlags = append(config.BuildFlags, "-tags="+strings.Join(tags, ","))
	}

	return config
}

type Strings []string

func (strs *Strings) Set(key, value string) {
	i := strs.IndexOf(key)
	if i < 0 {
		*strs = append(*strs, key+"="+value)
		return
	}
	(*strs)[i] = key + "=" + value
}

func (strs Strings) ValueOf(key string) string {
	i := strs.IndexOf(key)
	if i < 0 {
		return ""
	}
	_, value := KeyValue(strs[i])
	return value
}

func (strs Strings) IndexOf(key string) int {
	prefix := strings.ToLower(key + "=")
	for i, x := range strs {
		x = strings.ToLower(x)
		if strings.HasPrefix(x, prefix) {
			return i
		}
	}
	return -1
}

func (strs Strings) Clone() Strings {
	return append(Strings{}, strs...)
}

// KeyValue parses s into a key and value.
func KeyValue(s string) (string, string) {
	k, v, _ := strings.Cut(s, "=")
	return k, v
}
