package janitor

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	lsifstoremocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/lsifstore/mocks"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	storemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store/mocks"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
)

func TestHardDeleteDeletedRecords(t *testing.T) {
	bundleDir := testRoot(t)

	mockStore := storemocks.NewMockStore()
	mockLSIFStore := lsifstoremocks.NewMockStore()
	mockStore.TransactFunc.SetDefaultReturn(mockStore, nil)
	mockStore.GetUploadsFunc.PushReturn([]store.Upload{{ID: 1}, {ID: 2}, {ID: 3}}, 5, nil)
	mockStore.GetUploadsFunc.PushReturn([]store.Upload{{ID: 4}, {ID: 5}}, 2, nil)

	j := &Janitor{
		store:     mockStore,
		lsifStore: mockLSIFStore,
		bundleDir: bundleDir,
		metrics:   NewJanitorMetrics(metrics.TestRegisterer),
	}
	j.hardDeleteDeletedRecords(context.Background())

	var ids1 []int
	for _, call := range mockStore.HardDeleteUploadByIDFunc.History() {
		ids1 = append(ids1, call.Arg1...)
	}
	sort.Ints(ids1)

	if diff := cmp.Diff([]int{1, 2, 3, 4, 5}, ids1); diff != "" {
		t.Errorf("unexpected dump ids (-want +got):\n%s", diff)
	}

	var ids2 []int
	for _, call := range mockLSIFStore.ClearFunc.History() {
		ids2 = append(ids2, call.Arg1...)
	}
	sort.Ints(ids2)

	if diff := cmp.Diff([]int{1, 2, 3, 4, 5}, ids2); diff != "" {
		t.Errorf("unexpected dump ids (-want +got):\n%s", diff)
	}
}

func TestRemoveRecordsForDeletedRepositories(t *testing.T) {
	bundleDir := testRoot(t)
	mockStore := storemocks.NewMockStore()

	j := &Janitor{
		store:     mockStore,
		bundleDir: bundleDir,
		metrics:   NewJanitorMetrics(metrics.TestRegisterer),
	}
	j.removeRecordsForDeletedRepositories(context.Background())

	if len(mockStore.DeleteUploadsWithoutRepositoryFunc.History()) != 1 {
		t.Errorf("unexpected number of DeleteUploadsWithoutRepository calls. want=%d have=%d", 1, len(mockStore.DeleteUploadsWithoutRepositoryFunc.History()))
	}
}
