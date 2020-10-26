package index

import (
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/sourcegraph/internal/jsonc"
)

type jsonIndexConfiguration struct {
	SharedSteps []jsonDockerStep `json:"shared_steps"`
	IndexJobs   []jsonIndexJob   `json:"index_jobs"`
}

type jsonIndexJob struct {
	Steps       []jsonDockerStep `json:"steps"`
	Root        string           `json:"root"`
	Indexer     string           `json:"indexer"`
	IndexerArgs []string         `json:"indexer_args"`
	Outfile     string           `json:"outfile"`
}

type jsonDockerStep struct {
	Root     string   `json:"root"`
	Image    string   `json:"image"`
	Commands []string `json:"commands"`
}

func UnmarshalJSON(data []byte) (IndexConfiguration, error) {
	jsonData, err := jsonc.Parse(string(data))
	if err != nil {
		return IndexConfiguration{}, fmt.Errorf("invalid JSON: %v", err)
	}

	configuration := jsonIndexConfiguration{}
	if err := json.Unmarshal(jsonData, &configuration); err != nil {
		return IndexConfiguration{}, fmt.Errorf("invalid JSON: %v", err)
	}

	var indexJobs []IndexJob
	for _, value := range configuration.IndexJobs {
		indexJobs = append(indexJobs, IndexJob{
			Steps:       convertJSONDockerSteps(value.Steps),
			Root:        value.Root,
			Indexer:     value.Indexer,
			IndexerArgs: value.IndexerArgs,
			Outfile:     value.Outfile,
		})
	}

	return IndexConfiguration{
		SharedSteps: convertJSONDockerSteps(configuration.SharedSteps),
		IndexJobs:   indexJobs,
	}, nil
}

func convertJSONDockerSteps(raw []jsonDockerStep) []DockerStep {
	steps := []DockerStep{}
	for _, val := range raw {
		steps = append(steps, DockerStep{
			Root:     val.Root,
			Image:    val.Image,
			Commands: val.Commands,
		})
	}

	return steps
}
