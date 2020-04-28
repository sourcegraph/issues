package campaigns

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
)

// FakeChangesetSource is a fake implementation of the repos.ChangesetSource
// interface to be used in tests.
type FakeChangesetSource struct {
	Svc *repos.ExternalService

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

	// error to be returned from every method
	Err error
}

func (s FakeChangesetSource) CreateChangeset(ctx context.Context, c *repos.Changeset) (bool, error) {
	if s.Err != nil {
		return s.ChangesetExists, s.Err
	}

	if c.HeadRef != s.WantHeadRef {
		return s.ChangesetExists, fmt.Errorf("wrong HeadRef. want=%s, have=%s", s.WantHeadRef, c.HeadRef)
	}

	if c.BaseRef != s.WantBaseRef {
		return s.ChangesetExists, fmt.Errorf("wrong BaseRef. want=%s, have=%s", s.WantBaseRef, c.BaseRef)
	}

	c.SetMetadata(s.FakeMetadata)

	return s.ChangesetExists, s.Err
}

func (s FakeChangesetSource) UpdateChangeset(ctx context.Context, c *repos.Changeset) error {
	if s.Err != nil {
		return s.Err
	}

	if c.BaseRef != s.WantBaseRef {
		return fmt.Errorf("wrong BaseRef. want=%s, have=%s", s.WantBaseRef, c.BaseRef)
	}

	c.SetMetadata(s.FakeMetadata)
	return nil
}

var fakeNotImplemented = errors.New("not implement in FakeChangesetSource")

func (s FakeChangesetSource) ListRepos(ctx context.Context, results chan repos.SourceResult) {
	results <- repos.SourceResult{Source: s, Err: fakeNotImplemented}
}

func (s FakeChangesetSource) ExternalServices() repos.ExternalServices {
	return repos.ExternalServices{s.Svc}
}
func (s FakeChangesetSource) LoadChangesets(ctx context.Context, cs ...*repos.Changeset) error {
	return fakeNotImplemented
}
func (s FakeChangesetSource) CloseChangeset(ctx context.Context, c *repos.Changeset) error {
	return fakeNotImplemented
}
