package main

import (
	_ "embed"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"gopkg.in/yaml.v2"
)

//go:embed config.yaml
var configRaw []byte

type Config struct {
	Groups []*QueryGroupConfig
}

type QueryGroupConfig struct {
	Name    string
	Queries []*QueryConfig
}

type QueryConfig struct {
	Query string
	Name  string

	// An unset interval defaults to 1m
	Interval time.Duration

	// An empty value for Protocols means "all"
	Protocols []Protocol
}

var allProtocols = []Protocol{Batch, Stream}

// Protocol represents either the graphQL Protocol or the streaming Protocol
type Protocol uint8

const (
	Batch Protocol = iota
	Stream
)

func (s *Protocol) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v string
	if err := unmarshal(&v); err != nil {
		return err
	}

	switch v {
	case "stream", "streaming":
		*s = Stream
	case "batch":
		*s = Batch
	default:
		return errors.Errorf("invalid search type %s", v)
	}

	return nil
}

func loadQueries() (*Config, error) {
	var config Config
	err := yaml.UnmarshalStrict(configRaw, &config)
	if err != nil {
		return nil, err
	}
	massageYaml(&config)
	return &config, nil
}

func massageYaml(c *Config) {
	for _, group := range c.Groups {
		for _, q := range group.Queries {
			// Remove newlines and extra space from "readable" yaml
			parts := strings.Split(q.Query, "\n")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			q.Query = strings.TrimSpace(strings.Join(parts, " "))
		}
	}
}
