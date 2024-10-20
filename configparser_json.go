package configparser

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/vingarcia/structscanner"
)

func MustParseJSONFile(filepath string, targetStruct any) {
	err := ParseJSONFile(filepath, targetStruct)
	if err != nil {
		panic(err)
	}
}

func ParseJSONFile(path string, targetStruct any) error {
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
	return ParseJSONFromReader(file, targetStruct)
}

func MustParseJSON(file []byte, targetStruct any) {
	err := ParseJSON(file, targetStruct)
	if err != nil {
		panic(err)
	}
}

func ParseJSON(file []byte, targetStruct any) error {
	return ParseJSONFromReader(bytes.NewReader(file), targetStruct)
}

func MustParseJSONFromReader(file io.Reader, targetStruct any) {
	err := ParseJSONFromReader(file, targetStruct)
	if err != nil {
		panic(err)
	}
}

func ParseJSONFromReader(file io.Reader, targetStruct any) error {
	var parsedJson map[string]any
	err := json.NewDecoder(file).Decode(&parsedJson)
	if err != nil {
		return err
	}

	return structscanner.Decode(
		targetStruct,
		newMapTagDecoder("json", parsedJson),
	)
}
