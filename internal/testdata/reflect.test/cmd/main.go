package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	_ "reflect.test/clean"
	"reflect.test/dirty"
)

func main() {
	flag.Parse()
	fmt.Println(context.Background())
	json.NewEncoder(nil).Encode(nil)
	dirty.CallByName(struct{ Hello func() }{}, "Hello")
}
