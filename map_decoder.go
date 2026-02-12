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
	err := structi.ForEach(structPtr, func(field structi.Field) error {
		// Ignore multiples fields if there is a `,` as in `json:"foo,omitempty"`
		key := strings.SplitN(field.Tags[tagName], ",", 2)[0]
		if key == "" {
			return nil
		}

		required := false

		validations := []Validator{}
		if field.Tags["validate"] != "" {
			expressions := strings.Split(field.Tags["validate"], ",")
			for _, exp := range expressions {
				validatorName, rule := extractValidatorNameAndRule(exp)

				if validatorName == "required" {
					required = true
					continue
				}

				cacheKey := cacheKey{
					Kind:       field.Type.Kind(),
					FieldName:  field.Name,
					Expression: exp,
				}

				validator, err := withCache(cacheKey, func() (validator Validator, err error) {
					factory, found := validatorFactoryMap[validatorFactoryMapKey{validatorName, field.Type.Kind()}]
					if !found {
						return nil, fmt.Errorf(
							"unrecognized validation exp: '%s' on struct field: '%s'",
							exp, field.Name,
						)
					}

					return factory(field.Name, rule)
				})
				if err != nil {
					return err
				}

				validations = append(validations, validator)
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

		if field.Kind == reflect.Slice && field.Type.Elem().Kind() == reflect.Struct {
			var data []LazyDecoder
			err := sourceMap[key].Decode(&data)
			if err != nil {
				return fmt.Errorf(
					"can't map %T into nested slice %s of type %v",
					sourceMap[key], field.Name, field.Type,
				)
			}

			// For slice of structs, we need to convert each LazyDecoder to map[string]LazyDecoder
			sliceValue := reflect.MakeSlice(field.Type, len(data), len(data))

			for i, lazyDecoder := range data {
				var itemMap map[string]LazyDecoder
				err := lazyDecoder.Decode(&itemMap)
				if err != nil {
					return fmt.Errorf(
						"can't map element %d of slice %s into map[string]LazyDecoder: %v",
						i, field.Name, err,
					)
				}

				// Get a pointer to the slice element at position i
				elemPtr := sliceValue.Index(i).Addr().Interface()

				// Recursively parse the struct
				err = parseFromMap(tagName, elemPtr, itemMap)
				if err != nil {
					return fmt.Errorf(
						"error parsing element %d of slice %s: %v",
						i, field.Name, err,
					)
				}
			}

			// Set the slice value to the field
			return field.Set(sliceValue.Interface())
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

func extractValidatorNameAndRule(exp string) (validatorName string, rule string) {
	if exp == "" {
		return "", ""
	}

	// Parse the operator name:
	i := 0
	for i < len(exp) && isAlpha(exp[i]) {
		i++
	}

	validatorName = exp[:i]
	rule = exp[i:]
	return validatorName, rule
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

type cacheKey struct {
	// The validation needs to be compiled (with generics) for each kind of data
	Kind reflect.Kind

	// We need to consider the field name because error messages will output this name,
	// so we each field name requires a new validator function so the errors show up properly.
	FieldName string

	// If the expression is the same even on different structs we can reuse the same key
	Expression string
}

var validatorCache sync.Map

func withCache(cacheKey cacheKey, fn func() (Validator, error)) (Validator, error) {
	if v, _ := validatorCache.Load(cacheKey); v != nil {
		return v.(Validator), nil
	}

	validator, err := fn()
	if err != nil {
		return nil, err
	}

	validatorCache.Store(cacheKey, validator)

	return validator, nil
}
