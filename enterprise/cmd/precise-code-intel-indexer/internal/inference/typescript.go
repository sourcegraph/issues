package inference

import (
	"path/filepath"
	"regexp"
)

const lsifTscImage = "sourcegraph/lsif-node:latest"
const nodeInstallImage = "node:alpine3.12"

type lsifTscJobRecognizer struct{}

var _ IndexJobRecognizer = lsifTscJobRecognizer{}

func (lsifTscJobRecognizer) CanIndex(paths []string) bool {
	for _, path := range paths {
		if filepath.Base(path) == "tsconfig.json" && !containsSegment(path, "node_modules") {
			return true
		}

		// TODO(efritz) - check for javascript files
	}

	return false
}

func (lsifTscJobRecognizer) InferIndexJobs(paths []string) (indexes []IndexJob) {
	for _, path := range paths {
		if filepath.Base(path) == "tsconfig.json" && !containsSegment(path, "node_modules") {
			var dockerSteps []DockerStep
			for _, dir := range ancestorDirs(path) {
				if !contains(paths, filepath.Join(dir, "package.json")) {
					continue
				}

				var commands []string
				if contains(paths, filepath.Join(dir, "yarn.lock")) {
					commands = append(commands, "yarn", "--ignore-engines")
				} else {
					commands = append(commands, "npm", "install")
				}

				dockerSteps = append(dockerSteps, DockerStep{
					Root:     dir,
					Image:    nodeInstallImage,
					Commands: commands,
				})
			}

			n := len(dockerSteps)
			for i := 0; i < n/2; i++ {
				dockerSteps[i], dockerSteps[n-i-1] = dockerSteps[n-i-1], dockerSteps[i]
			}

			indexes = append(indexes, IndexJob{
				DockerSteps: dockerSteps,
				Root:        dirWithoutDot(path),
				Indexer:     lsifTscImage,
				IndexerArgs: []string{"lsif-tsc", "-p", "."},
				Outfile:     "",
			})
		}
	}

	return indexes
}

func (lsifTscJobRecognizer) Patterns() []*regexp.Regexp {
	return []*regexp.Regexp{
		suffixPattern("tsconfig.json"),
		suffixPattern("package.json"),
	}
}
