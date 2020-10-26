package campaigns

import (
	"sort"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
)

// changesetHistory is a collection of external changeset states
// (open/closed/merged state and review state) over time.
type changesetHistory []changesetStatesAtTime

// StatesAtTime returns the changeset's states valid at the given time. If the
// changeset didn't exist yet, the second parameter is false.
func (h changesetHistory) StatesAtTime(t time.Time) (changesetStatesAtTime, bool) {
	if len(h) == 0 {
		return changesetStatesAtTime{}, false
	}

	var (
		states changesetStatesAtTime
		found  bool
	)

	for _, s := range h {
		if s.t.After(t) {
			break
		}
		states = s
		found = true
	}

	return states, found
}

type changesetStatesAtTime struct {
	t             time.Time
	externalState campaigns.ChangesetExternalState
	reviewState   campaigns.ChangesetReviewState
}

// computeHistory calculates the changesetHistory for the given Changeset and
// its ChangesetEvents.
// The ChangesetEvents MUST be sorted by their Timestamp.
func computeHistory(ch *campaigns.Changeset, ce ChangesetEvents) (changesetHistory, error) {
	if !sort.IsSorted(ce) {
		return nil, errors.New("changeset events not sorted")
	}

	var (
		states = []changesetStatesAtTime{}

		currentExtState    = initialExternalState(ch, ce)
		currentReviewState = campaigns.ChangesetReviewStatePending

		lastReviewByAuthor = map[string]campaigns.ChangesetReviewState{}
	)

	pushStates := func(t time.Time) {
		states = append(states, changesetStatesAtTime{
			t:             t,
			externalState: currentExtState,
			reviewState:   currentReviewState,
		})
	}

	openedAt := ch.ExternalCreatedAt()
	if openedAt.IsZero() {
		return nil, errors.New("changeset ExternalCreatedAt has zero value")
	}
	pushStates(openedAt)

	for _, e := range ce {
		et := e.Timestamp()
		if et.IsZero() {
			continue
		}

		switch e.Kind {
		case campaigns.ChangesetEventKindGitHubClosed,
			campaigns.ChangesetEventKindBitbucketServerDeclined,
			campaigns.ChangesetEventKindGitLabClosed:
			// Merged is a final state. We can ignore everything after.
			if currentExtState != campaigns.ChangesetExternalStateMerged {
				currentExtState = campaigns.ChangesetExternalStateClosed
				pushStates(et)
			}

		case campaigns.ChangesetEventKindGitHubMerged,
			campaigns.ChangesetEventKindBitbucketServerMerged,
			campaigns.ChangesetEventKindGitLabMerged:
			currentExtState = campaigns.ChangesetExternalStateMerged
			pushStates(et)

		case campaigns.ChangesetEventKindGitLabMarkWorkInProgress:
			// This event only matters when the changeset is open, otherwise a change in the title won't change the overall external state.
			if currentExtState == campaigns.ChangesetExternalStateOpen {
				currentExtState = campaigns.ChangesetExternalStateDraft
				pushStates(et)
			}

		case campaigns.ChangesetEventKindGitHubConvertToDraft:
			// Merged is a final state. We can ignore everything after.
			if currentExtState != campaigns.ChangesetExternalStateMerged {
				currentExtState = campaigns.ChangesetExternalStateDraft
				pushStates(et)
			}

		case campaigns.ChangesetEventKindGitLabUnmarkWorkInProgress:
			// This event only matters when the changeset is open, otherwise a change in the title won't change the overall external state.
			if currentExtState == campaigns.ChangesetExternalStateDraft {
				currentExtState = campaigns.ChangesetExternalStateOpen
				pushStates(et)
			}

		case campaigns.ChangesetEventKindGitHubReopened,
			campaigns.ChangesetEventKindBitbucketServerReopened,
			campaigns.ChangesetEventKindGitLabReopened,
			campaigns.ChangesetEventKindGitHubReadyForReview:
			// Merged is a final state. We can ignore everything after.
			if currentExtState != campaigns.ChangesetExternalStateMerged {
				currentExtState = campaigns.ChangesetExternalStateOpen
				pushStates(et)
			}

		case campaigns.ChangesetEventKindGitHubReviewed,
			campaigns.ChangesetEventKindBitbucketServerApproved,
			campaigns.ChangesetEventKindBitbucketServerReviewed,
			campaigns.ChangesetEventKindGitLabApproved:

			s, err := e.ReviewState()
			if err != nil {
				return nil, err
			}

			// We only care about "Approved", "ChangesRequested" or "Dismissed" reviews
			if s != campaigns.ChangesetReviewStateApproved &&
				s != campaigns.ChangesetReviewStateChangesRequested &&
				s != campaigns.ChangesetReviewStateDismissed {
				continue
			}

			author := e.ReviewAuthor()
			// If the user has been deleted, skip their reviews, as they don't count towards the final state anymore.
			if author == "" {
				continue
			}

			// Save current review state, then insert new review or delete
			// dismissed review, then recompute overall review state
			oldReviewState := currentReviewState

			if s == campaigns.ChangesetReviewStateDismissed {
				// In case of a dismissed review we dismiss _all_ of the
				// previous reviews by the author, since that is what GitHub
				// does in its UI.
				delete(lastReviewByAuthor, author)
			} else {
				lastReviewByAuthor[author] = s
			}

			newReviewState := reduceReviewStates(lastReviewByAuthor)

			if newReviewState != oldReviewState {
				currentReviewState = newReviewState
				pushStates(et)
			}

		case campaigns.ChangesetEventKindGitHubReviewDismissed:
			// We specifically ignore ChangesetEventKindGitHubReviewDismissed
			// events since GitHub updates the original
			// ChangesetEventKindGitHubReviewed event when a review has been
			// dismissed.
			// See: https://github.com/sourcegraph/sourcegraph/pull/9461
			continue

		case campaigns.ChangesetEventKindBitbucketServerUnapproved,
			campaigns.ChangesetEventKindBitbucketServerDismissed,
			campaigns.ChangesetEventKindGitLabUnapproved:
			author := e.ReviewAuthor()
			// If the user has been deleted, skip their reviews, as they don't count towards the final state anymore.
			if author == "" {
				continue
			}

			if e.Type() == campaigns.ChangesetEventKindBitbucketServerUnapproved {
				// A BitbucketServer Unapproved can only follow a previous Approved by
				// the same author.
				lastReview, ok := lastReviewByAuthor[author]
				if !ok || lastReview != campaigns.ChangesetReviewStateApproved {
					log15.Warn("Bitbucket Server Unapproval not following an Approval", "event", e)
					continue
				}
			}

			if e.Type() == campaigns.ChangesetEventKindBitbucketServerDismissed {
				// A BitbucketServer Dismissed event can only follow a previous "Changes Requested" review by
				// the same author.
				lastReview, ok := lastReviewByAuthor[author]
				if !ok || lastReview != campaigns.ChangesetReviewStateChangesRequested {
					log15.Warn("Bitbucket Server Dismissal not following a Review", "event", e)
					continue
				}
			}

			// Save current review state, then remove last approval and
			// recompute overall review state
			oldReviewState := currentReviewState
			delete(lastReviewByAuthor, author)
			newReviewState := reduceReviewStates(lastReviewByAuthor)

			if newReviewState != oldReviewState {
				currentReviewState = newReviewState
				pushStates(et)
			}
		}
	}

	// We don't have an event for the deletion of a Changeset, but we set
	// ExternalDeletedAt manually in the Syncer.
	deletedAt := ch.ExternalDeletedAt
	if !deletedAt.IsZero() {
		currentExtState = campaigns.ChangesetExternalStateClosed
		pushStates(deletedAt)
	}

	return states, nil
}

// reduceReviewStates reduces the given a map of review per author down to a
// single overall ChangesetReviewState.
func reduceReviewStates(statesByAuthor map[string]campaigns.ChangesetReviewState) campaigns.ChangesetReviewState {
	states := make(map[campaigns.ChangesetReviewState]bool)
	for _, s := range statesByAuthor {
		states[s] = true
	}
	return selectReviewState(states)
}

// initialExternalState infers from the changeset state and the list of events in which
// ChangesetExternalState the changeset must have been when it has been created.
func initialExternalState(ch *campaigns.Changeset, ce ChangesetEvents) campaigns.ChangesetExternalState {
	open := true
	switch m := ch.Metadata.(type) {
	case *github.PullRequest:
		if m.IsDraft {
			open = false
		}

	case *gitlab.MergeRequest:
		if m.WorkInProgress {
			open = false
		}
	default:
		return campaigns.ChangesetExternalStateOpen
	}
	// Walk the events backwards, since we need to look from the current time to the past.
	for i := len(ce) - 1; i >= 0; i-- {
		e := ce[i]
		switch e.Metadata.(type) {
		case *gitlab.UnmarkWorkInProgressEvent, *github.ReadyForReviewEvent:
			open = false
		case *gitlab.MarkWorkInProgressEvent, *github.ConvertToDraftEvent:
			open = true
		}
	}
	if open {
		return campaigns.ChangesetExternalStateOpen
	}
	return campaigns.ChangesetExternalStateDraft
}
