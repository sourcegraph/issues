package heartbeat

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	indexmanager "github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-indexer-vm/internal/index_manager"
	queuemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/queue/client/mocks"
)

func TestHeartbeat(t *testing.T) {
	queueClient := queuemocks.NewMockClient()
	indexManager := indexmanager.New()

	indexManager.AddID(1)
	indexManager.AddID(2)
	indexManager.AddID(4)
	indexManager.AddID(5)

	heartbeater := &Heartbeater{
		queueClient:  queueClient,
		indexManager: indexManager,
	}

	if err := heartbeater.Handle(context.Background()); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	}

	if callCount := len(queueClient.HeartbeatFunc.History()); callCount < 1 {
		t.Errorf("unexpected heartbeat call count. want>=%d have=%d", 1, callCount)
	} else {
		ids := queueClient.HeartbeatFunc.History()[0].Arg1
		sort.Ints(ids)

		if diff := cmp.Diff([]int{1, 2, 4, 5}, ids); diff != "" {
			t.Errorf("unexpected ids (-want +got):\n%s", diff)
		}
	}
}
