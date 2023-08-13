package abi

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
)

func randomInt(min, max int) int {
	// Make sure max is greater than min
	if max <= min {
		panic(fmt.Errorf("max must be greater than min"))
	}

	// Calculate the range
	diff := big.NewInt(int64(max - min))

	// Generate a random number within the specified range
	randomValue, err := rand.Int(rand.Reader, diff)
	if err != nil {
		panic(err)
	}

	// Add the minimum value to the generated random number
	randomNumber := int(randomValue.Int64()) + min

	return randomNumber
}

func generateNumber(t Type) interface{} {
	b := make([]byte, t.Size/8)
	if t.T == UintTy {
		rand.Read(b)
	} else {
		rand.Read(b[1:])
	}

	num := big.NewInt(1).SetBytes(b)
	if t.Size == 8 || t.Size == 16 || t.Size == 32 || t.Size == 64 {
		return reflect.ValueOf(num.Int64()).Convert(t.GetType()).Interface()
	}
	return num
}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randString(n int, dict string) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = dict[randomInt(0, len(dict))]
	}
	return string(b)
}

func GenerateRandomTypeForMethod(m Method) []interface{} {
	var types []interface{}
	for _, input := range m.Inputs {
		types = append(types, GenerateRandomType(input.Type))
	}
	return types
}

func GenerateRandomType(t Type) interface{} {
	switch t.T {
	case IntTy:
		fallthrough
	case UintTy:
		return generateNumber(t)

	case BoolTy:
		if randomInt(0, 1) == 1 {
			return true
		}
		return false

	case AddressTy:
		buf := common.Address{}
		rand.Read(buf[:])
		return buf

	case StringTy:
		return randString(randomInt(1, 100), letters)

	case BytesTy:
		buf := make([]byte, randomInt(1, 100))
		rand.Read(buf)
		return buf

	case FixedBytesTy, FunctionTy:
		buf := make([]byte, t.Size)
		rand.Read(buf)

		val := reflect.New(t.GetType()).Elem()
		for i := 0; i < len(buf); i++ {
			val.Index(i).Set(reflect.ValueOf(buf[i]))
		}
		return val.Interface()

	case SliceTy:
		size := randomInt(1, 10)
		sliceType := reflect.SliceOf(reflect.TypeOf(GenerateRandomType(*t.Elem)))
		val := reflect.MakeSlice(sliceType, size, size)
		for i := 0; i < size; i++ {
			val.Index(i).Set(reflect.ValueOf(GenerateRandomType(*t.Elem)))
		}
		return val.Interface()

	case ArrayTy:
		size := t.Size
		arrayType := reflect.ArrayOf(size, reflect.TypeOf(GenerateRandomType(*t.Elem)))
		val := reflect.New(arrayType).Elem()
		for i := 0; i < size; i++ {
			val.Index(i).Set(reflect.ValueOf(GenerateRandomType(*t.Elem)))
		}
		return val.Interface()

	case TupleTy:
		structFields := make([]reflect.StructField, len(t.TupleElems))
		for i, field := range t.TupleElems {
			structFields[i] = reflect.StructField{
				Name: ToCamelCase(t.TupleRawNames[i]),
				Type: reflect.TypeOf(GenerateRandomType(*field)),
			}
		}
		structType := reflect.StructOf(structFields)
		val := reflect.New(structType).Elem()
		for i, field := range t.TupleElems {
			val.Field(i).Set(reflect.ValueOf(GenerateRandomType(*field)))
		}
		return val.Interface()

	default:
		panic(fmt.Errorf("type not implemented: %v", t.T))
	}
}
