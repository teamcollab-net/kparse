package kparse

import (
	"testing"

	tt "github.com/vingarcia/kparse/internal/testtools"
	"gopkg.in/yaml.v3"
)

func TestParseYAMLReader(t *testing.T) {
	tests := []struct {
		desc               string
		input              map[string]any
		targetStruct       any
		expectedStruct     any
		expectErrToContain []string
	}{
		{
			desc: "should work with simple yaml files",
			input: map[string]any{
				"foo": "bar",
			},
			targetStruct: &struct {
				Foo string `yaml:"foo"`
			}{},
			expectedStruct: &struct {
				Foo string `yaml:"foo"`
			}{
				Foo: "bar",
			},
		},
		{
			desc: "should work with nested yaml files",
			input: map[string]any{
				"foo": "bar",
				"bar": map[string]any{
					"subFoo": "bar",
				},
			},
			targetStruct: &struct {
				Foo string `yaml:"foo"`
				Bar struct {
					SubFoo string `yaml:"subFoo"`
				} `yaml:"bar"`
			}{},
			expectedStruct: &struct {
				Foo string `yaml:"foo"`
				Bar struct {
					SubFoo string `yaml:"subFoo"`
				} `yaml:"bar"`
			}{
				Foo: "bar",
				Bar: struct {
					SubFoo string `yaml:"subFoo"`
				}{
					SubFoo: "bar",
				},
			},
		},
		{
			desc: "should work with required fields",
			input: map[string]any{
				"foo": 42,
				"bar": "foo",
			},
			targetStruct: &struct {
				Foo int    `yaml:"foo" validate:"required"`
				Bar string `yaml:"bar"`
			}{},
			expectedStruct: &struct {
				Foo int    `yaml:"foo" validate:"required"`
				Bar string `yaml:"bar"`
			}{
				Foo: 42,
				Bar: "foo",
			},
		},
		{
			desc: "should report errors if a required field is missing",
			input: map[string]any{
				"bar": "foo",
			},
			targetStruct: &struct {
				Foo int    `yaml:"foo" validate:"required"`
				Bar string `yaml:"bar"`
			}{},
			expectErrToContain: []string{"missing", "required", "foo"},
		},
		{
			desc: "should work with default fields",
			input: map[string]any{
				"foo": 42,
				"bar": "foo",
			},
			targetStruct: &struct {
				Foo int    `yaml:"foo" default:"42"`
				Bar string `yaml:"bar"`
			}{},
			expectedStruct: &struct {
				Foo int    `yaml:"foo" default:"42"`
				Bar string `yaml:"bar"`
			}{
				Foo: 42,
				Bar: "foo",
			},
		},
		{
			desc: "should work with string slices",
			input: map[string]any{
				"bar": []string{"fakeItem1", "fakeItem2"},
			},
			targetStruct: &struct {
				Slice []string `yaml:"bar"`
			}{},
			expectedStruct: &struct {
				Slice []string `yaml:"bar"`
			}{
				Slice: []string{"fakeItem1", "fakeItem2"},
			},
		},
		{
			desc: "should work with map[string]any attributes",
			input: map[string]any{
				"map": map[string]string{
					"fakeKey1": "fakeItem1",
					"fakeKey2": "fakeItem2",
				},
			},
			targetStruct: &struct {
				Map map[string]any `yaml:"map"`
			}{},
			expectedStruct: &struct {
				Map map[string]any `yaml:"map"`
			}{
				Map: map[string]any{
					"fakeKey1": "fakeItem1",
					"fakeKey2": "fakeItem2",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			inputYaml, err := yaml.Marshal(test.input)
			tt.AssertNoErr(t, err)

			err = ParseYAML(inputYaml, test.targetStruct)
			if test.expectErrToContain != nil {
				tt.AssertErrContains(t, err, test.expectErrToContain...)
				t.Skip()
			}
			tt.AssertNoErr(t, err)

			tt.AssertEqual(t, test.targetStruct, test.expectedStruct)
		})
	}
}
