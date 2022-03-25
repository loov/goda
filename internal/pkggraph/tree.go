package pkggraph

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/vcs"
)

func (g *Graph) Tree() (*Tree, error) {
	goModCachePath, err := goModCache()
	if err != nil {
		return nil, err
	}

	t := Tree{
		Repos: make(map[string]*Repo),
	}
	for _, n := range g.Sorted {
		repo := t.NodeRepo(n)

		if n.Module != nil {
			mod := repo.NodeModule(n, goModCachePath)
			mod.NodePackage(n)
		} else {
			repo.NodePackage(n)
		}
	}

	t.Walk(func(tn TreeNode) {
		if s, ok := tn.(interface{ Sort() }); ok {
			s.Sort()
		}
	})

	return &t, nil
}

type TreeNode interface {
	Path() string
	Package() *TreePackage
	VisitChildren(func(TreeNode))
}

type Tree struct {
	Repos       map[string]*Repo
	sortedRepos []string
}

func (t *Tree) Path() string {
	return ""
}

func (t *Tree) Package() *TreePackage {
	return nil
}

func (t *Tree) LookupTable() map[*GraphNode]*TreePackage {
	table := map[*GraphNode]*TreePackage{}
	t.Walk(func(tn TreeNode) {
		if pkg := tn.Package(); pkg != nil {
			table[pkg.GraphNode] = pkg
		}
	})
	return table
}

func (t *Tree) NodeRepo(n *GraphNode) *Repo {
	repo, ok := t.Repos[n.Repo.Root]
	if !ok {
		repo = &Repo{
			Root:    n.Repo,
			Modules: make(map[string]*Module),
		}
		t.Repos[n.Repo.Root] = repo
		t.sortedRepos = append(t.sortedRepos, n.Repo.Root)
	}
	return repo
}

func (t *Tree) Walk(fn func(TreeNode)) {
	fn(t)

	var visit func(TreeNode)
	visit = func(tn TreeNode) {
		fn(tn)
		tn.VisitChildren(visit)
	}
	t.VisitChildren(visit)
}

func (t *Tree) VisitChildren(fn func(TreeNode)) {
	for _, rp := range t.sortedRepos {
		fn(t.Repos[rp])
	}
}

func (t *Tree) Sort() {
	sort.Strings(t.sortedRepos)
}

type Repo struct {
	Root *vcs.RepoRoot

	Modules    map[string]*Module
	sortedMods []string

	Pkgs       map[string]*TreePackage
	sortedPkgs []string
}

func (r *Repo) Path() string {
	return r.Root.Root
}

func (r *Repo) Package() *TreePackage {
	return nil
}

func (r *Repo) SameAsOnlyModule() bool {
	if len(r.Modules) != 1 {
		return false
	}
	mod := r.Modules[r.sortedMods[0]]
	prefix, pathMajor, ok := module.SplitPathVersion(mod.Mod.Path)
	if !ok || r.Path() != prefix {
		return false
	}
	if pathMajor == "" || mod.Local {
		// Local modules will not have a version, assume it is the same without
		// checking the major version matches.
		return true
	}
	return module.CheckPathMajor(mod.Mod.Version, pathMajor) == nil
}

func (r *Repo) NodeModule(n *GraphNode, goModCachePath string) *Module {
	mod, ok := r.Modules[n.Module.Path]
	if !ok {
		mod = &Module{
			Parent: r,
			Mod:    n.Module,
			Pkgs:   make(map[string]*TreePackage),
		}
		if n.Module.Replace == nil {
			if rp, err := filepath.Rel(goModCachePath, n.Module.Dir); err == nil {
				// If the module is in the module cache its path relative to GOMODCACHE
				// will not start with "..". If it does, then it is outside the
				// GOMODCACHE and is likely a local copy of the module.
				mod.Local = strings.HasPrefix(rp, "..")
			}
		}
		r.Modules[n.Module.Path] = mod
		r.sortedMods = append(r.sortedMods, n.Module.Path)
	}
	return mod
}

func (r *Repo) NodePackage(n *GraphNode) *TreePackage {
	pkg, ok := r.Pkgs[n.PkgPath]
	if !ok {
		pkg = &TreePackage{
			Parent:    r,
			GraphNode: n,
		}
		r.Pkgs[n.PkgPath] = pkg
		r.sortedPkgs = append(r.sortedPkgs, n.PkgPath)
	}
	return pkg
}

func (r *Repo) VisitChildren(fn func(TreeNode)) {
	for _, mp := range r.sortedMods {
		fn(r.Modules[mp])
	}

	for _, pp := range r.sortedPkgs {
		fn(r.Pkgs[pp])
	}
}

func (r *Repo) Sort() {
	sort.Strings(r.sortedMods)
	sort.Strings(r.sortedPkgs)
}

type Module struct {
	Parent *Repo

	Mod   *packages.Module
	Local bool

	Pkgs       map[string]*TreePackage
	sortedPkgs []string
}

func (m *Module) Path() string {
	return m.Mod.Path
}

func (m *Module) Package() *TreePackage {
	return nil
}

func (m *Module) NodePackage(n *GraphNode) *TreePackage {
	pkg, ok := m.Pkgs[n.PkgPath]
	if !ok {
		pkg = &TreePackage{
			Parent:    m,
			GraphNode: n,
		}
		m.Pkgs[n.PkgPath] = pkg
		m.sortedPkgs = append(m.sortedPkgs, n.PkgPath)
	}
	return pkg
}

func (m *Module) VisitChildren(fn func(TreeNode)) {
	for _, pp := range m.sortedPkgs {
		fn(m.Pkgs[pp])
	}
}

func (m *Module) Sort() {
	sort.Strings(m.sortedPkgs)
}

type TreePackage struct {
	Parent    TreeNode
	GraphNode *GraphNode
}

func (p *TreePackage) Path() string {
	return p.GraphNode.PkgPath
}

func (p *TreePackage) Package() *TreePackage {
	return p
}

func (p *TreePackage) OnlyChild() bool {
	count := 0
	p.Parent.VisitChildren(func(TreeNode) {
		count++
	})
	return count == 1
}

func (p *TreePackage) VisitChildren(_ func(TreeNode)) {}

func goModCache() (string, error) {
	cmd := exec.Command("go", "env", "GOMODCACHE")
	out, err := cmd.Output()
	switch err := err.(type) {
	case *exec.ExitError:
		return "", fmt.Errorf("failed to determine GOMODCACHE: %s", err.Stderr)
	default:
		return "", fmt.Errorf("failed to determine GOMODCACHE: %s", err)
	case nil:
		// just continue
	}
	return strings.TrimSpace(string(out)), nil
}
