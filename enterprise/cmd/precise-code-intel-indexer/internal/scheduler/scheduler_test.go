package scheduler

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	storemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store/mocks"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Verbose() {
		log15.Root().SetHandler(log15.DiscardHandler())
	}
	os.Exit(m.Run())
}

func TestUpdateIndexConfigurationInDatabase(t *testing.T) {
	indexConfiguration := store.IndexConfiguration{
		ID:           1,
		RepositoryID: 42,
		Data: []byte(`{
			"shared_steps": [
				{
					"root": "/",
					"image": "node:12",
					"commands": [
						"yarn install --frozen-lockfile --non-interactive",
					],
				}
			],
			"index_jobs": [
				{
					"steps": [
						{
							// Comments are the future
							"image": "go:latest",
							"commands": ["go mod vendor"],
						}
					],
					"indexer": "lsif-go",
					"indexer_args": ["--no-animation"],
				},
				{
					"root": "web/",
					"indexer": "lsif-tsc",
					"indexer_args": ["-p", "."],
					"outfile": "lsif.dump",
				},
			]
		}`),
	}

	mockStore := storemocks.NewMockStore()
	mockStore.TransactFunc.SetDefaultReturn(mockStore, nil)
	mockStore.GetRepositoriesWithIndexConfigurationFunc.SetDefaultReturn([]int{42}, nil)
	mockStore.GetIndexConfigurationByRepositoryIDFunc.SetDefaultReturn(indexConfiguration, true, nil)

	mockGitserverClient := NewMockGitserverClient()
	mockGitserverClient.HeadFunc.SetDefaultHook(func(ctx context.Context, store store.Store, repositoryID int) (string, error) {
		return fmt.Sprintf("c%d", repositoryID), nil
	})

	scheduler := &Scheduler{
		store:           mockStore,
		gitserverClient: mockGitserverClient,
		metrics:         NewSchedulerMetrics(metrics.TestRegisterer),
	}

	if err := scheduler.Handle(context.Background()); err != nil {
		t.Fatalf("unexpected error performing update: %s", err)
	}

	if len(mockStore.GetIndexConfigurationByRepositoryIDFunc.History()) != 1 {
		t.Errorf("unexpected number of calls to GetIndexConfigurationByRepositoryID. want=%d have=%d", 1, len(mockStore.GetIndexConfigurationByRepositoryIDFunc.History()))
	} else {
		var repositoryIDs []int
		for _, call := range mockStore.GetIndexConfigurationByRepositoryIDFunc.History() {
			repositoryIDs = append(repositoryIDs, call.Arg1)
		}
		sort.Ints(repositoryIDs)

		if diff := cmp.Diff([]int{42}, repositoryIDs); diff != "" {
			t.Errorf("unexpected repository identifiers (-want +got):\n%s", diff)
		}
	}

	if len(mockStore.IsQueuedFunc.History()) != 1 {
		t.Errorf("unexpected number of calls to IsQueued. want=%d have=%d", 1, len(mockStore.IsQueuedFunc.History()))
	} else {
		var commits []string
		for _, call := range mockStore.IsQueuedFunc.History() {
			commits = append(commits, call.Arg2)
		}
		sort.Strings(commits)

		if diff := cmp.Diff([]string{"c42"}, commits); diff != "" {
			t.Errorf("unexpected commits (-want +got):\n%s", diff)
		}
	}

	if len(mockStore.InsertIndexFunc.History()) != 2 {
		t.Errorf("unexpected number of calls to InsertIndex. want=%d have=%d", 2, len(mockStore.InsertIndexFunc.History()))
	} else {
		var indexes []store.Index
		for _, call := range mockStore.InsertIndexFunc.History() {
			indexes = append(indexes, call.Arg1)
		}

		expectedIndexes := []store.Index{
			{
				RepositoryID: 42,
				Commit:       "c42",
				State:        "queued",
				DockerSteps: []store.DockerStep{
					{
						Root:     "/",
						Image:    "node:12",
						Commands: []string{"yarn install --frozen-lockfile --non-interactive"},
					},
					{
						Image:    "go:latest",
						Commands: []string{"go mod vendor"},
					},
				},
				Indexer:     "lsif-go",
				IndexerArgs: []string{"--no-animation"},
			},
			{
				RepositoryID: 42,
				Commit:       "c42",
				State:        "queued",
				DockerSteps: []store.DockerStep{
					{
						Root:     "/",
						Image:    "node:12",
						Commands: []string{"yarn install --frozen-lockfile --non-interactive"},
					},
				},
				Root:        "web/",
				Indexer:     "lsif-tsc",
				IndexerArgs: []string{"-p", "."},
				Outfile:     "lsif.dump",
			},
		}
		if diff := cmp.Diff(expectedIndexes, indexes); diff != "" {
			t.Errorf("unexpected indexes (-want +got):\n%s", diff)
		}
	}
}

var yamlIndexConfiguration = []byte(`
shared_steps:
  - root: /
    image: node:12
    commands:
      - yarn install --frozen-lockfile --non-interactive

index_jobs:
  -
    steps:
      - image: go:latest
        commands:
          - go mod vendor
    indexer: lsif-go
    indexer_args:
      - --no-animation
  -
    root: web/
    indexer: lsif-tsc
    indexer_args: ['-p', '.']
    outfile: lsif.dump
`)

func TestUpdateIndexConfigurationInRepository(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockStore.TransactFunc.SetDefaultReturn(mockStore, nil)
	mockStore.GetRepositoriesWithIndexConfigurationFunc.SetDefaultReturn([]int{42}, nil)

	mockGitserverClient := NewMockGitserverClient()
	mockGitserverClient.HeadFunc.SetDefaultHook(func(ctx context.Context, store store.Store, repositoryID int) (string, error) {
		return fmt.Sprintf("c%d", repositoryID), nil
	})
	mockGitserverClient.FileExistsFunc.SetDefaultHook(func(ctx context.Context, store store.Store, repositoryID int, commit, file string) (bool, error) {
		return file == "sourcegraph.yaml", nil
	})
	mockGitserverClient.RawContentsFunc.SetDefaultReturn(yamlIndexConfiguration, nil)

	scheduler := &Scheduler{
		store:           mockStore,
		gitserverClient: mockGitserverClient,
		metrics:         NewSchedulerMetrics(metrics.TestRegisterer),
	}

	if err := scheduler.Handle(context.Background()); err != nil {
		t.Fatalf("unexpected error performing update: %s", err)
	}

	if len(mockStore.IsQueuedFunc.History()) != 1 {
		t.Errorf("unexpected number of calls to IsQueued. want=%d have=%d", 1, len(mockStore.IsQueuedFunc.History()))
	} else {
		var commits []string
		for _, call := range mockStore.IsQueuedFunc.History() {
			commits = append(commits, call.Arg2)
		}
		sort.Strings(commits)

		if diff := cmp.Diff([]string{"c42"}, commits); diff != "" {
			t.Errorf("unexpected commits (-want +got):\n%s", diff)
		}
	}

	if len(mockStore.InsertIndexFunc.History()) != 2 {
		t.Errorf("unexpected number of calls to InsertIndex. want=%d have=%d", 2, len(mockStore.InsertIndexFunc.History()))
	} else {
		var indexes []store.Index
		for _, call := range mockStore.InsertIndexFunc.History() {
			indexes = append(indexes, call.Arg1)
		}

		expectedIndexes := []store.Index{
			{
				RepositoryID: 42,
				Commit:       "c42",
				State:        "queued",
				DockerSteps: []store.DockerStep{
					{
						Root:     "/",
						Image:    "node:12",
						Commands: []string{"yarn install --frozen-lockfile --non-interactive"},
					},
					{
						Image:    "go:latest",
						Commands: []string{"go mod vendor"},
					},
				},
				Indexer:     "lsif-go",
				IndexerArgs: []string{"--no-animation"},
			},
			{
				RepositoryID: 42,
				Commit:       "c42",
				State:        "queued",
				DockerSteps: []store.DockerStep{
					{
						Root:     "/",
						Image:    "node:12",
						Commands: []string{"yarn install --frozen-lockfile --non-interactive"},
					},
				},
				Root:        "web/",
				Indexer:     "lsif-tsc",
				IndexerArgs: []string{"-p", "."},
				Outfile:     "lsif.dump",
			},
		}
		if diff := cmp.Diff(expectedIndexes, indexes); diff != "" {
			t.Errorf("unexpected indexes (-want +got):\n%s", diff)
		}
	}
}

func TestUpdateIndexConfigurationInferred(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockStore.TransactFunc.SetDefaultReturn(mockStore, nil)
	mockStore.IndexableRepositoriesFunc.SetDefaultReturn([]store.IndexableRepository{
		{RepositoryID: 41},
		{RepositoryID: 42},
		{RepositoryID: 43},
		{RepositoryID: 44},
	}, nil)
	mockStore.IsQueuedFunc.SetDefaultHook(func(ctx context.Context, repositoryID int, commit string) (bool, error) {
		return repositoryID%2 != 0, nil
	})

	mockGitserverClient := NewMockGitserverClient()
	mockGitserverClient.HeadFunc.SetDefaultHook(func(ctx context.Context, store store.Store, repositoryID int) (string, error) {
		return fmt.Sprintf("c%d", repositoryID), nil
	})
	mockGitserverClient.ListFilesFunc.SetDefaultHook(func(ctx context.Context, store store.Store, repositoryID int, commit string, pattern *regexp.Regexp) ([]string, error) {
		switch repositoryID {
		case 42:
			return []string{"go.mod"}, nil
		case 44:
			return []string{"a/go.mod", "b/go.mod"}, nil
		default:
			return nil, nil
		}
	})

	scheduler := &Scheduler{
		store:           mockStore,
		gitserverClient: mockGitserverClient,
		metrics:         NewSchedulerMetrics(metrics.TestRegisterer),
	}

	if err := scheduler.Handle(context.Background()); err != nil {
		t.Fatalf("unexpected error performing update: %s", err)
	}

	if len(mockStore.IsQueuedFunc.History()) != 4 {
		t.Errorf("unexpected number of calls to IsQueued. want=%d have=%d", 4, len(mockStore.IsQueuedFunc.History()))
	} else {
		var commits []string
		for _, call := range mockStore.IsQueuedFunc.History() {
			commits = append(commits, call.Arg2)
		}
		sort.Strings(commits)

		if diff := cmp.Diff([]string{"c41", "c42", "c43", "c44"}, commits); diff != "" {
			t.Errorf("unexpected commits (-want +got):\n%s", diff)
		}
	}

	if len(mockStore.InsertIndexFunc.History()) != 3 {
		t.Errorf("unexpected number of calls to InsertIndex. want=%d have=%d", 3, len(mockStore.InsertIndexFunc.History()))
	} else {
		indexRoots := map[int][]string{}
		for _, call := range mockStore.InsertIndexFunc.History() {
			indexRoots[call.Arg1.RepositoryID] = append(indexRoots[call.Arg1.RepositoryID], call.Arg1.Root)
		}

		expectedIndexRoots := map[int][]string{
			42: {""},
			44: {"a", "b"},
		}
		if diff := cmp.Diff(expectedIndexRoots, indexRoots); diff != "" {
			t.Errorf("unexpected indexes (-want +got):\n%s", diff)
		}
	}

	if len(mockStore.UpdateIndexableRepositoryFunc.History()) != 2 {
		t.Errorf("unexpected number of calls to UpdateIndexableRepository. want=%d have=%d", 2, len(mockStore.UpdateIndexableRepositoryFunc.History()))
	}
}
