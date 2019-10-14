package a8n

import (
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/a8n"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
)

type ChangesetCounts struct {
	Time                 time.Time
	Total                int32
	Merged               int32
	Closed               int32
	Open                 int32
	OpenApproved         int32
	OpenChangesRequested int32
	OpenPending          int32
}

func (cc *ChangesetCounts) String() string {
	return fmt.Sprintf("%s (Total: %d, Merged: %d, Closed: %d, Open: %d, OpenApproved: %d, OpenChangesRequested: %d, OpenPending: %d)",
		cc.Time.String(),
		cc.Total,
		cc.Merged,
		cc.Closed,
		cc.Open,
		cc.OpenApproved,
		cc.OpenChangesRequested,
		cc.OpenPending,
	)
}

type Event interface {
	Timestamp() time.Time
	Type() a8n.ChangesetEventKind
	Changeset() int64
}

type Events []Event

func (es Events) Len() int      { return len(es) }
func (es Events) Swap(i, j int) { es[i], es[j] = es[j], es[i] }

// Less sorts events by their timestamps
func (es Events) Less(i, j int) bool {
	return es[i].Timestamp().Before(es[j].Timestamp())
}

func CalcCounts(start, end time.Time, cs []*a8n.Changeset, es ...Event) ([]*ChangesetCounts, error) {
	ts := generateTimestamps(start, end)
	counts := make([]*ChangesetCounts, len(ts))
	for i, t := range ts {
		counts[i] = &ChangesetCounts{Time: t}
	}

	// Sort all events once by their timestamps
	events := Events(es)
	sort.Sort(events)

	// Map sorted events to their changesets
	byChangeset := make(map[*a8n.Changeset]Events)
	for _, c := range cs {
		group := Events{}
		for _, e := range events {
			if e.Changeset() == c.ID {
				group = append(group, e)
			}
		}
		byChangeset[c] = group
	}

	for c, csEvents := range byChangeset {
		// We don't have an event for "open", so we check when it was
		// created on codehost
		openedAt := c.ExternalCreatedAt()
		if openedAt.IsZero() {
			continue
		}

		// For each changeset and its events, go through every point in time we
		// want to record the counts for and reconstruct the state of the changeset at that
		// point in time
		for _, c := range counts {
			if openedAt.After(c.Time) {
				// No need to look at events if changeset was not created yet
				continue
			}

			err := computeCounts(c, csEvents)
			if err != nil {
				return counts, err
			}
		}
	}

	return counts, nil
}

func computeCounts(c *ChangesetCounts, csEvents Events) error {
	// Since some events cancel out another events effects we need to keep track of the
	// changesets state up until an event so we know what to revert
	// i.e. "merge" decrements OpenApproved counts, but only if
	// changeset was previously approved
	var (
		closed = false
	)

	c.Total++
	c.Open++
	c.OpenPending++

	lastReviewByAuthor := map[string]a8n.ChangesetReviewState{}

	for _, e := range csEvents {
		// Event happened after point in time we're looking at, no need to look
		// at the events in future
		et := e.Timestamp()
		if et.IsZero() || et.After(c.Time) {
			return nil
		}

		// Compute previous overall review state
		previousReviewState := computeReviewState(lastReviewByAuthor)

		switch e.Type() {
		case a8n.ChangesetEventKindGitHubClosed:
			c.Open--
			c.Closed++
			closed = true

			switch previousReviewState {
			case a8n.ChangesetReviewStatePending:
				c.OpenPending--
			case a8n.ChangesetReviewStateApproved:
				c.OpenApproved--
			case a8n.ChangesetReviewStateChangesRequested:
				c.OpenChangesRequested--
			}

		case a8n.ChangesetEventKindGitHubReopened:
			c.Open++
			c.Closed--
			closed = false

			switch previousReviewState {
			case a8n.ChangesetReviewStatePending:
				c.OpenPending++
			case a8n.ChangesetReviewStateApproved:
				c.OpenApproved++
			case a8n.ChangesetReviewStateChangesRequested:
				c.OpenChangesRequested++
			}

		case a8n.ChangesetEventKindGitHubMerged:
			// If it was closed, all "review counts" have been updated by the
			// closed events and we just need to reverse these two counts
			if closed {
				c.Closed--
				c.Merged++
				return nil
			}
			switch previousReviewState {
			case a8n.ChangesetReviewStatePending:
				c.OpenPending--
			case a8n.ChangesetReviewStateApproved:
				c.OpenApproved--
			case a8n.ChangesetReviewStateChangesRequested:
				c.OpenChangesRequested--
			}
			c.Merged++
			c.Open--

			// Merged is a final state, we return here and don't need to look at
			// other events
			return nil

		case a8n.ChangesetEventKindGitHubReviewed:
			s, err := reviewState(e)
			if err != nil {
				return err
			}

			author, err := reviewAuthor(e)
			if err != nil {
				return err
			}

			// Insert new review, potentially replacing old review, but only if
			// it's not "PENDING" or "COMMENTED"
			if s == a8n.ChangesetReviewStateApproved || s == a8n.ChangesetReviewStateChangesRequested {
				lastReviewByAuthor[author] = s
			}

			// Compute new overall review state
			newOverallState := computeReviewState(lastReviewByAuthor)

			switch newOverallState {
			case a8n.ChangesetReviewStateApproved:
				switch previousReviewState {
				case a8n.ChangesetReviewStatePending:
					c.OpenApproved++
					c.OpenPending--
				case a8n.ChangesetReviewStateChangesRequested:
					c.OpenChangesRequested--
					c.OpenApproved++
				}

			case a8n.ChangesetReviewStateChangesRequested:
				switch previousReviewState {
				case a8n.ChangesetReviewStatePending:
					c.OpenChangesRequested++
					c.OpenPending--
				case a8n.ChangesetReviewStateApproved:
					c.OpenChangesRequested++
					c.OpenApproved--
				}
			case a8n.ChangesetReviewStatePending:
			case a8n.ChangesetReviewStateCommented:
				// Ignore
			}
		}
	}

	return nil
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

func reviewState(e Event) (a8n.ChangesetReviewState, error) {
	var s a8n.ChangesetReviewState
	changesetEvent, ok := e.(*a8n.ChangesetEvent)
	if !ok {
		return s, errors.New("Reviewed event not ChangesetEvent")
	}

	review, ok := changesetEvent.Metadata.(*github.PullRequestReview)
	if !ok {
		return s, errors.New("ChangesetEvent metadata event not PullRequestReview")
	}

	s = a8n.ChangesetReviewState(review.State)
	if !s.Valid() {
		return s, fmt.Errorf("invalid review state: %s", review.State)
	}
	return s, nil
}

func reviewAuthor(e Event) (string, error) {
	changesetEvent, ok := e.(*a8n.ChangesetEvent)
	if !ok {
		return "", errors.New("Reviewed event not ChangesetEvent")
	}

	review, ok := changesetEvent.Metadata.(*github.PullRequestReview)
	if !ok {
		return "", errors.New("ChangesetEvent metadata event not PullRequestReview")
	}

	login := review.Author.Login
	if login == "" {
		return "", errors.New("review author is blank")
	}

	return login, nil
}

func computeReviewState(statesByAuthor map[string]a8n.ChangesetReviewState) a8n.ChangesetReviewState {
	states := make(map[a8n.ChangesetReviewState]bool)
	for _, s := range statesByAuthor {
		states[s] = true
	}
	return a8n.SelectReviewState(states)
}
