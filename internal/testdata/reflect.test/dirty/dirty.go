package dirty

import "reflect"

// CallByName uses reflect.Value.MethodByName, which disables
// dead code elimination for all interface methods.
func CallByName(v any, name string) []reflect.Value {
	return reflect.ValueOf(v).MethodByName(name).Call(nil)
}
