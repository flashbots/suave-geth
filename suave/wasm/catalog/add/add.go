package main

//export add
func add(x, y uint32, other string) uint32 {
	return x + y
}

// main is required for the `wasi` target, even if it isn't used.
func main() {}
