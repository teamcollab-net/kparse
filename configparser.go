package configparser

import (
	"bytes"
	"io"
	"os"

	"github.com/vingarcia/structscanner"
	"gopkg.in/yaml.v2"
)

func MustParseYAMLFile(targetStruct interface{}, filename string) {
	err := ParseYAMLFile(targetStruct, filename)
	if err != nil {
		panic(err)
	}
}

func ParseYAMLFile(targetStruct interface{}, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	defer file.Close()
	return ParseYAMLReader(targetStruct, file)
}

func MustParseYAML(targetStruct interface{}, file []byte) {
	err := ParseYAML(targetStruct, file)
	if err != nil {
		panic(err)
	}
}

func ParseYAML(targetStruct interface{}, file []byte) error {
	return ParseYAMLReader(targetStruct, bytes.NewReader(file))
}

func MustParseYAMLReader(targetStruct interface{}, file io.Reader) {
	err := ParseYAMLReader(targetStruct, file)
	if err != nil {
		panic(err)
	}
}

func ParseYAMLReader(targetStruct interface{}, file io.Reader) error {
	var parsedYaml map[string]interface{}
	err := yaml.NewDecoder(file).Decode(&parsedYaml)
	if err != nil {
		return err
	}

	return structscanner.Decode(
		targetStruct,
		newMapTagDecoder("yaml", parsedYaml),
	)
}
