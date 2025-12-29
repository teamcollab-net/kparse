package kparse

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/vingarcia/structi"
	"gopkg.in/yaml.v3"
)

type Validator func(value any) error

// parseFromMap can be used to fill a struct with the values of a map.
//
// It works recursively so you can pass nested structs to it.
func parseFromMap(tagName string, structPtr any, sourceMap map[string]LazyDecoder) (errs error) {
	structType := reflect.TypeOf(structPtr)

	err := structi.ForEach(structPtr, func(field structi.Field) error {
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
				op, value := extractOpAndValue(rule)

				switch op {
				case "required":
					required = true

				case ">-": // fix incorrectly identified op:
					// op = ">"
					value = "-" + value

					fallthrough
				case ">":
					validation, err := parseRangeValidationWithCache(structType, field.Name, field.Type, value, "")
					if err != nil {
						return fmt.Errorf("error parsing greater than (`>`) validator: %w", err)
					}

					validations = append(validations, validation)

				case "<-": // fix incorrectly identified op:
					// op = "<"
					value = "-" + value

					fallthrough
				case "<": // less than
					validation, err := parseRangeValidationWithCache(structType, field.Name, field.Type, "", value)
					if err != nil {
						return fmt.Errorf("error parsing less than (`<`) validator: %w", err)
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
			errs = errors.Join(errs, validation(field.Value))
		}

		return nil
	})

	return errors.Join(err, errs)
}

func extractOpAndValue(rule string) (op string, value string) {
	if rule == "" {
		return "", ""
	}

	i := 0
	// Check if rule starts with a letter (named operation)
	if isAlpha(rule[0]) {
		// Extract word containing only letters
		for i < len(rule) && isAlpha(rule[i]) {
			i++
		}
	} else {
		// Otherwise, extract non-alphanumeric characters as the operator
		for i < len(rule) && !isAlphaNumeric(rule[i]) {
			i++
		}
	}

	op = rule[:i]
	value = rule[i:]
	return op, value
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isAlphaNumeric(c byte) bool {
	return isAlpha(c) || c >= '0' && c <= '9'
}

type cacheKey struct {
	t    reflect.Type
	name string
}

var validatorCache sync.Map

func parseRangeValidationWithCache(
	structType reflect.Type,
	fieldName string,
	fieldType reflect.Type,
	minStr string,
	maxStr string,
) (fn Validator, err error) {
	cacheKey := cacheKey{
		t:    structType,
		name: fieldName,
	}
	if v, _ := validatorCache.Load(cacheKey); v != nil {
		return v.(Validator), nil
	}

	switch fieldType.Kind() {
	case reflect.Int:
		fn, err = newRangeValidator[int](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Int8:
		fn, err = newRangeValidator[int8](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Int16:
		fn, err = newRangeValidator[int16](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Int32:
		fn, err = newRangeValidator[int32](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Int64:
		fn, err = newRangeValidator[int64](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Uint:
		fn, err = newRangeValidator[uint](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Uint8:
		fn, err = newRangeValidator[uint8](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Uint16:
		fn, err = newRangeValidator[uint16](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Uint32:
		fn, err = newRangeValidator[uint32](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Uint64:
		fn, err = newRangeValidator[uint64](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Float32:
		fn, err = newRangeValidator[float32](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	case reflect.Float64:
		fn, err = newRangeValidator[float64](fieldName, minStr, maxStr)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("invalid field type for min/max validations: %v", fieldType)
	}

	validatorCache.Store(cacheKey, fn)

	return fn, nil
}

func newRangeValidator[T Number](fieldName string, minStr string, maxStr string) (_ Validator, err error) {
	var min, max T
	if minStr != "" {
		err := yaml.Unmarshal([]byte(minStr), &min)
		if err != nil {
			return nil, fmt.Errorf("unable to parse min value for field %q: %q is not a valid integer", fieldName, minStr)
		}
	}

	if maxStr != "" {
		err = yaml.Unmarshal([]byte(maxStr), &max)
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

		if minStr != "" && *v < min {
			return fmt.Errorf(
				"field %q with value %v is below the min value of %v",
				fieldName, *v, min,
			)
		}

		if maxStr != "" && *v > max {
			return fmt.Errorf(
				"field %q with value %v is above the max value of %v",
				fieldName, *v, max,
			)
		}

		return nil
	}, nil
}
