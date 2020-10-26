package janitor

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	lsifstoremocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/lsifstore/mocks"
	storemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store/mocks"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
)

func TestRemoveOrphanedData(t *testing.T) {
	var dumpIDs []int
	for i := 0; i < orphanBatchSize*5-1; i++ {
		dumpIDs = append(dumpIDs, i)
	}

	states := map[int]string{}
	for _, i := range dumpIDs {
		if i%5 == 0 {
			continue
		}

		states[i] = "completed"
	}

	mockStore := storemocks.NewMockStore()
	mockLSIFStore := lsifstoremocks.NewMockStore()
	mockStore.GetStatesFunc.SetDefaultReturn(states, nil)
	mockLSIFStore.DumpIDsFunc.SetDefaultHook(func(ctx context.Context, limit, offset int) ([]int, error) {
		n := offset + limit
		if n > len(dumpIDs) {
			n = len(dumpIDs)
		}
		return dumpIDs[offset:n], nil
	})

	j := &Janitor{
		store:     mockStore,
		lsifStore: mockLSIFStore,
		metrics:   NewJanitorMetrics(metrics.TestRegisterer),
	}
	j.removeOrphanedData(context.Background())

	if len(mockStore.GetStatesFunc.History()) != 5 {
		t.Errorf("unexpected number of GetStatesFunc calls. want=%d have=%d", 5, len(mockStore.GetStatesFunc.History()))
	} else {
		var ids []int
		for _, call := range mockStore.GetStatesFunc.History() {
			ids = append(ids, call.Arg1...)
		}

		if diff := cmp.Diff(dumpIDs, ids); diff != "" {
			t.Errorf("unexpected dump ids (-want +got):\n%s", diff)
		}
	}

	if len(mockLSIFStore.ClearFunc.History()) != 100 {
		t.Errorf("unexpected number of ClearFunc calls. want=%d have=%d", 100, len(mockLSIFStore.ClearFunc.History()))
	} else {
		var ids []int
		for _, call := range mockLSIFStore.ClearFunc.History() {
			ids = append(ids, call.Arg1...)
		}

		var expectedIDs []int
		for _, i := range dumpIDs {
			if i%5 == 0 {
				expectedIDs = append(expectedIDs, i)
			}
		}

		if diff := cmp.Diff(expectedIDs, ids); diff != "" {
			t.Errorf("unexpected dump ids (-want +got):\n%s", diff)
		}
	}
}
