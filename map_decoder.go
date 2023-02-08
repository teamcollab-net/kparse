package configparser

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/vingarcia/structscanner"
	"gopkg.in/yaml.v2"
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

	required := false
	if len(keys) > 1 {
		if keys[1] != "required" {
			return nil, fmt.Errorf(
				"unrecognized modifier: '%s' on struct field: '%s'",
				keys[1], info.Name,
			)
		}

		required = keys[1] != ""
	}

	if e.sourceMap[key] == nil {
		defaultYAML := info.Tags["default"]
		if defaultYAML != "" {
			value := reflect.New(info.Type)
			return value.Interface(), yaml.Unmarshal([]byte(defaultYAML), value.Interface())
		}

		if required {
			return nil, fmt.Errorf(
				"missing required config field '%s' of type %v",
				key, info.Type,
			)
		}

		// If it is a struct we keep parsing its fields
		// just to set the default values if they exist:
		if info.Kind == reflect.Struct {
			return newMapTagDecoder(e.tagName, map[string]interface{}{}), nil
		}

		// If it is not required we can safely ignore it:
		return nil, nil
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
