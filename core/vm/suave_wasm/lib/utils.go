package lib

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"unsafe"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func UnpackInputs(argSpec abi.Arguments) ([]interface{}, error) {
	inputBuf := bytes.NewBuffer([]byte{})
	io.Copy(inputBuf, os.Stdin)

	unpacked, err := argSpec.Unpack(inputBuf.Bytes())
	if err != nil {
		return nil, err
	}

	return unpacked, nil
}

func ReturnPackedArgs(argSpec abi.Arguments, args ...interface{}) {
	packed, err := argSpec.Pack(args...)
	if err != nil {
		Fail(fmt.Errorf("could not pack output: %w", err))
	}

	ReturnBytes(packed)
}

func Fail(err error) {
	io.Copy(os.Stdout, bytes.NewBufferString(err.Error()))
	os.Exit(1)
}

func ReturnBytes(data []byte) {
	io.Copy(os.Stdout, bytes.NewReader(data))
}

//go:inline
func bytesToPointer(b []byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(unsafe.SliceData(b))))
}

//go:inline
func stringToPointer(s string) uint32 {
	return uint32(uintptr(unsafe.Pointer(unsafe.StringData(s))))
}
