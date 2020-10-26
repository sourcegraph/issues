package campaigns

import (
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sourcegraph/go-diff/diff"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	gitlabwebhooks "github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab/webhooks"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

// ChangesetPublicationState defines the possible publication states of a Changeset.
type ChangesetPublicationState string

// ChangesetState constants.
const (
	ChangesetPublicationStateUnpublished ChangesetPublicationState = "UNPUBLISHED"
	ChangesetPublicationStatePublished   ChangesetPublicationState = "PUBLISHED"
)

// Valid returns true if the given ChangesetPublicationState is valid.
func (s ChangesetPublicationState) Valid() bool {
	switch s {
	case ChangesetPublicationStateUnpublished, ChangesetPublicationStatePublished:
		return true
	default:
		return false
	}
}

// Published returns true if the given state is ChangesetPublicationStatePublished.
func (s ChangesetPublicationState) Published() bool { return s == ChangesetPublicationStatePublished }

// Unpublished returns true if the given state is ChangesetPublicationStateUnpublished.
func (s ChangesetPublicationState) Unpublished() bool {
	return s == ChangesetPublicationStateUnpublished
}

// ReconcilerState defines the possible states of a Reconciler.
type ReconcilerState string

// ReconcilerState constants.
const (
	ReconcilerStateQueued     ReconcilerState = "QUEUED"
	ReconcilerStateProcessing ReconcilerState = "PROCESSING"
	ReconcilerStateErrored    ReconcilerState = "ERRORED"
	ReconcilerStateCompleted  ReconcilerState = "COMPLETED"
)

// Valid returns true if the given ReconcilerState is valid.
func (s ReconcilerState) Valid() bool {
	switch s {
	case ReconcilerStateQueued,
		ReconcilerStateProcessing,
		ReconcilerStateErrored,
		ReconcilerStateCompleted:
		return true
	default:
		return false
	}
}

// ToDB returns the database representation of the reconciler state. That's
// needed because we want to use UPPERCASE ReconcilerStates in the application
// and GraphQL layer, but need to use lowercase in the database to make it work
// with workerutil.Worker.
func (s ReconcilerState) ToDB() string { return strings.ToLower(string(s)) }

// ChangesetExternalState defines the possible states of a Changeset on a code host.
type ChangesetExternalState string

// ChangesetExternalState constants.
const (
	ChangesetExternalStateDraft   ChangesetExternalState = "DRAFT"
	ChangesetExternalStateOpen    ChangesetExternalState = "OPEN"
	ChangesetExternalStateClosed  ChangesetExternalState = "CLOSED"
	ChangesetExternalStateMerged  ChangesetExternalState = "MERGED"
	ChangesetExternalStateDeleted ChangesetExternalState = "DELETED"
)

// Valid returns true if the given ChangesetExternalState is valid.
func (s ChangesetExternalState) Valid() bool {
	switch s {
	case ChangesetExternalStateOpen,
		ChangesetExternalStateDraft,
		ChangesetExternalStateClosed,
		ChangesetExternalStateMerged,
		ChangesetExternalStateDeleted:
		return true
	default:
		return false
	}
}

// ChangesetLabel represents a label applied to a changeset
type ChangesetLabel struct {
	Name        string
	Color       string
	Description string
}

// ChangesetReviewState defines the possible states of a Changeset's review.
type ChangesetReviewState string

// ChangesetReviewState constants.
const (
	ChangesetReviewStateApproved         ChangesetReviewState = "APPROVED"
	ChangesetReviewStateChangesRequested ChangesetReviewState = "CHANGES_REQUESTED"
	ChangesetReviewStatePending          ChangesetReviewState = "PENDING"
	ChangesetReviewStateCommented        ChangesetReviewState = "COMMENTED"
	ChangesetReviewStateDismissed        ChangesetReviewState = "DISMISSED"
)

// Valid returns true if the given Changeset review state is valid.
func (s ChangesetReviewState) Valid() bool {
	switch s {
	case ChangesetReviewStateApproved,
		ChangesetReviewStateChangesRequested,
		ChangesetReviewStatePending,
		ChangesetReviewStateCommented,
		ChangesetReviewStateDismissed:
		return true
	default:
		return false
	}
}

// ChangesetCheckState constants.
type ChangesetCheckState string

const (
	ChangesetCheckStateUnknown ChangesetCheckState = "UNKNOWN"
	ChangesetCheckStatePending ChangesetCheckState = "PENDING"
	ChangesetCheckStatePassed  ChangesetCheckState = "PASSED"
	ChangesetCheckStateFailed  ChangesetCheckState = "FAILED"
)

// Valid returns true if the given Changeset check state is valid.
func (s ChangesetCheckState) Valid() bool {
	switch s {
	case ChangesetCheckStateUnknown,
		ChangesetCheckStatePending,
		ChangesetCheckStatePassed,
		ChangesetCheckStateFailed:
		return true
	default:
		return false
	}
}

// A Changeset is a changeset on a code host belonging to a Repository and many
// Campaigns.
type Changeset struct {
	ID                  int64
	RepoID              api.RepoID
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Metadata            interface{}
	CampaignIDs         []int64
	ExternalID          string
	ExternalServiceType string
	ExternalBranch      string
	ExternalDeletedAt   time.Time
	ExternalUpdatedAt   time.Time
	ExternalState       ChangesetExternalState
	ExternalReviewState ChangesetReviewState
	ExternalCheckState  ChangesetCheckState
	DiffStatAdded       *int32
	DiffStatChanged     *int32
	DiffStatDeleted     *int32
	SyncState           ChangesetSyncState

	// The campaign that "owns" this changeset: it can create/close it on code host.
	OwnedByCampaignID int64
	// Whether it was imported/tracked by a campaign.
	AddedToCampaign bool

	// This is 0 if the Changeset isn't owned by Sourcegraph.
	CurrentSpecID  int64
	PreviousSpecID int64

	PublicationState ChangesetPublicationState // "unpublished", "published"

	// All of the following fields are used by workerutil.Worker.
	ReconcilerState ReconcilerState
	FailureMessage  *string
	StartedAt       time.Time
	FinishedAt      time.Time
	ProcessAfter    time.Time
	NumResets       int64
	NumFailures     int64

	// Unsynced is true if the changeset tracks an external changeset but the
	// data hasn't been synced yet.
	Unsynced bool

	// Closing is set to true (along with the ReocncilerState) when the
	// reconciler should close the changeset.
	Closing bool
}

// RecordID is needed to implement the workerutil.Record interface.
func (c *Changeset) RecordID() int { return int(c.ID) }

// Clone returns a clone of a Changeset.
func (c *Changeset) Clone() *Changeset {
	tt := *c
	tt.CampaignIDs = c.CampaignIDs[:len(c.CampaignIDs):len(c.CampaignIDs)]
	return &tt
}

// PublishedAndSynced returns whether the Changeset has been published on the
// code host and is fully synced.
// This can be used as a check before accessing the fields based on synced
// metadata, such as Title or Body, etc.
func (c *Changeset) PublishedAndSynced() bool {
	return !c.Unsynced && c.PublicationState.Published()
}

// Published returns whether the Changeset's PublicationState is Published.
func (c *Changeset) Published() bool { return c.PublicationState.Published() }

// Unpublished returns whether the Changeset's PublicationState is Unpublished.
func (c *Changeset) Unpublished() bool { return c.PublicationState.Unpublished() }

// DiffStat returns a *diff.Stat if DiffStatAdded, DiffStatChanged, and
// DiffStatDeleted are set, or nil if one or more is not.
func (c *Changeset) DiffStat() *diff.Stat {
	if c.DiffStatAdded == nil || c.DiffStatChanged == nil || c.DiffStatDeleted == nil {
		return nil
	}

	return &diff.Stat{
		Added:   *c.DiffStatAdded,
		Changed: *c.DiffStatChanged,
		Deleted: *c.DiffStatDeleted,
	}
}

func (c *Changeset) SetDiffStat(stat *diff.Stat) {
	if stat == nil {
		c.DiffStatAdded = nil
		c.DiffStatChanged = nil
		c.DiffStatDeleted = nil
	} else {
		added := stat.Added
		c.DiffStatAdded = &added

		changed := stat.Changed
		c.DiffStatChanged = &changed

		deleted := stat.Deleted
		c.DiffStatDeleted = &deleted
	}
}

func (c *Changeset) SetMetadata(meta interface{}) error {
	switch pr := meta.(type) {
	case *github.PullRequest:
		c.Metadata = pr
		c.ExternalID = strconv.FormatInt(pr.Number, 10)
		c.ExternalServiceType = extsvc.TypeGitHub
		c.ExternalBranch = pr.HeadRefName
		c.ExternalUpdatedAt = pr.UpdatedAt
	case *bitbucketserver.PullRequest:
		c.Metadata = pr
		c.ExternalID = strconv.FormatInt(int64(pr.ID), 10)
		c.ExternalServiceType = extsvc.TypeBitbucketServer
		c.ExternalBranch = git.AbbreviateRef(pr.FromRef.ID)
		c.ExternalUpdatedAt = unixMilliToTime(int64(pr.UpdatedDate))
	case *gitlab.MergeRequest:
		c.Metadata = pr
		c.ExternalID = strconv.FormatInt(int64(pr.IID), 10)
		c.ExternalServiceType = extsvc.TypeGitLab
		c.ExternalBranch = pr.SourceBranch
		c.ExternalUpdatedAt = pr.UpdatedAt.Time
	default:
		return errors.New("unknown changeset type")
	}
	return nil
}

// RemoveCampaignID removes the given id from the Changesets CampaignIDs slice.
// If the id is not in CampaignIDs calling this method doesn't have an effect.
func (c *Changeset) RemoveCampaignID(id int64) {
	for i := len(c.CampaignIDs) - 1; i >= 0; i-- {
		if c.CampaignIDs[i] == id {
			c.CampaignIDs = append(c.CampaignIDs[:i], c.CampaignIDs[i+1:]...)
		}
	}
}

// Title of the Changeset.
func (c *Changeset) Title() (string, error) {
	switch m := c.Metadata.(type) {
	case *github.PullRequest:
		return m.Title, nil
	case *bitbucketserver.PullRequest:
		return m.Title, nil
	case *gitlab.MergeRequest:
		return m.Title, nil
	default:
		return "", errors.New("unknown changeset type")
	}
}

// ExternalCreatedAt is when the Changeset was created on the codehost. When it
// cannot be determined when the changeset was created, a zero-value timestamp
// is returned.
func (c *Changeset) ExternalCreatedAt() time.Time {
	switch m := c.Metadata.(type) {
	case *github.PullRequest:
		return m.CreatedAt
	case *bitbucketserver.PullRequest:
		return unixMilliToTime(int64(m.CreatedDate))
	case *gitlab.MergeRequest:
		return m.CreatedAt.Time
	default:
		return time.Time{}
	}
}

// Body of the Changeset.
func (c *Changeset) Body() (string, error) {
	switch m := c.Metadata.(type) {
	case *github.PullRequest:
		return m.Body, nil
	case *bitbucketserver.PullRequest:
		return m.Description, nil
	case *gitlab.MergeRequest:
		return m.Description, nil
	default:
		return "", errors.New("unknown changeset type")
	}
}

// SetDeleted sets the internal state of a Changeset so that its State is
// ChangesetStateDeleted.
func (c *Changeset) SetDeleted() {
	c.ExternalDeletedAt = time.Now().UTC().Truncate(time.Microsecond)
}

// IsDeleted returns true when the Changeset's ExternalDeletedAt is a non-zero
// timestamp.
func (c *Changeset) IsDeleted() bool {
	return !c.ExternalDeletedAt.IsZero()
}

// HasDiff returns true when the changeset is in an open state. That is because
// currently we do not support diff rendering for historic branches, because we
// can't guarantee that we have the refs on gitserver.
func (c *Changeset) HasDiff() bool {
	return c.ExternalState == ChangesetExternalStateDraft || c.ExternalState == ChangesetExternalStateOpen
}

// URL of a Changeset.
func (c *Changeset) URL() (s string, err error) {
	switch m := c.Metadata.(type) {
	case *github.PullRequest:
		return m.URL, nil
	case *bitbucketserver.PullRequest:
		if len(m.Links.Self) < 1 {
			return "", errors.New("bitbucketserver pull request has no self links")
		}
		selfLink := m.Links.Self[0]
		return selfLink.Href, nil
	case *gitlab.MergeRequest:
		return m.WebURL, nil
	default:
		return "", errors.New("unknown changeset type")
	}
}

// Events returns the list of ChangesetEvents from the Changeset's metadata.
func (c *Changeset) Events() (events []*ChangesetEvent) {
	switch m := c.Metadata.(type) {
	case *github.PullRequest:
		events = make([]*ChangesetEvent, 0, len(m.TimelineItems))
		for _, ti := range m.TimelineItems {
			ev := ChangesetEvent{ChangesetID: c.ID}

			switch e := ti.Item.(type) {
			case *github.PullRequestReviewThread:
				for _, c := range e.Comments {
					ev := ev
					ev.Key = c.Key()
					ev.Kind = ChangesetEventKindFor(c)
					ev.Metadata = c
					events = append(events, &ev)
				}

			case *github.ReviewRequestedEvent:
				// If the reviewer of a ReviewRequestedEvent has been deleted,
				// the fields are blank and we cannot match the event to an
				// entry in the database and/or reliably use it, so we drop it.
				if e.ReviewerDeleted() {
					continue
				}
				ev.Key = e.Key()
				ev.Kind = ChangesetEventKindFor(e)
				ev.Metadata = e
				events = append(events, &ev)

			default:
				ev.Key = ti.Item.(Keyer).Key()
				ev.Kind = ChangesetEventKindFor(ti.Item)
				ev.Metadata = ti.Item
				events = append(events, &ev)
			}
		}

	case *bitbucketserver.PullRequest:
		events = make([]*ChangesetEvent, 0, len(m.Activities)+len(m.CommitStatus))
		addEvent := func(e Keyer) {
			events = append(events, &ChangesetEvent{
				ChangesetID: c.ID,
				Key:         e.Key(),
				Kind:        ChangesetEventKindFor(e),
				Metadata:    e,
			})
		}
		for _, a := range m.Activities {
			addEvent(a)
		}
		for _, s := range m.CommitStatus {
			addEvent(s)
		}

	case *gitlab.MergeRequest:
		events = make([]*ChangesetEvent, 0, len(m.Notes)+len(m.Pipelines))

		for _, note := range m.Notes {
			if event := note.ToEvent(); event != nil {
				events = append(events, &ChangesetEvent{
					ChangesetID: c.ID,
					Key:         event.(Keyer).Key(),
					Kind:        ChangesetEventKindFor(event),
					Metadata:    event,
				})
			}
		}

		for _, pipeline := range m.Pipelines {
			events = append(events, &ChangesetEvent{
				ChangesetID: c.ID,
				Key:         pipeline.Key(),
				Kind:        ChangesetEventKindFor(pipeline),
				Metadata:    pipeline,
			})
		}
	}
	return events
}

// HeadRefOid returns the git ObjectID of the HEAD reference associated with
// Changeset on the codehost. If the codehost doesn't include the ObjectID, an
// empty string is returned.
func (c *Changeset) HeadRefOid() (string, error) {
	switch m := c.Metadata.(type) {
	case *github.PullRequest:
		return m.HeadRefOid, nil
	case *bitbucketserver.PullRequest:
		return "", nil
	case *gitlab.MergeRequest:
		return m.DiffRefs.HeadSHA, nil
	default:
		return "", errors.New("unknown changeset type")
	}
}

// HeadRef returns the full ref (e.g. `refs/heads/my-branch`) of the
// HEAD reference associated with the Changeset on the codehost.
func (c *Changeset) HeadRef() (string, error) {
	switch m := c.Metadata.(type) {
	case *github.PullRequest:
		return "refs/heads/" + m.HeadRefName, nil
	case *bitbucketserver.PullRequest:
		return m.FromRef.ID, nil
	case *gitlab.MergeRequest:
		return "refs/heads/" + m.SourceBranch, nil
	default:
		return "", errors.New("unknown changeset type")
	}
}

// BaseRefOid returns the git ObjectID of the base reference associated with the
// Changeset on the codehost. If the codehost doesn't include the ObjectID, an
// empty string is returned.
func (c *Changeset) BaseRefOid() (string, error) {
	switch m := c.Metadata.(type) {
	case *github.PullRequest:
		return m.BaseRefOid, nil
	case *bitbucketserver.PullRequest:
		return "", nil
	case *gitlab.MergeRequest:
		return m.DiffRefs.BaseSHA, nil
	default:
		return "", errors.New("unknown changeset type")
	}
}

// BaseRef returns the full ref (e.g. `refs/heads/my-branch`) of the base ref
// associated with the Changeset on the codehost.
func (c *Changeset) BaseRef() (string, error) {
	switch m := c.Metadata.(type) {
	case *github.PullRequest:
		return "refs/heads/" + m.BaseRefName, nil
	case *bitbucketserver.PullRequest:
		return m.ToRef.ID, nil
	case *gitlab.MergeRequest:
		return "refs/heads/" + m.TargetBranch, nil
	default:
		return "", errors.New("unknown changeset type")
	}
}

// SupportsLabels returns whether the code host on which the changeset is
// hosted supports labels and whether it's safe to call the
// (*Changeset).Labels() method.
func (c *Changeset) SupportsLabels() bool {
	return ExternalServiceSupports(c.ExternalServiceType, CodehostCapabilityLabels)
}

// SupportsDraft returns whether the code host on which the changeset is
// hosted supports draft changesets.
func (c *Changeset) SupportsDraft() bool {
	return ExternalServiceSupports(c.ExternalServiceType, CodehostCapabilityDraftChangesets)
}

func (c *Changeset) Labels() []ChangesetLabel {
	switch m := c.Metadata.(type) {
	case *github.PullRequest:
		labels := make([]ChangesetLabel, len(m.Labels.Nodes))
		for i, l := range m.Labels.Nodes {
			labels[i] = ChangesetLabel{
				Name:        l.Name,
				Color:       l.Color,
				Description: l.Description,
			}
		}
		return labels
	case *gitlab.MergeRequest:
		// Similarly to GitHub above, GitLab labels can have colors (foreground
		// _and_ background, in fact) and descriptions. Unfortunately, the REST
		// API only returns this level of detail on the list endpoint (with an
		// option added in GitLab 12.7), and not when retrieving individual MRs.
		//
		// When our minimum GitLab version is 12.0, we should be able to switch
		// to retrieving MRs via GraphQL, and then we can start retrieving
		// richer label data.
		labels := make([]ChangesetLabel, len(m.Labels))
		for i, l := range m.Labels {
			labels[i] = ChangesetLabel{Name: l, Color: "000000"}
		}
		return labels
	default:
		return []ChangesetLabel{}
	}
}

// ResetQueued resets the failure message and reset count and sets the changesets ReconcilerState to queued.
func (c *Changeset) ResetQueued() {
	c.ReconcilerState = ReconcilerStateQueued
	c.NumResets = 0
	c.NumFailures = 0
	c.FailureMessage = nil
}

// Changesets is a slice of *Changesets.
type Changesets []*Changeset

// IDs returns the IDs of all changesets in the slice.
func (cs Changesets) IDs() []int64 {
	ids := make([]int64, len(cs))
	for i, c := range cs {
		ids[i] = c.ID
	}
	return ids
}

// IDs returns the unique RepoIDs of all changesets in the slice.
func (cs Changesets) RepoIDs() []api.RepoID {
	repoIDMap := make(map[api.RepoID]struct{})
	for _, c := range cs {
		repoIDMap[c.RepoID] = struct{}{}
	}
	repoIDs := make([]api.RepoID, len(repoIDMap))
	for id := range repoIDMap {
		repoIDs = append(repoIDs, id)
	}
	return repoIDs
}

// Filter returns a new Changesets slice in which changesets have been filtered
// out for which the predicate didn't return true.
func (cs Changesets) Filter(predicate func(*Changeset) bool) (filtered Changesets) {
	for _, c := range cs {
		if predicate(c) {
			filtered = append(filtered, c)
		}
	}

	return filtered
}

// Find returns the first changeset in the slice for which the predicate
// returned true.
func (cs Changesets) Find(predicate func(*Changeset) bool) *Changeset {
	for _, c := range cs {
		if predicate(c) {
			return c
		}
	}

	return nil
}

// WithCurrentSpecID returns a predicate function that can be passed to
// Changesets.Filter/Find, etc.
func WithCurrentSpecID(id int64) func(*Changeset) bool {
	return func(c *Changeset) bool { return c.CurrentSpecID == id }
}

// WithExternalID returns a predicate function that can be passed to
// Changesets.Filter/Find, etc.
func WithExternalID(id string) func(*Changeset) bool {
	return func(c *Changeset) bool { return c.ExternalID == id }
}

// ChangesetEventKindFor returns the ChangesetEventKind for the given
// specific code host event.
func ChangesetEventKindFor(e interface{}) ChangesetEventKind {
	switch e := e.(type) {
	case *github.AssignedEvent:
		return ChangesetEventKindGitHubAssigned
	case *github.ClosedEvent:
		return ChangesetEventKindGitHubClosed
	case *github.IssueComment:
		return ChangesetEventKindGitHubCommented
	case *github.RenamedTitleEvent:
		return ChangesetEventKindGitHubRenamedTitle
	case *github.MergedEvent:
		return ChangesetEventKindGitHubMerged
	case *github.PullRequestReview:
		return ChangesetEventKindGitHubReviewed
	case *github.PullRequestReviewComment:
		return ChangesetEventKindGitHubReviewCommented
	case *github.ReopenedEvent:
		return ChangesetEventKindGitHubReopened
	case *github.ReviewDismissedEvent:
		return ChangesetEventKindGitHubReviewDismissed
	case *github.ReviewRequestRemovedEvent:
		return ChangesetEventKindGitHubReviewRequestRemoved
	case *github.ReviewRequestedEvent:
		return ChangesetEventKindGitHubReviewRequested
	case *github.ReadyForReviewEvent:
		return ChangesetEventKindGitHubReadyForReview
	case *github.ConvertToDraftEvent:
		return ChangesetEventKindGitHubConvertToDraft
	case *github.UnassignedEvent:
		return ChangesetEventKindGitHubUnassigned
	case *github.PullRequestCommit:
		return ChangesetEventKindGitHubCommit
	case *github.LabelEvent:
		if e.Removed {
			return ChangesetEventKindGitHubUnlabeled
		}
		return ChangesetEventKindGitHubLabeled
	case *github.CommitStatus:
		return ChangesetEventKindCommitStatus
	case *github.CheckSuite:
		return ChangesetEventKindCheckSuite
	case *github.CheckRun:
		return ChangesetEventKindCheckRun
	case *bitbucketserver.Activity:
		return ChangesetEventKind("bitbucketserver:" + strings.ToLower(string(e.Action)))
	case *bitbucketserver.ParticipantStatusEvent:
		return ChangesetEventKind("bitbucketserver:participant_status:" + strings.ToLower(string(e.Action)))
	case *bitbucketserver.CommitStatus:
		return ChangesetEventKindBitbucketServerCommitStatus
	case *gitlab.Pipeline:
		return ChangesetEventKindGitLabPipeline
	case *gitlab.ReviewApprovedEvent:
		return ChangesetEventKindGitLabApproved
	case *gitlab.ReviewUnapprovedEvent:
		return ChangesetEventKindGitLabUnapproved
	case *gitlab.MarkWorkInProgressEvent:
		return ChangesetEventKindGitLabMarkWorkInProgress
	case *gitlab.UnmarkWorkInProgressEvent:
		return ChangesetEventKindGitLabUnmarkWorkInProgress
	case *gitlabwebhooks.MergeRequestCloseEvent:
		return ChangesetEventKindGitLabClosed
	case *gitlabwebhooks.MergeRequestMergeEvent:
		return ChangesetEventKindGitLabMerged
	case *gitlabwebhooks.MergeRequestReopenEvent:
		return ChangesetEventKindGitLabReopened
	default:
		panic(errors.Errorf("unknown changeset event kind for %T", e))
	}
}

// NewChangesetEventMetadata returns a new metadata object for the given
// ChangesetEventKind.
func NewChangesetEventMetadata(k ChangesetEventKind) (interface{}, error) {
	switch {
	case strings.HasPrefix(string(k), "bitbucketserver"):
		switch k {
		case ChangesetEventKindBitbucketServerCommitStatus:
			return new(bitbucketserver.CommitStatus), nil
		case ChangesetEventKindBitbucketServerDismissed:
			return new(bitbucketserver.ParticipantStatusEvent), nil
		default:
			return new(bitbucketserver.Activity), nil
		}
	case strings.HasPrefix(string(k), "github"):
		switch k {
		case ChangesetEventKindGitHubAssigned:
			return new(github.AssignedEvent), nil
		case ChangesetEventKindGitHubClosed:
			return new(github.ClosedEvent), nil
		case ChangesetEventKindGitHubCommented:
			return new(github.IssueComment), nil
		case ChangesetEventKindGitHubRenamedTitle:
			return new(github.RenamedTitleEvent), nil
		case ChangesetEventKindGitHubMerged:
			return new(github.MergedEvent), nil
		case ChangesetEventKindGitHubReviewed:
			return new(github.PullRequestReview), nil
		case ChangesetEventKindGitHubReviewCommented:
			return new(github.PullRequestReviewComment), nil
		case ChangesetEventKindGitHubReopened:
			return new(github.ReopenedEvent), nil
		case ChangesetEventKindGitHubReviewDismissed:
			return new(github.ReviewDismissedEvent), nil
		case ChangesetEventKindGitHubReviewRequestRemoved:
			return new(github.ReviewRequestRemovedEvent), nil
		case ChangesetEventKindGitHubReviewRequested:
			return new(github.ReviewRequestedEvent), nil
		case ChangesetEventKindGitHubReadyForReview:
			return new(github.ReadyForReviewEvent), nil
		case ChangesetEventKindGitHubConvertToDraft:
			return new(github.ConvertToDraftEvent), nil
		case ChangesetEventKindGitHubUnassigned:
			return new(github.UnassignedEvent), nil
		case ChangesetEventKindGitHubCommit:
			return new(github.PullRequestCommit), nil
		case ChangesetEventKindGitHubLabeled:
			return new(github.LabelEvent), nil
		case ChangesetEventKindGitHubUnlabeled:
			return &github.LabelEvent{Removed: true}, nil
		case ChangesetEventKindCommitStatus:
			return new(github.CommitStatus), nil
		case ChangesetEventKindCheckSuite:
			return new(github.CheckSuite), nil
		case ChangesetEventKindCheckRun:
			return new(github.CheckRun), nil
		}
	case strings.HasPrefix(string(k), "gitlab"):
		switch k {
		case ChangesetEventKindGitLabApproved:
			return new(gitlab.ReviewApprovedEvent), nil
		case ChangesetEventKindGitLabPipeline:
			return new(gitlab.Pipeline), nil
		case ChangesetEventKindGitLabUnapproved:
			return new(gitlab.ReviewUnapprovedEvent), nil
		case ChangesetEventKindGitLabMarkWorkInProgress:
			return new(gitlab.MarkWorkInProgressEvent), nil
		case ChangesetEventKindGitLabUnmarkWorkInProgress:
			return new(gitlab.UnmarkWorkInProgressEvent), nil
		case ChangesetEventKindGitLabClosed:
			return new(gitlabwebhooks.MergeRequestCloseEvent), nil
		case ChangesetEventKindGitLabMerged:
			return new(gitlabwebhooks.MergeRequestMergeEvent), nil
		case ChangesetEventKindGitLabReopened:
			return new(gitlabwebhooks.MergeRequestReopenEvent), nil
		}
	}
	return nil, errors.Errorf("unknown changeset event kind %q", k)
}
