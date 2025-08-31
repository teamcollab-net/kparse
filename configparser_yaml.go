package kparser

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func MustParseYAMLFile(filepath string, targetStruct any) {
	err := ParseYAMLFile(filepath, targetStruct)
	if err != nil {
		panic(err)
	}
}

func ParseYAMLFile(path string, targetStruct any) (err error) {
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
	defer func() {
		err = errors.Join(err, file.Close())
	}()

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
	var data map[string]any
	err := yaml.NewDecoder(file).Decode(&data)
	if err != nil {
		return err
	}

	return parseFromMap("yaml", targetStruct, data)
}
