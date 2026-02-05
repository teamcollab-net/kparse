package main

import (
	"fmt"

	"github.com/teamcollab-net/kparse"
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

		MaxRetries         int      `yaml:"maxRetries" validate:">0,<=10"` // greater than zero less or equal to 10
		ServiceDescription string   `yaml:"desc" validate:"len>=30"`       // At least 30 characters
		AllowedDomains     []string `yaml:"domains" validate:"len>=1"`     // At least one allowed domain
	}

	kparse.MustParseYAMLFile("./examples/simple_usage/config.yaml", &config)

	fmt.Println(config)
}
