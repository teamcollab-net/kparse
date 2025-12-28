package kparse

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// LazyDecoder is used to allow the parser to work with
// multiple encoding types like JSON and YAML only Unmarshalling
// each value when necessary and preventing the parseFromMap function
// from being directly coupled to different encoding technologies.
//
// Tip: This struct works much like the json.RawMessage type.
type LazyDecoder func(target any) error

func (l LazyDecoder) Decode(target any) error {
	return l(target)
}

// UnmarshalJSON implements the json.Unmarshaler interface
// in a lazy way (much like json.RawMessage)
func (l *LazyDecoder) UnmarshalJSON(b []byte) error {
	*l = func(target any) error {
		return json.Unmarshal(b, target)
	}

	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
// in a lazy way (much like json.RawMessage)
func (l *LazyDecoder) UnmarshalYAML(value *yaml.Node) error {
	*l = func(target any) error {
		return value.Decode(target)
	}

	return nil
}
