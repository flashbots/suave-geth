package abi

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mitchellh/mapstructure"
)

func (e Event) ParseLogToObject(output interface{}, log *types.Log) error {
	vals, err := e.ParseLog(log)
	if err != nil {
		return err
	}

	config := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   output,
		TagName:  "abi",
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	if err := decoder.Decode(vals); err != nil {
		return err
	}
	return nil
}

func (e Event) ParseLog(log *types.Log) (map[string]interface{}, error) {
	// validate that the log matches the event
	eventTopic := log.Topics[0]
	if eventTopic != e.ID {
		return nil, fmt.Errorf("topic %x does not match event %s", eventTopic, e.Name)
	}

	vals := map[string]interface{}{}

	// unpack the non-indexed arguments
	if err := e.Inputs.UnpackIntoMap(vals, log.Data); err != nil {
		return nil, err
	}

	// unpack the indexed values
	indxFields := []Argument{}
	topics := log.Topics[1:]

	for _, arg := range e.Inputs {
		if arg.Indexed {
			indxFields = append(indxFields, arg)
		}
	}
	if len(indxFields) != len(topics) {
		return nil, fmt.Errorf("event %s has %d topics but %d indexed fields", e.Name, len(topics), len(indxFields))
	}
	for indx, arg := range indxFields {
		var val interface{}

		switch arg.Type.T {
		case TupleTy:
			return nil, fmt.Errorf("tuple type in topic reconstruction")
		case StringTy, BytesTy, SliceTy, ArrayTy:
			val = topics[indx]
		default:
			var err error
			if val, err = toGoType(0, arg.Type, topics[indx].Bytes()); err != nil {
				return nil, err
			}
		}

		vals[arg.Name] = val
	}

	return vals, nil
}
