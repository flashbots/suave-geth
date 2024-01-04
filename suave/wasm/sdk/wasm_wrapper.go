package sdk

import "unsafe"

// TODO: Enable this only for wasm

var defaultRouter = &router{}

func Register(fn interface{}) {
	if err := defaultRouter.Register(fn); err != nil {

	}
}

func Handle(valuePosition *uint32, length uint32) uint64 {
	input := readBufferFromMemory(valuePosition, length)

	output, err := defaultRouter.Run(input)
	if err != nil {
		panic(err)
	}

	posSizePairValue := copyBufferToMemory([]byte(output))

	// return the position and size
	return posSizePairValue
}

// readBufferFromMemory returns a buffer from WebAssembly
func readBufferFromMemory(bufferPosition *uint32, length uint32) []byte {
	subjectBuffer := make([]byte, length)
	pointer := uintptr(unsafe.Pointer(bufferPosition))
	for i := 0; i < int(length); i++ {
		s := *(*int32)(unsafe.Pointer(pointer + uintptr(i)))
		subjectBuffer[i] = byte(s)
	}
	return subjectBuffer
}

// copyBufferToMemory returns a single value
// (a kind of pair with position and length)
func copyBufferToMemory(buffer []byte) uint64 {
	bufferPtr := &buffer[0]
	unsafePtr := uintptr(unsafe.Pointer(bufferPtr))

	ptr := uint32(unsafePtr)
	size := uint32(len(buffer))

	return (uint64(ptr) << uint64(32)) | uint64(size)
}
