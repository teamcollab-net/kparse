package configparser

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/vingarcia/structscanner"
)

// MapTagDecoder can be used to fill a struct with the values of a map.
//
// It works recursively so you can pass nested structs to it.
type mapTagDecoder struct {
	tagName   string
	sourceMap map[string]any
}

// newMapTagDecoder returns a new decoder for filling a given struct
// with the values from the sourceMap argument.
//
// The values from the sourceMap will be mapped to the struct using the key
// present in the tagName of each field of the struct.
func newMapTagDecoder(tagName string, sourceMap map[string]interface{}) mapTagDecoder {
	return mapTagDecoder{
		tagName:   tagName,
		sourceMap: sourceMap,
	}
}

// DecodeField implements the TagDecoder interface
func (e mapTagDecoder) DecodeField(info structscanner.Field) (interface{}, error) {
	keys := strings.SplitN(info.Tags[e.tagName], ",", 2)
	key := keys[0]
	required := len(keys) > 1 && keys[1] == "required"

	if required && e.sourceMap[key] == nil {
		return nil, fmt.Errorf(
			"missing required field '%s' of type %v",
			key, info.Type,
		)
	}

	if info.Kind == reflect.Struct {
		nestedMap, ok := e.sourceMap[key].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf(
				"can't map %T into nested struct %s of type %v",
				e.sourceMap[key], info.Name, info.Type,
			)
		}

		// By returning a decoder you tell the library to run
		// it recursively on this nestedMap:
		return newMapTagDecoder(e.tagName, nestedMap), nil
	}

	return e.sourceMap[key], nil
}
