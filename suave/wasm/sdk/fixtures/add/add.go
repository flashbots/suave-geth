package main

import (
	"reflect"
)

func _add(x, y uint32) uint32 {
	return x + y
}

func do(fn interface{}) {
	methodTyp := reflect.TypeOf(fn)
	if kind := methodTyp.Kind(); kind != reflect.Func {
		panic("x")
	}

	reflect.ValueOf(fn)
	//methodTyp.NumIn()
}

//export export
func export(valuePosition *uint32, length uint32) uint64 {
	//sdk.Register(_add)
	do(_add)

	return 0
	//return sdk.Handle(valuePosition, length)
}

// main is required for the `wasi` target, even if it isn't used.
func main() {}
