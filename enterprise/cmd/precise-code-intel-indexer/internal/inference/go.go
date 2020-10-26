package inference

import (
	"path/filepath"
	"regexp"
)

const lsifGoImage = "sourcegraph/lsif-go:latest"

type lsifGoJobRecognizer struct{}

var _ IndexJobRecognizer = lsifGoJobRecognizer{}

func (lsifGoJobRecognizer) CanIndex(paths []string) bool {
	for _, path := range paths {
		if filepath.Base(path) == "go.mod" && !containsSegment(path, "vendor") {
			return true
		}

		// TODO(efritz) - support glide, dep, other historic package managers
		// TODO(efritz) - support projects without go.mod but a vendor dir and go files
	}

	return false
}

func (lsifGoJobRecognizer) InferIndexJobs(paths []string) (indexes []IndexJob) {
	for _, path := range paths {
		if filepath.Base(path) == "go.mod" && !containsSegment(path, "vendor") {
			root := dirWithoutDot(path)

			dockerSteps := []DockerStep{
				{
					Root:     root,
					Image:    lsifGoImage,
					Commands: []string{"go", "mod", "download"},
				},
			}

			indexes = append(indexes, IndexJob{
				DockerSteps: dockerSteps,
				Root:        root,
				Indexer:     lsifGoImage,
				IndexerArgs: []string{"lsif-go", "--no-animation"},
				Outfile:     "",
			})
		}
	}

	return indexes
}

func (lsifGoJobRecognizer) Patterns() []*regexp.Regexp {
	return []*regexp.Regexp{
		suffixPattern("go.mod"),
		segmentPattern("vendor"),
	}
}
