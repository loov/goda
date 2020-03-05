package pkgset

import (
	"context"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Context struct {
	Context context.Context
	Tags    Strings
	Env     Strings
}

func (ctx Context) Clone() *Context {
	return &Context{
		Context: ctx.Context,
		Tags:    ctx.Tags.Clone(),
		Env:     ctx.Env.Clone(),
	}
}

func (ctx Context) Load(patterns ...string) ([]*packages.Package, error) {
	return packages.Load(ctx.Config(), patterns...)
}

var envvars = map[string]struct{}{
	"GOOS":        {},
	"GOARCH":      {},
	"GOENV":       {},
	"GOFLAGS":     {},
	"GOROOT":      {},
	"CGO_ENABLED": {},
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
		Mode:    packages.LoadImports,
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

func KeyValue(s string) (string, string) {
	p := strings.LastIndexByte(s, '=')
	if p < 0 {
		return s, ""
	}
	return s[:p], s[p+1:]
}
