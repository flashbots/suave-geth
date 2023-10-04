package main

import (
	"bytes"
	"io"
	"os"
	"unsafe"
)

var (
	bid  = []byte{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef}
	key  = "someKey"
	data = make([]byte, 1024)
)

func main() {
	keyOffset := stringToPointer(key)
	keySize := uint32(len(key))

	bidOffset := bytesToPointer(bid)
	bidSize := uint32(len(bid))

	bufOffset := bytesToPointer(data)
	bufSize := uint32(len(data))

	n := uint32(0)
	nOffset := uint32(uintptr(unsafe.Pointer(&n)))

	errno := retrieve(keyOffset, keySize, bidOffset, bidSize, bufOffset, bufSize, nOffset)
	if errno != 0 {
		os.Exit(1)
	}

	io.Copy(os.Stdout, bytes.NewReader(data[:n]))
}

//go:wasmimport suavexec retrieve
//go:noescape
func retrieve(keyOffset, keySize, bidOffset, bidSize, bufOffset, bufSize, n uint32) uint32

//go:inline
func bytesToPointer(b []byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(unsafe.SliceData(b))))
}

//go:inline
func stringToPointer(s string) uint32 {
	return uint32(uintptr(unsafe.Pointer(unsafe.StringData(s))))
}
