package configparser

import (
	"bytes"
	"io"
	"os"

	"github.com/vingarcia/structscanner"
	"gopkg.in/yaml.v2"
)

func MustParseYAMLFile(filename string, targetStruct any) {
	err := ParseYAMLFile(filename, targetStruct)
	if err != nil {
		panic(err)
	}
}

func ParseYAMLFile(filename string, targetStruct any) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	defer file.Close()
	return ParseYAMLReader(file, targetStruct)
}

func MustParseYAML(file []byte, targetStruct any) {
	err := ParseYAML(file, targetStruct)
	if err != nil {
		panic(err)
	}
}

func ParseYAML(file []byte, targetStruct any) error {
	return ParseYAMLReader(bytes.NewReader(file), targetStruct)
}

func MustParseYAMLReader(file io.Reader, targetStruct any) {
	err := ParseYAMLReader(file, targetStruct)
	if err != nil {
		panic(err)
	}
}

func ParseYAMLReader(file io.Reader, targetStruct any) error {
	var parsedYaml map[any]any
	err := yaml.NewDecoder(file).Decode(&parsedYaml)
	if err != nil {
		return err
	}

	return structscanner.Decode(
		targetStruct,
		newMapTagDecoder("yaml", parsedYaml),
	)
}
