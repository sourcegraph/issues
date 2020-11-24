package background

import (
	"context"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
)

func newHistoryMigrator(ctx context.Context, campaignsStore *campaigns.Store) goroutine.BackgroundRoutine {
	migrateHistory := goroutine.NewHandlerWithErrorMessage("migrate changeset histories", func(ctx context.Context) error {
		for {
			cs, next, err := campaignsStore.ListChangesets(ctx, campaigns.ListChangesetsOpts{WithoutHistory: true, LimitOpts: campaigns.LimitOpts{Limit: 50}})
			if err != nil {
				return err
			}
			for _, c := range cs {
				events, _, err := campaignsStore.ListChangesetEvents(ctx, campaigns.ListChangesetEventsOpts{ChangesetIDs: []int64{c.ID}})
				if err != nil {
					log15.Warn("failed to migrate changeset history", "err", errors.Wrap(err, "fetch events"))
					continue
				}
				campaigns.SetDerivedState(ctx, c, events)
				if err := campaignsStore.UpdateChangeset(ctx, c); err != nil {
					log15.Warn("failed to migrate changeset history", "err", errors.Wrap(err, "update changeset"))
					continue
				}
			}
			if next == 0 {
				break
			}
		}
		return nil
	})
	return goroutine.NewPeriodicGoroutine(ctx, 2*time.Minute, migrateHistory)
}
