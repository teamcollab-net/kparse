package kparse

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/vingarcia/structi"
	"gopkg.in/yaml.v3"
)

// parseFromMap can be used to fill a struct with the values of a map.
//
// It works recursively so you can pass nested structs to it.
func parseFromMap(tagName string, structPtr any, sourceMap map[string]any) error {
	return structi.ForEach(structPtr, func(field structi.Field) error {
		// Ignore multiples fields if there is a `,` as in `json:"foo,omitempty"`
		key := strings.SplitN(field.Tags[tagName], ",", 2)[0]
		if key == "" {
			return nil
		}

		required := false
		if field.Tags["validate"] != "" {
			validations := strings.Split(field.Tags["validate"], ",")
			if validations[0] != "required" {
				return fmt.Errorf(
					"unrecognized validation: '%s' on struct field: '%s'",
					validations[0], field.Name,
				)
			}

			required = true
		}

		if sourceMap[key] == nil {
			defaultYAML := field.Tags["default"]
			if defaultYAML != "" {
				value := reflect.New(field.Type)
				err := yaml.Unmarshal([]byte(defaultYAML), value.Interface())
				if err != nil {
					return fmt.Errorf(`error parsing "default" value as YAML: %s`, err)
				}

				return field.Set(value)
			}

			if required {
				return fmt.Errorf(
					"missing required config field '%s' of type %v",
					key, field.Type,
				)
			}

			// If it is a struct we keep parsing its fields
			// just to set the default values if they exist:
			if field.Kind == reflect.Struct {
				return parseFromMap(tagName, field.Value, map[string]any{})
			}

			// If it is not required we can safely ignore it:
			return nil
		}

		if field.Kind == reflect.Struct {
			nestedMap, ok := sourceMap[key].(map[string]any)
			if !ok {
				return fmt.Errorf(
					"can't map %T into nested struct %s of type %v",
					sourceMap[key], field.Name, field.Type,
				)
			}

			// By returning a decoder you tell the library to run
			// it recursively on this nestedMap:
			return parseFromMap(tagName, field.Value, nestedMap)
		}

		return field.Set(sourceMap[key])
	})
}
