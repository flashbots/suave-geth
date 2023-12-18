package abi

import (
	"fmt"
	"reflect"
	"strings"
)

func NewTypeFromString(s string) (Type, error) {
	if strings.HasPrefix(s, "tuple") {
		sig, args, err := newTypeForTuple(s)
		if err != nil {
			return Type{}, err
		}
		return NewType(sig, "", args)
	}
	return NewType(s, "", nil)
}

func (t Type) Pack(v interface{}) ([]byte, error) {
	return t.pack(reflect.ValueOf(v))
}

func (t Type) Unpack(data []byte, obj interface{}) error {
	fmt.Println(toGoType(0, t, data))

	return nil
}

// newTypeForTuple implements the format described in https://blog.ricmoo.com/human-readable-contract-abis-in-ethers-js-141902f4d917
func newTypeForTuple(s string) (string, []ArgumentMarshaling, error) {
	if !strings.HasPrefix(s, "tuple") {
		return "", nil, fmt.Errorf("'tuple' prefix not found")
	}

	pos := strings.Index(s, "(")
	if pos == -1 {
		return "", nil, fmt.Errorf("not a tuple, '(' not found")
	}

	// this is the type of the tuple. It can either be
	// tuple, tuple[] or tuple[x]
	sig := s[:pos]
	s = s[pos:]

	// Now, decode the arguments of the tuple
	// tuple(arg1, arg2, tuple(arg3, arg4)).
	// We need to find the commas that are not inside a nested tuple.
	// We do this by keeping a counter of the number of open parens.

	var (
		parenthesisCount int
		fields           []string
	)

	lastComma := 1
	for indx, c := range s {
		switch c {
		case '(':
			parenthesisCount++
		case ')':
			parenthesisCount--
			if parenthesisCount == 0 {
				fields = append(fields, s[lastComma:indx])

				// this should be the end of the tuple
				if indx != len(s)-1 {
					return "", nil, fmt.Errorf("invalid tuple, it does not end with ')'")
				}
			}
		case ',':
			if parenthesisCount == 1 {
				fields = append(fields, s[lastComma:indx])
				lastComma = indx + 1
			}
		}
	}

	// trim the args of spaces
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}

	// decode the type of each field
	var args []ArgumentMarshaling
	for _, field := range fields {
		// anonymous fields are not supported so the first
		// string should be the identifier of the field.

		spacePos := strings.Index(field, " ")
		if spacePos == -1 {
			return "", nil, fmt.Errorf("invalid tuple field name not found '%s'", field)
		}

		name := field[:spacePos]
		field = field[spacePos+1:]

		if strings.HasPrefix(field, "tuple") {
			// decode a recursive tuple
			sig, elems, err := newTypeForTuple(field)
			if err != nil {
				return "", nil, err
			}
			args = append(args, ArgumentMarshaling{
				Name:       name,
				Type:       sig,
				Components: elems,
			})
		} else {
			// basic type. Try to decode it to see
			// if it is a correct abi type.
			if _, err := NewType(field, "", nil); err != nil {
				return "", nil, fmt.Errorf("invalid tuple basic field '%s': %v", field, err)
			}
			args = append(args, ArgumentMarshaling{
				Name: name,
				Type: field,
			})
		}
	}

	return sig, args, nil
}
