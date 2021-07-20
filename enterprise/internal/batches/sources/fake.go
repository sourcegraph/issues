package sources

import (
	"context"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

type fakeSourcer struct {
	err    error
	source ChangesetSource
}

func (s *fakeSourcer) ForChangeset(ctx context.Context, tx SourcerStore, ch *btypes.Changeset) (ChangesetSource, error) {
	return s.source, s.err
}

func (s *fakeSourcer) ForRepo(ctx context.Context, tx SourcerStore, repo *types.Repo) (ChangesetSource, error) {
	return s.source, s.err
}

func (s *fakeSourcer) ForExternalService(ctx context.Context, tx SourcerStore, opts store.GetExternalServiceIDsOpts) (ChangesetSource, error) {
	return s.source, s.err
}

// FakeChangesetSource is a fake implementation of the ChangesetSource
// interface to be used in tests.
type FakeChangesetSource struct {
	Svc *types.ExternalService

	CurrentAuthenticator auth.Authenticator

	CreateDraftChangesetCalled  bool
	UndraftedChangesetsCalled   bool
	CreateChangesetCalled       bool
	UpdateChangesetCalled       bool
	ListReposCalled             bool
	ExternalServicesCalled      bool
	LoadChangesetCalled         bool
	CloseChangesetCalled        bool
	ReopenChangesetCalled       bool
	CreateCommentCalled         bool
	AuthenticatedUsernameCalled bool
	ValidateAuthenticatorCalled bool
	MergeChangesetCalled        bool

	// The Changeset.HeadRef to be expected in CreateChangeset/UpdateChangeset calls.
	WantHeadRef string
	// The Changeset.BaseRef to be expected in CreateChangeset/UpdateChangeset calls.
	WantBaseRef string

	// The metadata the FakeChangesetSource should set on the created/updated
	// Changeset with changeset.SetMetadata.
	FakeMetadata interface{}

	// Whether or not the changeset already ChangesetExists on the code host at the time
	// when CreateChangeset is called.
	ChangesetExists bool

	// When true, ValidateAuthenticator will return no error.
	AuthenticatorIsValid bool

	// error to be returned from every method
	Err error

	// ClosedChangesets contains the changesets that were passed to CloseChangeset
	ClosedChangesets []*Changeset

	// CreatedChangesets contains the changesets that were passed to
	// CreateChangeset
	CreatedChangesets []*Changeset

	// LoadedChangesets contains the changesets that were passed to LoadChangeset
	LoadedChangesets []*Changeset

	// UpdateChangesets contains the changesets that were passed to
	// UpdateChangeset
	UpdatedChangesets []*Changeset

	// ReopenedChangesets contains the changesets that were passed to ReopenedChangeset
	ReopenedChangesets []*Changeset

	// UndraftedChangesets contains the changesets that were passed to UndraftChangeset
	UndraftedChangesets []*Changeset

	// Username is the username returned by AuthenticatedUsername
	Username string
}

var _ ChangesetSource = &FakeChangesetSource{}
var _ DraftChangesetSource = &FakeChangesetSource{}

func (s *FakeChangesetSource) CreateDraftChangeset(ctx context.Context, c *Changeset) (bool, error) {
	s.CreateDraftChangesetCalled = true

	if s.Err != nil {
		return s.ChangesetExists, s.Err
	}

	if c.Repo == nil {
		return false, NoReposErr
	}

	if c.HeadRef != s.WantHeadRef {
		return s.ChangesetExists, errors.Errorf("wrong HeadRef. want=%s, have=%s", s.WantHeadRef, c.HeadRef)
	}

	if c.BaseRef != s.WantBaseRef {
		return s.ChangesetExists, errors.Errorf("wrong BaseRef. want=%s, have=%s", s.WantBaseRef, c.BaseRef)
	}

	if err := c.SetMetadata(s.FakeMetadata); err != nil {
		return s.ChangesetExists, err
	}

	s.CreatedChangesets = append(s.CreatedChangesets, c)
	return s.ChangesetExists, s.Err
}

func (s *FakeChangesetSource) UndraftChangeset(ctx context.Context, c *Changeset) error {
	s.UndraftedChangesetsCalled = true

	if s.Err != nil {
		return s.Err
	}

	if c.Repo == nil {
		return NoReposErr
	}

	s.UndraftedChangesets = append(s.UndraftedChangesets, c)

	return c.SetMetadata(s.FakeMetadata)
}

func (s *FakeChangesetSource) CreateChangeset(ctx context.Context, c *Changeset) (bool, error) {
	s.CreateChangesetCalled = true

	if s.Err != nil {
		return s.ChangesetExists, s.Err
	}

	if c.Repo == nil {
		return false, NoReposErr
	}

	if c.HeadRef != s.WantHeadRef {
		return s.ChangesetExists, errors.Errorf("wrong HeadRef. want=%s, have=%s", s.WantHeadRef, c.HeadRef)
	}

	if c.BaseRef != s.WantBaseRef {
		return s.ChangesetExists, errors.Errorf("wrong BaseRef. want=%s, have=%s", s.WantBaseRef, c.BaseRef)
	}

	if err := c.SetMetadata(s.FakeMetadata); err != nil {
		return s.ChangesetExists, err
	}

	s.CreatedChangesets = append(s.CreatedChangesets, c)
	return s.ChangesetExists, s.Err
}

func (s *FakeChangesetSource) UpdateChangeset(ctx context.Context, c *Changeset) error {
	s.UpdateChangesetCalled = true

	if s.Err != nil {
		return s.Err
	}
	if c.Repo == nil {
		return NoReposErr
	}

	if c.BaseRef != s.WantBaseRef {
		return errors.Errorf("wrong BaseRef. want=%s, have=%s", s.WantBaseRef, c.BaseRef)
	}

	s.UpdatedChangesets = append(s.UpdatedChangesets, c)
	return c.SetMetadata(s.FakeMetadata)
}

var fakeNotImplemented = errors.New("not implemented in FakeChangesetSource")

func (s *FakeChangesetSource) ListRepos(ctx context.Context, results chan repos.SourceResult) {
	s.ListReposCalled = true

	results <- repos.SourceResult{Source: s, Err: fakeNotImplemented}
}

func (s *FakeChangesetSource) ExternalServices() types.ExternalServices {
	s.ExternalServicesCalled = true

	return types.ExternalServices{s.Svc}
}
func (s *FakeChangesetSource) LoadChangeset(ctx context.Context, c *Changeset) error {
	s.LoadChangesetCalled = true

	if s.Err != nil {
		return s.Err
	}

	if c.Repo == nil {
		return NoReposErr
	}

	if err := c.SetMetadata(s.FakeMetadata); err != nil {
		return err
	}

	s.LoadedChangesets = append(s.LoadedChangesets, c)
	return nil
}

var NoReposErr = errors.New("no repository set on Changeset")

func (s *FakeChangesetSource) CloseChangeset(ctx context.Context, c *Changeset) error {
	s.CloseChangesetCalled = true

	if s.Err != nil {
		return s.Err
	}

	if c.Repo == nil {
		return NoReposErr
	}

	s.ClosedChangesets = append(s.ClosedChangesets, c)

	return c.SetMetadata(s.FakeMetadata)
}

func (s *FakeChangesetSource) ReopenChangeset(ctx context.Context, c *Changeset) error {
	s.ReopenChangesetCalled = true

	if s.Err != nil {
		return s.Err
	}

	if c.Repo == nil {
		return NoReposErr
	}

	s.ReopenedChangesets = append(s.ReopenedChangesets, c)

	return c.SetMetadata(s.FakeMetadata)
}

func (s *FakeChangesetSource) CreateComment(ctx context.Context, c *Changeset, body string) error {
	s.CreateCommentCalled = true
	return s.Err
}

func (s *FakeChangesetSource) GitserverPushConfig(ctx context.Context, store *database.ExternalServiceStore, repo *types.Repo) (*protocol.PushConfig, error) {
	return gitserverPushConfig(ctx, store, repo, s.CurrentAuthenticator)
}

func (s *FakeChangesetSource) WithAuthenticator(a auth.Authenticator) (ChangesetSource, error) {
	s.CurrentAuthenticator = a
	return s, nil
}

func (s *FakeChangesetSource) ValidateAuthenticator(context.Context) error {
	s.ValidateAuthenticatorCalled = true
	if s.AuthenticatorIsValid {
		return nil
	}
	return errors.New("invalid authenticator in fake source")
}

func (s *FakeChangesetSource) AuthenticatedUsername(ctx context.Context) (string, error) {
	s.AuthenticatedUsernameCalled = true
	return s.Username, nil
}

func (s *FakeChangesetSource) MergeChangeset(ctx context.Context, c *Changeset, squash bool) error {
	s.MergeChangesetCalled = true
	return s.Err
}
