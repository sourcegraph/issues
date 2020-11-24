package campaigns

import (
	"fmt"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/campaigns"
)

// ChangesetCounts represents the states in which a given set of Changesets was
// at a given point in time
type ChangesetCounts struct {
	Time                 time.Time
	Total                int32
	Merged               int32
	Closed               int32
	Draft                int32
	Open                 int32
	OpenApproved         int32
	OpenChangesRequested int32
	OpenPending          int32
}

func (cc *ChangesetCounts) String() string {
	return fmt.Sprintf("%s (Total: %d, Merged: %d, Closed: %d, Draft: %d, Open: %d, OpenApproved: %d, OpenChangesRequested: %d, OpenPending: %d)",
		cc.Time.String(),
		cc.Total,
		cc.Merged,
		cc.Closed,
		cc.Draft,
		cc.Open,
		cc.OpenApproved,
		cc.OpenChangesRequested,
		cc.OpenPending,
	)
}

// CalcCounts calculates ChangesetCounts for the given Changesets and their
// ChangesetEvents in the timeframe specified by the start and end parameters.
// The number of ChangesetCounts returned is the number of 1 day intervals
// between start and end, with each ChangesetCounts representing a point in
// time at the boundary of each 24h interval.
func CalcCounts(start, end time.Time, chs []*campaigns.ChangesetHistory) ([]*ChangesetCounts, error) {
	ts := generateTimestamps(start, end)
	counts := make([]*ChangesetCounts, len(ts))
	for i, t := range ts {
		counts[i] = &ChangesetCounts{Time: t}
	}

	for _, history := range chs {
		// Go through every point in time we want to record and check the
		// states of the changeset at that point in time
		afterIdx := 0
		for _, c := range counts {
			var states campaigns.ChangesetStatesAtTime
			var ok bool
			states, afterIdx, ok = history.StatesAtTime(c.Time, afterIdx)
			if !ok {
				// Changeset didn't exist yet
				continue
			}

			c.Total++
			switch states.ExternalState {
			case campaigns.ChangesetExternalStateDraft:
				c.Draft++
			case campaigns.ChangesetExternalStateOpen:
				c.Open++
				switch states.ReviewState {
				case campaigns.ChangesetReviewStatePending:
					c.OpenPending++
				case campaigns.ChangesetReviewStateApproved:
					c.OpenApproved++
				case campaigns.ChangesetReviewStateChangesRequested:
					c.OpenChangesRequested++
				}

			case campaigns.ChangesetExternalStateMerged:
				c.Merged++
			case campaigns.ChangesetExternalStateClosed:
				c.Closed++
			}

		}
	}

	return counts, nil
}

func generateTimestamps(start, end time.Time) []time.Time {
	// Walk backwards from `end` to >= `start` in 1 day intervals
	// Backwards so we always end exactly on `end`
	ts := []time.Time{}
	for t := end; !t.Before(start); t = t.AddDate(0, 0, -1) {
		ts = append(ts, t)
	}

	// Now reverse so we go from oldest to newest in slice
	for i := len(ts)/2 - 1; i >= 0; i-- {
		opp := len(ts) - 1 - i
		ts[i], ts[opp] = ts[opp], ts[i]
	}

	return ts
}
