# ConfigParser

This project was created for filling a gap I found on the existing
Golang ecosystem: There was no simple library that did the parsing of encoded data,
validation of required fields and allowed for default values at the same time.

The project currently supports JSON and YAML config files.

For each encoding type there are a few different options on how to
receive the data, for YAML for example we have:

- ParseYaml
- ParseYamlFile
- ParseYamlReader

and for each there is a version that panics instead of returning an error:

- MustParseYaml
- MustParseYamlFile
- MustParseYamlReader

## Usage Example


```golang
package main

import (
	"fmt"

	"github.com/vingarcia/kparser"
)

func main() {
	var config struct {
		SecretKey int    `yaml:"secretKey" validate:"required"`
		BaseURL   string `yaml:"baseUrl" default:"https://example.com"`

		Address struct {
			Street  string `yaml:"street" default:"defaultStreet"`
			City    string `yaml:"city"`
			Country string `yaml:"country"`
		} `yaml:"address"`
	}

	kparser.MustParseYAMLFile("./examples/simple_usage/config.yaml", &config)

	fmt.Println(config)
}
```
