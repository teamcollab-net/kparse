package kparse

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/vingarcia/structi"
	"gopkg.in/yaml.v3"
)

type Validator func(value any) error

// parseFromMap can be used to fill a struct with the values of a map.
//
// It works recursively so you can pass nested structs to it.
func parseFromMap(tagName string, structPtr any, sourceMap map[string]LazyDecoder) error {
	structType := reflect.TypeOf(structPtr)
	if structType.Kind() != reflect.Pointer {
		return fmt.Errorf("expected pointer to struct but got: %+v", structPtr)
	}
	structType = structType.Elem()

	return structi.ForEach(structPtr, func(field structi.Field) error {
		// Ignore multiples fields if there is a `,` as in `json:"foo,omitempty"`
		key := strings.SplitN(field.Tags[tagName], ",", 2)[0]
		if key == "" {
			return nil
		}

		required := false

		validations := []Validator{}
		if field.Tags["validate"] != "" {
			rules := strings.Split(field.Tags["validate"], ",")
			for _, rule := range rules {
				name, value, _ := strings.Cut(rule, "=")

				switch name {
				case "required":
					required = true

				case "min":
					validation, err := parseRangeValidationWithCache(structType, field.Name, field.Type, value+":")
					if err != nil {
						return fmt.Errorf("error parsing `min` validator: %w", err)
					}

					validations = append(validations, validation)

				case "max":
					validation, err := parseRangeValidationWithCache(structType, field.Name, field.Type, ":"+value)
					if err != nil {
						return fmt.Errorf("error parsing `max` validator: %w", err)
					}

					validations = append(validations, validation)

				case "range":
					validation, err := parseRangeValidationWithCache(structType, field.Name, field.Type, value)
					if err != nil {
						return fmt.Errorf("error parsing `range` validator: %w", err)
					}

					validations = append(validations, validation)

				default:
					return fmt.Errorf(
						"unrecognized validation rule: '%s' on struct field: '%s'",
						rule, field.Name,
					)
				}
			}
		}

		if sourceMap[key] == nil {
			defaultYAML := field.Tags["default"]
			if defaultYAML != "" {
				err := yaml.Unmarshal([]byte(defaultYAML), field.Value)
				if err != nil {
					return fmt.Errorf(`error parsing "default" value as YAML: %s`, err)
				}

				return nil
			}

			if required {
				return fmt.Errorf(
					"missing required field '%s' of type %v",
					key, field.Type,
				)
			}

			// If it is a struct we keep parsing its fields
			// just to set the default values if they exist:
			if field.Kind == reflect.Struct {
				return parseFromMap(tagName, field.Value, map[string]LazyDecoder{})
			}

			// If it is not required we can safely ignore it:
			return nil
		}

		if field.Kind == reflect.Struct {
			var data map[string]LazyDecoder
			err := sourceMap[key].Decode(&data)
			if err != nil {
				return fmt.Errorf(
					"can't map %T into nested struct %s of type %v",
					sourceMap[key], field.Name, field.Type,
				)
			}

			return parseFromMap(tagName, field.Value, data)
		}

		err := sourceMap[key].Decode(field.Value)
		if err != nil {
			return err
		}

		// Run the validations only after decoding the value:
		for _, validation := range validations {
			err := validation(field.Value)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

type cacheKey struct {
	t    reflect.Type
	name string
}

var rangeValidatorCache sync.Map

func parseRangeValidationWithCache(structType reflect.Type, fieldName string, fieldType reflect.Type, value string) (fn Validator, err error) {
	if v, _ := rangeValidatorCache.Load(cacheKey{t: structType, name: fieldName}); v != nil {
		return v.(Validator), nil
	}

	minStr, maxStr, _ := strings.Cut(value, ":")
	switch fieldType.Kind() {
	case reflect.Int:
		fn, err = newIntRangeValidator[int](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Int8:
		fn, err = newIntRangeValidator[int8](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Int16:
		fn, err = newIntRangeValidator[int16](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Int32:
		fn, err = newIntRangeValidator[int32](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Int64:
		fn, err = newIntRangeValidator[int64](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return nil, fmt.Errorf("support for unsigned int is not yet implemented")

	case reflect.Float32, reflect.Float64:
		return nil, fmt.Errorf("support for float is not yet implemented")

	case reflect.Complex64, reflect.Complex128:
		return nil, fmt.Errorf("support for complex is not yet implemented")

	default:
		return nil, fmt.Errorf("invalid field type for min, max and range validations: %v", fieldType)
	}

	rangeValidatorCache.Store(cacheKey{t: structType, name: fieldName}, fn)

	return fn, nil
}

func newIntRangeValidator[T Integer](fieldName string, minStr string, maxStr string) (_ Validator, err error) {
	var min, max int64
	if minStr != "" {
		min, err = strconv.ParseInt(minStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse min value for field %q: %q is not a valid integer", fieldName, minStr)
		}
	}

	if maxStr != "" {
		max, err = strconv.ParseInt(maxStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse max value for field %q: %q is not a valid integer", fieldName, maxStr)
		}
	}

	return func(value any) error {
		v, ok := value.(*T)
		if !ok {
			return fmt.Errorf("kparser code error: integer validator called for wrong type: %T", value)
		}
		if v == nil {
			return nil
		}

		intValue := int64(*v)

		if minStr != "" && intValue < min {
			return fmt.Errorf(
				"field %q with value %d is below the min value of %d",
				fieldName, intValue, min,
			)
		}

		if maxStr != "" && intValue > max {
			return fmt.Errorf(
				"field %q with value %d is above the max value of %d",
				fieldName, intValue, max,
			)
		}

		return nil
	}, nil
}
