package sources

import (
	"context"
	"fmt"

	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

// ChangesetNotFoundError is returned by LoadChangeset if the changeset
// could not be found on the codehost. This is only returned, if the
// changeset is actually not found. Other not found errors, such as
// repo not found should NOT raise this error, since it will cause
// the changeset to be marked as deleted.
type ChangesetNotFoundError struct {
	Changeset *Changeset
}

func (e ChangesetNotFoundError) Error() string {
	return fmt.Sprintf("Changeset with external ID %s not found", e.Changeset.Changeset.ExternalID)
}

func (e ChangesetNotFoundError) NonRetryable() bool { return true }

// A DraftChangesetSource can create draft changesets and undraft them.
type DraftChangesetSource interface {
	// CreateDraftChangeset will create the Changeset on the source. If it already
	// exists, *Changeset will be populated and the return value will be
	// true.
	CreateDraftChangeset(context.Context, *Changeset) (bool, error)
	// UndraftChangeset will update the Changeset on the source to be not in draft mode anymore.
	UndraftChangeset(context.Context, *Changeset) error
	// RedraftChangest will update the Changeset on the source to be in draft mode again.
	RedraftChangeset(context.Context, *Changeset) error
}

// A ChangesetSource can load the latest state of a list of Changesets.
type ChangesetSource interface {
	// GitserverPushConfig returns an authenticated push config used for pushing
	// commits to the code host.
	GitserverPushConfig(context.Context, *database.ExternalServiceStore, *types.Repo) (*protocol.PushConfig, error)
	// WithAuthenticator returns a copy of the original Source configured to use
	// the given authenticator, provided that authenticator type is supported by
	// the code host.
	WithAuthenticator(auth.Authenticator) (ChangesetSource, error)
	// ValidateAuthenticator validates the currently set authenticator is usable.
	// Returns an error, when validating the Authenticator yielded an error.
	ValidateAuthenticator(ctx context.Context) error

	// LoadChangeset loads the given Changeset from the source and updates it.
	// If the Changeset could not be found on the source, a ChangesetNotFoundError is returned.
	LoadChangeset(context.Context, *Changeset) error
	// CreateChangeset will create the Changeset on the source. If it already
	// exists, *Changeset will be populated and the return value will be
	// true.
	CreateChangeset(context.Context, *Changeset) (bool, error)
	// CloseChangeset will close the Changeset on the source, where "close"
	// means the appropriate final state on the codehost (e.g. "declined" on
	// Bitbucket Server).
	CloseChangeset(context.Context, *Changeset) error
	// UpdateChangeset can update Changesets.
	UpdateChangeset(context.Context, *Changeset) error
	// ReopenChangeset will reopen the Changeset on the source, if it's closed.
	// If not, it's a noop.
	ReopenChangeset(context.Context, *Changeset) error
	// CreateComment posts a comment on the Changeset.
	CreateComment(context.Context, *Changeset, string) error
}

// A Changeset of an existing Repo.
type Changeset struct {
	Title   string
	Body    string
	HeadRef string
	BaseRef string

	*btypes.Changeset
	*types.Repo
}

// IsOutdated returns true when the attributes of the nested
// batches.Changeset do not match the attributes (title, body, ...) set on
// the Changeset.
func (c *Changeset) IsOutdated() (bool, error) {
	currentTitle, err := c.Changeset.Title()
	if err != nil {
		return false, err
	}

	if currentTitle != c.Title {
		return true, nil
	}

	currentBody, err := c.Changeset.Body()
	if err != nil {
		return false, err
	}

	if currentBody != c.Body {
		return true, nil
	}

	currentBaseRef, err := c.Changeset.BaseRef()
	if err != nil {
		return false, err
	}

	if git.EnsureRefPrefix(currentBaseRef) != git.EnsureRefPrefix(c.BaseRef) {
		return true, nil
	}

	return false, nil
}
