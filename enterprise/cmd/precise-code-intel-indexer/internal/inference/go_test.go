package inference

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLSIFGoJobRecognizerCanIndex(t *testing.T) {
	recognizer := lsifGoJobRecognizer{}
	testCases := []struct {
		paths    []string
		expected bool
	}{
		{paths: []string{"go.mod"}, expected: true},
		{paths: []string{"a/go.mod"}, expected: true},
		{paths: []string{"package.json"}, expected: false},
		{paths: []string{"vendor/foo/bar/package.json"}, expected: false},
		{paths: []string{"foo/bar-go.mod"}, expected: false},
	}

	for _, testCase := range testCases {
		name := strings.Join(testCase.paths, ", ")

		t.Run(name, func(t *testing.T) {
			if value := recognizer.CanIndex(testCase.paths); value != testCase.expected {
				t.Errorf("unexpected result from CanIndex. want=%v have=%v", testCase.expected, value)
			}
		})
	}
}

func TestLsifGoJobRecognizerInferIndexJobsGoModRoot(t *testing.T) {
	recognizer := lsifGoJobRecognizer{}
	paths := []string{
		"go.mod",
	}

	expectedIndexJobs := []IndexJob{
		{
			DockerSteps: []DockerStep{
				{
					Root:     "",
					Image:    lsifGoImage,
					Commands: []string{"go", "mod", "download"},
				},
			},
			Root:        "",
			Indexer:     lsifGoImage,
			IndexerArgs: []string{"lsif-go", "--no-animation"},
			Outfile:     "",
		},
	}
	if diff := cmp.Diff(expectedIndexJobs, recognizer.InferIndexJobs(paths)); diff != "" {
		t.Errorf("unexpected index jobs (-want +got):\n%s", diff)
	}
}

func TestLsifGoJobRecognizerInferIndexJobsGoModSubdirs(t *testing.T) {
	recognizer := lsifGoJobRecognizer{}
	paths := []string{
		"a/go.mod",
		"b/go.mod",
		"c/go.mod",
	}

	expectedIndexJobs := []IndexJob{
		{
			DockerSteps: []DockerStep{
				{
					Root:     "a",
					Image:    lsifGoImage,
					Commands: []string{"go", "mod", "download"},
				},
			},
			Root:        "a",
			Indexer:     lsifGoImage,
			IndexerArgs: []string{"lsif-go", "--no-animation"},
			Outfile:     "",
		},
		{
			DockerSteps: []DockerStep{
				{
					Root:     "b",
					Image:    lsifGoImage,
					Commands: []string{"go", "mod", "download"},
				},
			},
			Root:        "b",
			Indexer:     lsifGoImage,
			IndexerArgs: []string{"lsif-go", "--no-animation"},
			Outfile:     "",
		},
		{
			DockerSteps: []DockerStep{
				{
					Root:     "c",
					Image:    lsifGoImage,
					Commands: []string{"go", "mod", "download"},
				},
			},
			Root:        "c",
			Indexer:     lsifGoImage,
			IndexerArgs: []string{"lsif-go", "--no-animation"},
			Outfile:     "",
		},
	}
	if diff := cmp.Diff(expectedIndexJobs, recognizer.InferIndexJobs(paths)); diff != "" {
		t.Errorf("unexpected index jobs (-want +got):\n%s", diff)
	}
}

func TestLSIFGoJobRecognizerPatterns(t *testing.T) {
	recognizer := lsifGoJobRecognizer{}
	paths := []string{
		"go.mod",
		"subdir/go.mod",
		"vendor/foo/go.mod",
		"foo/vendor/go.mod",
		"foo/go.mod/vendor",
	}

	for _, path := range paths {
		match := false
		for _, pattern := range recognizer.Patterns() {
			if pattern.MatchString(path) {
				match = true
				break
			}
		}

		if !match {
			t.Error(fmt.Sprintf("failed to match %s", path))
		}
	}
}
