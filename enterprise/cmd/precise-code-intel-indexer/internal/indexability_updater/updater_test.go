package indexabilityupdater

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
	"golang.org/x/time/rate"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Verbose() {
		log15.Root().SetHandler(log15.DiscardHandler())
	}
	os.Exit(m.Run())
}

func TestUpdate(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockStore.RepoUsageStatisticsFunc.SetDefaultReturn([]store.RepoUsageStatistics{
		{RepositoryID: 1, SearchCount: 200, PreciseCount: 50},
		{RepositoryID: 2, SearchCount: 150, PreciseCount: 25},
		{RepositoryID: 3, SearchCount: 100, PreciseCount: 35},
		{RepositoryID: 4, SearchCount: 50, PreciseCount: 100},
	}, nil)

	mockGitserverClient := NewMockGitserverClient()
	mockGitserverClient.HeadFunc.SetDefaultHook(func(ctx context.Context, store store.Store, repositoryID int) (string, error) {
		return fmt.Sprintf("c%d", repositoryID), nil
	})
	mockGitserverClient.ListFilesFunc.SetDefaultHook(func(ctx context.Context, store store.Store, repositoryID int, commit string, pattern *regexp.Regexp) ([]string, error) {
		if repositoryID%2 == 0 {
			return []string{"go.mod"}, nil
		}
		return nil, nil
	})

	updater := &Updater{
		store:              mockStore,
		gitserverClient:    mockGitserverClient,
		metrics:            NewUpdaterMetrics(metrics.TestRegisterer),
		limiter:            rate.NewLimiter(MaxGitserverRequestsPerSecond, 1),
		enableIndexingCNCF: false,
	}

	if err := updater.Handle(context.Background()); err != nil {
		t.Fatalf("unexpected error performing update: %s", err)
	}

	if len(mockGitserverClient.ListFilesFunc.History()) != 4 {
		t.Errorf("unexpected number of calls to ListFiles. want=%d have=%d", 2, len(mockGitserverClient.ListFilesFunc.History()))
	} else {
		var repositoryIDs []int
		for _, call := range mockGitserverClient.ListFilesFunc.History() {
			repositoryIDs = append(repositoryIDs, call.Arg2)
			expectedCommit := fmt.Sprintf("c%d", call.Arg2)

			if call.Arg3 != expectedCommit {
				t.Errorf("unexpected commit argument. want=%q have=%q", expectedCommit, call.Arg3)
			}
		}
		sort.Ints(repositoryIDs)

		if diff := cmp.Diff([]int{1, 2, 3, 4}, repositoryIDs); diff != "" {
			t.Errorf("unexpected repository ids (-want +got):\n%s", diff)
		}
	}

	if len(mockStore.UpdateIndexableRepositoryFunc.History()) != 2 {
		t.Errorf("unexpected number of calls to UpdateIndexableRepository. want=%d have=%d", 2, len(mockStore.UpdateIndexableRepositoryFunc.History()))
	} else {
		var repositoryIDs []int
		for _, call := range mockStore.UpdateIndexableRepositoryFunc.History() {
			repositoryIDs = append(repositoryIDs, call.Arg1.RepositoryID)
		}
		sort.Ints(repositoryIDs)

		if diff := cmp.Diff([]int{2, 4}, repositoryIDs); diff != "" {
			t.Errorf("unexpected repository ids (-want +got):\n%s", diff)
		}
	}

	if len(mockStore.ResetIndexableRepositoriesFunc.History()) != 1 {
		t.Errorf("unexpected number of calls to ResetIndexableRepositories. want=%d have=%d", 2, len(mockStore.ResetIndexableRepositoriesFunc.History()))
	}
}
