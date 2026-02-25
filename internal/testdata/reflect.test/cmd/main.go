package main

import (
	_ "reflect.test/clean"
	"reflect.test/dirty"
)

func main() {
	dirty.CallByName(struct{ Hello func() }{}, "Hello")
}
