package testing

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/go-diff/diff"

	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

type TestChangesetOpts struct {
	Repo         api.RepoID
	BatchChange  int64
	CurrentSpec  int64
	PreviousSpec int64
	BatchChanges []btypes.BatchChangeAssoc

	ExternalServiceType string
	ExternalID          string
	ExternalBranch      string
	ExternalState       btypes.ChangesetExternalState
	ExternalReviewState btypes.ChangesetReviewState
	ExternalCheckState  btypes.ChangesetCheckState

	DiffStatAdded   int32
	DiffStatChanged int32
	DiffStatDeleted int32

	PublicationState   btypes.ChangesetPublicationState
	UiPublicationState *btypes.ChangesetUiPublicationState

	ReconcilerState btypes.ReconcilerState
	FailureMessage  string
	NumFailures     int64
	NumResets       int64

	SyncErrorMessage string

	OwnedByBatchChange int64

	Closing    bool
	IsArchived bool
	Archive    bool

	Metadata interface{}
}

type CreateChangeseter interface {
	CreateChangeset(ctx context.Context, changeset *btypes.Changeset) error
}

func CreateChangeset(
	t *testing.T,
	ctx context.Context,
	store CreateChangeseter,
	opts TestChangesetOpts,
) *btypes.Changeset {
	t.Helper()

	changeset := BuildChangeset(opts)

	if err := store.CreateChangeset(ctx, changeset); err != nil {
		t.Fatalf("creating changeset failed: %s", err)
	}

	return changeset
}

func BuildChangeset(opts TestChangesetOpts) *btypes.Changeset {
	if opts.ExternalServiceType == "" {
		opts.ExternalServiceType = extsvc.TypeGitHub
	}

	changeset := &btypes.Changeset{
		RepoID:         opts.Repo,
		CurrentSpecID:  opts.CurrentSpec,
		PreviousSpecID: opts.PreviousSpec,
		BatchChanges:   opts.BatchChanges,

		ExternalServiceType: opts.ExternalServiceType,
		ExternalID:          opts.ExternalID,
		ExternalState:       opts.ExternalState,
		ExternalReviewState: opts.ExternalReviewState,
		ExternalCheckState:  opts.ExternalCheckState,

		PublicationState:   opts.PublicationState,
		UiPublicationState: opts.UiPublicationState,

		OwnedByBatchChangeID: opts.OwnedByBatchChange,

		Closing: opts.Closing,

		ReconcilerState: opts.ReconcilerState,
		NumFailures:     opts.NumFailures,
		NumResets:       opts.NumResets,

		Metadata: opts.Metadata,
	}

	if opts.SyncErrorMessage != "" {
		changeset.SyncErrorMessage = &opts.SyncErrorMessage
	}

	if opts.ExternalBranch != "" {
		changeset.ExternalBranch = git.EnsureRefPrefix(opts.ExternalBranch)
	}

	if opts.FailureMessage != "" {
		changeset.FailureMessage = &opts.FailureMessage
	}

	if opts.BatchChange != 0 {
		changeset.BatchChanges = []btypes.BatchChangeAssoc{
			{BatchChangeID: opts.BatchChange, IsArchived: opts.IsArchived, Archive: opts.Archive},
		}
	}

	if opts.DiffStatAdded > 0 || opts.DiffStatChanged > 0 || opts.DiffStatDeleted > 0 {
		changeset.DiffStatAdded = &opts.DiffStatAdded
		changeset.DiffStatChanged = &opts.DiffStatChanged
		changeset.DiffStatDeleted = &opts.DiffStatDeleted
	}

	return changeset
}

type ChangesetAssertions struct {
	Repo               api.RepoID
	CurrentSpec        int64
	PreviousSpec       int64
	OwnedByBatchChange int64
	ReconcilerState    btypes.ReconcilerState
	PublicationState   btypes.ChangesetPublicationState
	UiPublicationState *btypes.ChangesetUiPublicationState
	ExternalState      btypes.ChangesetExternalState
	ExternalID         string
	ExternalBranch     string
	DiffStat           *diff.Stat
	Closing            bool

	Title string
	Body  string

	FailureMessage   *string
	SyncErrorMessage *string
	NumFailures      int64
	NumResets        int64

	AttachedTo []int64
	DetachFrom []int64

	ArchiveIn                  int64
	ArchivedInOwnerBatchChange bool
}

func AssertChangeset(t *testing.T, c *btypes.Changeset, a ChangesetAssertions) {
	t.Helper()

	if c == nil {
		t.Fatalf("changeset is nil")
	}

	if have, want := c.RepoID, a.Repo; have != want {
		t.Fatalf("changeset RepoID wrong. want=%d, have=%d", want, have)
	}

	if have, want := c.CurrentSpecID, a.CurrentSpec; have != want {
		t.Fatalf("changeset CurrentSpecID wrong. want=%d, have=%d", want, have)
	}

	if have, want := c.PreviousSpecID, a.PreviousSpec; have != want {
		t.Fatalf("changeset PreviousSpecID wrong. want=%d, have=%d", want, have)
	}

	if have, want := c.OwnedByBatchChangeID, a.OwnedByBatchChange; have != want {
		t.Fatalf("changeset OwnedByBatchChangeID wrong. want=%d, have=%d", want, have)
	}

	if have, want := c.ReconcilerState, a.ReconcilerState; have != want {
		t.Fatalf("changeset ReconcilerState wrong. want=%s, have=%s", want, have)
	}

	if have, want := c.PublicationState, a.PublicationState; have != want {
		t.Fatalf("changeset PublicationState wrong. want=%s, have=%s", want, have)
	}

	if diff := cmp.Diff(c.UiPublicationState, a.UiPublicationState); diff != "" {
		t.Fatalf("changeset UiPublicationState wrong. (-have +want):\n%s", diff)
	}

	if have, want := c.ExternalState, a.ExternalState; have != want {
		t.Fatalf("changeset ExternalState wrong. want=%s, have=%s", want, have)
	}

	if have, want := c.ExternalID, a.ExternalID; have != want {
		t.Fatalf("changeset ExternalID wrong. want=%s, have=%s", want, have)
	}

	if have, want := c.ExternalBranch, a.ExternalBranch; have != want {
		t.Fatalf("changeset ExternalBranch wrong. want=%s, have=%s", want, have)
	}

	if want, have := a.FailureMessage, c.FailureMessage; want == nil && have != nil {
		t.Fatalf("expected no failure message, but have=%q", *have)
	}

	if diff := cmp.Diff(a.DiffStat, c.DiffStat()); diff != "" {
		t.Fatalf("changeset DiffStat wrong. (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(a.Closing, c.Closing); diff != "" {
		t.Fatalf("changeset Closing wrong. (-want +got):\n%s", diff)
	}

	toDetach := []int64{}
	for _, assoc := range c.BatchChanges {
		if assoc.Detach {
			toDetach = append(toDetach, assoc.BatchChangeID)
		}
	}
	if a.DetachFrom == nil {
		a.DetachFrom = []int64{}
	}
	sort.Slice(toDetach, func(i, j int) bool { return toDetach[i] < toDetach[j] })
	sort.Slice(a.DetachFrom, func(i, j int) bool { return a.DetachFrom[i] < a.DetachFrom[j] })
	if diff := cmp.Diff(a.DetachFrom, toDetach); diff != "" {
		t.Fatalf("changeset DetachFrom wrong. (-want +got):\n%s", diff)
	}

	attachedTo := []int64{}
	for _, assoc := range c.BatchChanges {
		if !assoc.Detach {
			attachedTo = append(attachedTo, assoc.BatchChangeID)
		}
	}
	if a.AttachedTo == nil {
		a.AttachedTo = []int64{}
	}
	sort.Slice(attachedTo, func(i, j int) bool { return attachedTo[i] < attachedTo[j] })
	sort.Slice(a.AttachedTo, func(i, j int) bool { return a.AttachedTo[i] < a.AttachedTo[j] })
	if diff := cmp.Diff(a.AttachedTo, attachedTo); diff != "" {
		t.Fatalf("changeset AttachedTo wrong. (-want +got):\n%s", diff)
	}

	if a.ArchiveIn != 0 {
		found := false
		for _, assoc := range c.BatchChanges {
			if assoc.BatchChangeID == a.ArchiveIn {
				found = true
				if !assoc.Archive {
					t.Fatalf("changeset association to %d not set to Archive", a.ArchiveIn)
				}
			}
		}
		if !found {
			t.Fatalf("no changeset batchChange association set to archive")
		}
	}

	if a.ArchivedInOwnerBatchChange {
		found := false
		for _, assoc := range c.BatchChanges {
			if assoc.BatchChangeID == c.OwnedByBatchChangeID {
				found = true
				if !assoc.IsArchived {
					t.Fatalf("changeset association to %d not set to Archived", c.OwnedByBatchChangeID)
				}

				if assoc.Archive {
					t.Fatalf("changeset association to %d set to Archive, but should be Archived already", c.OwnedByBatchChangeID)
				}
			}
		}
		if !found {
			t.Fatalf("no changeset batchChange association archived")
		}
	}

	if want := c.FailureMessage; want != nil {
		if c.FailureMessage == nil {
			t.Fatalf("expected failure message %q but have none", *want)
		}
		if want, have := *a.FailureMessage, *c.FailureMessage; have != want {
			t.Fatalf("wrong failure message. want=%q, have=%q", want, have)
		}
	}

	if want := c.SyncErrorMessage; want != nil {
		if c.SyncErrorMessage == nil {
			t.Fatalf("expected sync error message %q but have none", *want)
		}
		if want, have := *a.SyncErrorMessage, *c.SyncErrorMessage; have != want {
			t.Fatalf("wrong sync error message. want=%q, have=%q", want, have)
		}
	}

	if have, want := c.NumFailures, a.NumFailures; have != want {
		t.Fatalf("changeset NumFailures wrong. want=%d, have=%d", want, have)
	}

	if have, want := c.NumResets, a.NumResets; have != want {
		t.Fatalf("changeset NumResets wrong. want=%d, have=%d", want, have)
	}

	if have, want := c.ExternalBranch, a.ExternalBranch; have != want {
		t.Fatalf("changeset ExternalBranch wrong. want=%s, have=%s", want, have)
	}

	if want := a.Title; want != "" {
		have, err := c.Title()
		if err != nil {
			t.Fatalf("changeset.Title failed: %s", err)
		}

		if have != want {
			t.Fatalf("changeset Title wrong. want=%s, have=%s", want, have)
		}
	}

	if want := a.Body; want != "" {
		have, err := c.Body()
		if err != nil {
			t.Fatalf("changeset.Body failed: %s", err)
		}

		if have != want {
			t.Fatalf("changeset Body wrong. want=%s, have=%s", want, have)
		}
	}
}

type GetChangesetByIDer interface {
	GetChangesetByID(ctx context.Context, id int64) (*btypes.Changeset, error)
}

func ReloadAndAssertChangeset(t *testing.T, ctx context.Context, s GetChangesetByIDer, c *btypes.Changeset, a ChangesetAssertions) (reloaded *btypes.Changeset) {
	t.Helper()

	reloaded, err := s.GetChangesetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("reloading changeset %d failed: %s", c.ID, err)
	}

	AssertChangeset(t, reloaded, a)

	return reloaded
}

type UpdateChangeseter interface {
	UpdateChangeset(ctx context.Context, changeset *btypes.Changeset) error
}

func SetChangesetPublished(t *testing.T, ctx context.Context, s UpdateChangeseter, c *btypes.Changeset, externalID, externalBranch string) {
	t.Helper()

	c.ExternalBranch = externalBranch
	c.ExternalID = externalID
	c.PublicationState = btypes.ChangesetPublicationStatePublished
	c.ReconcilerState = btypes.ReconcilerStateCompleted
	c.ExternalState = btypes.ChangesetExternalStateOpen

	if err := s.UpdateChangeset(ctx, c); err != nil {
		t.Fatalf("failed to update changeset: %s", err)
	}
}

var FailedChangesetFailureMessage = "Failed test"

func SetChangesetFailed(t *testing.T, ctx context.Context, s UpdateChangeseter, c *btypes.Changeset) {
	t.Helper()

	c.ReconcilerState = btypes.ReconcilerStateFailed
	c.FailureMessage = &FailedChangesetFailureMessage
	c.NumFailures = 5

	if err := s.UpdateChangeset(ctx, c); err != nil {
		t.Fatalf("failed to update changeset: %s", err)
	}
}

func SetChangesetClosed(t *testing.T, ctx context.Context, s UpdateChangeseter, c *btypes.Changeset) {
	t.Helper()

	c.PublicationState = btypes.ChangesetPublicationStatePublished
	c.ReconcilerState = btypes.ReconcilerStateCompleted
	c.Closing = false
	c.ExternalState = btypes.ChangesetExternalStateClosed

	assocs := make([]btypes.BatchChangeAssoc, 0)
	for _, assoc := range c.BatchChanges {
		if !assoc.Detach {
			if assoc.Archive {
				assoc.IsArchived = true
				assoc.Archive = false
			}
			assocs = append(assocs, assoc)
		}
	}

	c.BatchChanges = assocs

	if err := s.UpdateChangeset(ctx, c); err != nil {
		t.Fatalf("failed to update changeset: %s", err)
	}
}
