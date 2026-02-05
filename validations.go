package kparse

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

type validatorFactory func(fieldName string, rule string) (Validator, error)

type validatorFactoryMapKey struct {
	Op   string
	Kind reflect.Kind
}

var validatorFactoryMap = map[validatorFactoryMapKey]validatorFactory{
	// range validators are the only ones with no keyword prefix,
	// so they will match an empty string:
	{"", reflect.Int}:     newRangeValidator[int],
	{"", reflect.Int8}:    newRangeValidator[int8],
	{"", reflect.Int16}:   newRangeValidator[int16],
	{"", reflect.Int32}:   newRangeValidator[int32],
	{"", reflect.Int64}:   newRangeValidator[int64],
	{"", reflect.Uint}:    newRangeValidator[uint],
	{"", reflect.Uint8}:   newRangeValidator[uint8],
	{"", reflect.Uint16}:  newRangeValidator[uint16],
	{"", reflect.Uint32}:  newRangeValidator[uint32],
	{"", reflect.Uint64}:  newRangeValidator[uint64],
	{"", reflect.Float32}: newRangeValidator[float32],
	{"", reflect.Float64}: newRangeValidator[float64],

	{"len", reflect.String}: newLenValidator,
	{"len", reflect.Slice}:  newLenValidator,
	{"len", reflect.Map}:    newLenValidator,
	{"len", reflect.Array}:  newLenValidator,
	{"len", reflect.Chan}:   newLenValidator,
}

func newRangeValidator[T Number](fieldName string, rule string) (_ Validator, err error) {
	var i int
	for i < len(rule) && isInequalityChar(rule[i]) {
		i++
	}

	op := rule[:i]

	var isValid func(attr T, limit T) bool
	switch op {
	case "<":
		isValid = func(attr T, limit T) bool {
			return attr < limit
		}
	case "<=":
		isValid = func(attr T, limit T) bool {
			return attr <= limit
		}
	case ">":
		isValid = func(attr T, limit T) bool {
			return attr > limit
		}
	case ">=":
		isValid = func(attr T, limit T) bool {
			return attr >= limit
		}
	case "=":
		isValid = func(attr T, limit T) bool {
			return attr == limit
		}
	default:
		return nil, fmt.Errorf("unrecognized validator format: '%s'", op+"."+rule)
	}

	var limit T
	err = yaml.Unmarshal([]byte(rule[i:]), &limit)
	if err != nil {
		return nil, fmt.Errorf("error parsing number for range validator: '%s', usage: [< | > | <= | >= | =]<number>", rule)
	}

	return func(value any) error {
		v, _ := value.(*T)
		if v == nil {
			return fmt.Errorf("kparser code error: range validator called for invalid input: %T", value)
		}

		if !isValid(*v, limit) {
			return fmt.Errorf(
				"field %q with value %v should be %s %v",
				fieldName, *v, op, limit,
			)
		}

		return nil
	}, nil
}

func isInequalityChar(c byte) bool {
	return c == '<' || c == '>' || c == '='
}

func newLenValidator(fieldName string, rule string) (_ Validator, err error) {
	var i int
	for i < len(rule) && isInequalityChar(rule[i]) {
		i++
	}

	op := rule[:i]

	var isValid func(len int, limit int) bool
	switch op {
	case "<":
		isValid = func(length int, limit int) bool {
			return length < limit
		}
	case "<=":
		isValid = func(length int, limit int) bool {
			return length <= limit
		}
	case ">":
		isValid = func(length int, limit int) bool {
			return length > limit
		}
	case ">=":
		isValid = func(length int, limit int) bool {
			return length >= limit
		}
	case "=":
		isValid = func(length int, limit int) bool {
			return length == limit
		}
	default:
		return nil, fmt.Errorf("unrecognized validator format: '%s'", op+"."+rule)
	}

	var limit int
	err = yaml.Unmarshal([]byte(rule[i:]), &limit)
	if err != nil {
		return nil, fmt.Errorf("error parsing number for length validator: '%s', usage: [< | > | <= | >= | =]<number>", rule)
	}

	return func(value any) error {
		len := reflect.ValueOf(value).Elem().Len()

		if !isValid(len, limit) {
			return fmt.Errorf(
				"field %q with len %v should be %s %v",
				fieldName, len, op, limit,
			)
		}

		return nil
	}, nil
}
