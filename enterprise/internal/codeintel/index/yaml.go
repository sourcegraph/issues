package index

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

type yamlIndexConfiguration struct {
	SharedSteps []yamlDockerStep `yaml:"shared_steps"`
	IndexJobs   []yamlIndexJob   `yaml:"index_jobs"`
}

type yamlIndexJob struct {
	Steps       []yamlDockerStep `yaml:"steps"`
	Root        string           `yaml:"root"`
	Indexer     string           `yaml:"indexer"`
	IndexerArgs []string         `yaml:"indexer_args"`
	Outfile     string           `yaml:"outfile"`
}

type yamlDockerStep struct {
	Root     string   `yaml:"root"`
	Image    string   `yaml:"image"`
	Commands []string `yaml:"commands"`
}

func UnmarshalYAML(data []byte) (IndexConfiguration, error) {
	configuration := yamlIndexConfiguration{}
	if err := yaml.Unmarshal(data, &configuration); err != nil {
		return IndexConfiguration{}, fmt.Errorf("invalid YAML: %v", err)
	}

	var indexJobs []IndexJob
	for _, value := range configuration.IndexJobs {
		indexJobs = append(indexJobs, IndexJob{
			Steps:       convertYAMLDockerSteps(value.Steps),
			Root:        value.Root,
			Indexer:     value.Indexer,
			IndexerArgs: sliceize(value.IndexerArgs),
			Outfile:     value.Outfile,
		})
	}

	return IndexConfiguration{
		SharedSteps: convertYAMLDockerSteps(configuration.SharedSteps),
		IndexJobs:   indexJobs,
	}, nil
}

func convertYAMLDockerSteps(raw []yamlDockerStep) []DockerStep {
	steps := []DockerStep{}
	for _, val := range raw {
		steps = append(steps, DockerStep{
			Root:     val.Root,
			Image:    val.Image,
			Commands: sliceize(val.Commands),
		})
	}

	return steps
}

func sliceize(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
