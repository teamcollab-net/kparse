# ConfigParser

This project was created for filling a gap I found on the existing
Golang ecosystem: There was no simple library that did config parsing,
validation of required fields and allowed for default values at the same time.

The project is meant to be simple and currently supports only yaml config files.

There are 3 convenience functions that internally do the same thing:

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

	"github.com/blackpointcyber/configparser"
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

	configparser.MustParseYAMLFile("./examples/simple_usage/config.yaml", &config)

	fmt.Println(config)
}
```
