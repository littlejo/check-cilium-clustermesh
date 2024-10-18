package lib

import (
	"strings"

	"gopkg.in/yaml.v2"
)

type Resource struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
}

func ParseYAMLResources(yamlInput string) (map[string]string, error) {
	var resources []Resource
	decoder := yaml.NewDecoder(strings.NewReader(yamlInput))

	for {
		var resource Resource
		if err := decoder.Decode(&resource); err != nil {
			break
		}
		resources = append(resources, resource)
	}

	result := make(map[string]string)
	for _, res := range resources {
		result[res.Kind] = res.Metadata.Name
	}

	return result, nil
}
