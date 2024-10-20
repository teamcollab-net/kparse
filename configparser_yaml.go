package configparser

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/vingarcia/structscanner"
	"gopkg.in/yaml.v3"
)

func MustParseYAMLFile(filepath string, targetStruct any) {
	err := ParseYAMLFile(filepath, targetStruct)
	if err != nil {
		panic(err)
	}
}

func ParseYAMLFile(path string, targetStruct any) error {
	if !filepath.IsAbs(path) {
		workingDir, err := os.Getwd()
		if err != nil {
			return err
		}
		path = filepath.Join(workingDir, path)
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer file.Close()
	return ParseYAMLFromReader(file, targetStruct)
}

func MustParseYAML(file []byte, targetStruct any) {
	err := ParseYAML(file, targetStruct)
	if err != nil {
		panic(err)
	}
}

func ParseYAML(file []byte, targetStruct any) error {
	return ParseYAMLFromReader(bytes.NewReader(file), targetStruct)
}

func MustParseYAMLFromReader(file io.Reader, targetStruct any) {
	err := ParseYAMLFromReader(file, targetStruct)
	if err != nil {
		panic(err)
	}
}

func ParseYAMLFromReader(file io.Reader, targetStruct any) error {
	var parsedYaml map[string]any
	err := yaml.NewDecoder(file).Decode(&parsedYaml)
	if err != nil {
		return err
	}

	return structscanner.Decode(
		targetStruct,
		newMapTagDecoder("yaml", parsedYaml),
	)
}
