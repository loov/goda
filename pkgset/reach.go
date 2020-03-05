package pkgset

import "golang.org/x/tools/go/packages"

// Reach returns packages in source that terminate in target
func Reach(source, target Set) Set {
	reach := &Reachability{
		Result: New(),

		Source: source,
		Target: target,

		Reaches:     source.Clone(),
		CannotReach: New(),
	}

	reach.Run()

	return reach.Result
}

// ReachWithout returns packages in source that terminate in target,
// but ensures that "blacklist" is not used to find the path.
func ReachWithout(source, target, blacklist Set) Set {
	reach := &Reachability{
		Result: New(),

		Source: source,
		Target: target,

		Blacklist: blacklist.Clone(),

		Reaches:     source.Clone(),
		CannotReach: New(),
	}

	reach.Run()

	return reach.Result
}

// ReachUsing returns packages in source that terminate in target,
// but ensures that paths are only in whitelist.
func ReachUsing(source, target, whitelist Set) Set {
	reach := &Reachability{
		Result: New(),

		Source: source,
		Target: target,

		Whitelist: whitelist.Clone(),

		Reaches:     source.Clone(),
		CannotReach: New(),
	}

	reach.Run()

	return reach.Result
}

// Reachability calculates reachability graph.
type Reachability struct {
	// Result will contain the final set.
	Result Set

	// Whitelist contains list of packages that can be used to find the path.
	Whitelist Set
	// Blacklist contains list of packages that cannot be used to find the path.
	Blacklist Set

	// Source contains the graph we are finding the paths in.
	Source Set
	// Target contains the nodes we are looking the path to.
	Target Set

	// Reaches contains packages that do reach the target.
	Reaches Set
	// CannotReach contains packages that cannot reach the target.
	CannotReach Set
}

// Run finds the reachability graph.
func (reach *Reachability) Run() {
	for _, p := range reach.Source {
		if _, reaches := reach.Target[p.ID]; reaches {
			reach.Result[p.ID] = p
			continue
		}
		reach.Check(p)
	}
}

// Check returns whether p can reach a target.
func (reach *Reachability) Check(p *packages.Package) bool {
	if _, ok := reach.Reaches[p.ID]; ok {
		return true
	}
	if _, ok := reach.CannotReach[p.ID]; ok {
		return false
	}

	// handle blacklist & whitelist
	if reach.Whitelist != nil {
		if _, allowed := reach.Whitelist[p.ID]; !allowed {
			return false
		}
	}
	if reach.Blacklist != nil {
		if _, disallowed := reach.Blacklist[p.ID]; disallowed {
			return false
		}
	}

	for _, dep := range p.Imports {
		if reach.Check(dep) {
			if _, insource := reach.Source[p.ID]; insource {
				reach.Result[p.ID] = p
			}
			reach.Reaches[p.ID] = p
			return true
		}
	}

	reach.CannotReach[p.ID] = p
	return false
}
