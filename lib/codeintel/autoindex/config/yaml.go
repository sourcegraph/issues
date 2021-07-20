package config

import (
	"github.com/cockroachdb/errors"
	"gopkg.in/yaml.v2"
)

func UnmarshalYAML(data []byte) (IndexConfiguration, error) {
	configuration := IndexConfiguration{}
	if err := yaml.Unmarshal(data, &configuration); err != nil {
		return IndexConfiguration{}, errors.Errorf("invalid YAML: %v", err)
	}

	return configuration, nil
}
