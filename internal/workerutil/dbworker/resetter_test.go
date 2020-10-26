package dbworker

import (
	"testing"
	"time"

	"github.com/efritz/glock"
	"github.com/prometheus/client_golang/prometheus"
	storemocks "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store/mocks"
)

func TestResetter(t *testing.T) {
	store := storemocks.NewMockStore()
	clock := glock.NewMockClock()
	options := ResetterOptions{
		Interval: time.Second,
		Metrics: ResetterMetrics{
			RecordResets:        prometheus.NewCounter(prometheus.CounterOpts{}),
			RecordResetFailures: prometheus.NewCounter(prometheus.CounterOpts{}),
			Errors:              prometheus.NewCounter(prometheus.CounterOpts{}),
		},
	}

	resetter := newResetter(store, options, clock)
	go func() { resetter.Start() }()
	clock.BlockingAdvance(time.Second)
	resetter.Stop()

	if callCount := len(store.ResetStalledFunc.History()); callCount < 1 {
		t.Errorf("unexpected reset stalled call count. want>=%d have=%d", 1, callCount)
	}
}
