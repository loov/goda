package main

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

}
