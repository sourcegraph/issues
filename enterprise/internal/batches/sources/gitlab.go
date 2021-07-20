package sources

import (
	"context"
	"net/url"
	"strconv"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
)

type GitLabSource struct {
	client *gitlab.Client
	au     auth.Authenticator
}

// NewGitLabSource returns a new GitLabSource from the given external service.
func NewGitLabSource(svc *types.ExternalService, cf *httpcli.Factory) (*GitLabSource, error) {
	var c schema.GitLabConnection
	if err := jsonc.Unmarshal(svc.Config, &c); err != nil {
		return nil, errors.Errorf("external service id=%d config error: %s", svc.ID, err)
	}
	return newGitLabSource(&c, cf, nil)
}

func newGitLabSource(c *schema.GitLabConnection, cf *httpcli.Factory, au auth.Authenticator) (*GitLabSource, error) {
	baseURL, err := url.Parse(c.Url)
	if err != nil {
		return nil, err
	}
	baseURL = extsvc.NormalizeBaseURL(baseURL)

	if cf == nil {
		cf = httpcli.NewExternalHTTPClientFactory()
	}

	var opts []httpcli.Opt
	if c.Certificate != "" {
		opts = append(opts, httpcli.NewCertPoolOpt(c.Certificate))
	}

	cli, err := cf.Doer(opts...)
	if err != nil {
		return nil, err
	}

	// Don't modify passed-in parameter.
	var authr auth.Authenticator
	if au == nil && c.Token != "" {
		switch c.TokenType {
		case "oauth":
			authr = &auth.OAuthBearerToken{Token: c.Token}
		default:
			authr = &gitlab.SudoableToken{Token: c.Token}
		}
	} else {
		authr = au
	}

	provider := gitlab.NewClientProvider(baseURL, cli)
	return &GitLabSource{
		au:     authr,
		client: provider.GetAuthenticatorClient(authr),
	}, nil
}

func (s GitLabSource) GitserverPushConfig(ctx context.Context, store *database.ExternalServiceStore, repo *types.Repo) (*protocol.PushConfig, error) {
	return gitserverPushConfig(ctx, store, repo, s.au)
}

func (s GitLabSource) WithAuthenticator(a auth.Authenticator) (ChangesetSource, error) {
	switch a.(type) {
	case *auth.OAuthBearerToken,
		*auth.OAuthBearerTokenWithSSH:
		break

	default:
		return nil, newUnsupportedAuthenticatorError("GitLabSource", a)
	}

	sc := s
	sc.au = a
	sc.client = sc.client.WithAuthenticator(a)

	return &sc, nil
}

func (s GitLabSource) ValidateAuthenticator(ctx context.Context) error {
	return s.client.ValidateToken(ctx)
}

// CreateChangeset creates a GitLab merge request. If it already exists,
// *Changeset will be populated and the return value will be true.
func (s *GitLabSource) CreateChangeset(ctx context.Context, c *Changeset) (bool, error) {
	project := c.Repo.Metadata.(*gitlab.Project)
	exists := false
	source := git.AbbreviateRef(c.HeadRef)
	target := git.AbbreviateRef(c.BaseRef)

	mr, err := s.client.CreateMergeRequest(ctx, project, gitlab.CreateMergeRequestOpts{
		SourceBranch: source,
		TargetBranch: target,
		Title:        c.Title,
		Description:  c.Body,
	})
	if err != nil {
		if err == gitlab.ErrMergeRequestAlreadyExists {
			exists = true

			mr, err = s.client.GetOpenMergeRequestByRefs(ctx, project, source, target)
			if err != nil {
				return exists, errors.Wrap(err, "retrieving an extant merge request")
			}
		} else {
			return exists, errors.Wrap(err, "creating the merge request")
		}
	}

	// These additional API calls can go away once we can use the GraphQL API.
	if err := s.decorateMergeRequestData(ctx, project, mr); err != nil {
		return exists, errors.Wrapf(err, "retrieving additional data for merge request %d", mr.IID)
	}

	if err := c.SetMetadata(mr); err != nil {
		return exists, errors.Wrap(err, "setting changeset metadata")
	}
	return exists, nil
}

// CreateDraftChangeset creates a GitLab merge request. If it already exists,
// *Changeset will be populated and the return value will be true.
func (s *GitLabSource) CreateDraftChangeset(ctx context.Context, c *Changeset) (bool, error) {
	c.Title = gitlab.SetWIP(c.Title)

	exists, err := s.CreateChangeset(ctx, c)
	if err != nil {
		return exists, err
	}

	mr, ok := c.Changeset.Metadata.(*gitlab.MergeRequest)
	if !ok {
		return false, errors.New("Changeset is not a GitLab merge request")
	}

	// If it already exists, but is not a WIP, we need to update the title.
	if exists && !mr.WorkInProgress {
		if err := s.UpdateChangeset(ctx, c); err != nil {
			return exists, err
		}
	}
	return exists, nil
}

// CloseChangeset closes the merge request on GitLab, leaving it unlocked.
func (s *GitLabSource) CloseChangeset(ctx context.Context, c *Changeset) error {
	project := c.Repo.Metadata.(*gitlab.Project)
	mr, ok := c.Changeset.Metadata.(*gitlab.MergeRequest)
	if !ok {
		return errors.New("Changeset is not a GitLab merge request")
	}

	// Title and TargetBranch are required, even though we're not actually
	// changing them.
	updated, err := s.client.UpdateMergeRequest(ctx, project, mr, gitlab.UpdateMergeRequestOpts{
		Title:        mr.Title,
		TargetBranch: mr.TargetBranch,
		StateEvent:   gitlab.UpdateMergeRequestStateEventClose,
	})
	if err != nil {
		return errors.Wrap(err, "updating GitLab merge request")
	}

	// These additional API calls can go away once we can use the GraphQL API.
	if err := s.decorateMergeRequestData(ctx, project, mr); err != nil {
		return errors.Wrapf(err, "retrieving additional data for merge request %d", mr.IID)
	}

	if err := c.SetMetadata(updated); err != nil {
		return errors.Wrap(err, "setting changeset metadata")
	}
	return nil
}

// LoadChangeset loads the given merge request from GitLab and updates it.
func (s *GitLabSource) LoadChangeset(ctx context.Context, cs *Changeset) error {
	project := cs.Repo.Metadata.(*gitlab.Project)

	iid, err := strconv.ParseInt(cs.ExternalID, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "parsing changeset external ID %s", cs.ExternalID)
	}

	mr, err := s.client.GetMergeRequest(ctx, project, gitlab.ID(iid))
	if err != nil {
		if errors.Is(err, gitlab.ErrMergeRequestNotFound) {
			return ChangesetNotFoundError{Changeset: cs}
		}
		return errors.Wrapf(err, "retrieving merge request %d", iid)
	}

	// These additional API calls can go away once we can use the GraphQL API.
	if err := s.decorateMergeRequestData(ctx, project, mr); err != nil {
		return errors.Wrapf(err, "retrieving additional data for merge request %d", iid)
	}

	if err := cs.SetMetadata(mr); err != nil {
		return errors.Wrapf(err, "setting changeset metadata for merge request %d", iid)
	}

	return nil
}

// ReopenChangeset closes the merge request on GitLab, leaving it unlocked.
func (s *GitLabSource) ReopenChangeset(ctx context.Context, c *Changeset) error {
	project := c.Repo.Metadata.(*gitlab.Project)
	mr, ok := c.Changeset.Metadata.(*gitlab.MergeRequest)
	if !ok {
		return errors.New("Changeset is not a GitLab merge request")
	}

	// Title and TargetBranch are required, even though we're not actually
	// changing them.
	updated, err := s.client.UpdateMergeRequest(ctx, project, mr, gitlab.UpdateMergeRequestOpts{
		Title:        mr.Title,
		TargetBranch: mr.TargetBranch,
		StateEvent:   gitlab.UpdateMergeRequestStateEventReopen,
	})
	if err != nil {
		return errors.Wrap(err, "reopening GitLab merge request")
	}

	// These additional API calls can go away once we can use the GraphQL API.
	if err := s.decorateMergeRequestData(ctx, project, mr); err != nil {
		return errors.Wrapf(err, "retrieving additional data for merge request %d", mr.IID)
	}

	if err := c.SetMetadata(updated); err != nil {
		return errors.Wrap(err, "setting changeset metadata")
	}
	return nil
}

func (s *GitLabSource) decorateMergeRequestData(ctx context.Context, project *gitlab.Project, mr *gitlab.MergeRequest) error {
	notes, err := s.getMergeRequestNotes(ctx, project, mr)
	if err != nil {
		return errors.Wrap(err, "retrieving notes")
	}

	events, err := s.getMergeRequestResourceStateEvents(ctx, project, mr)
	if err != nil {
		return errors.Wrap(err, "retrieving resource state events")
	}

	pipelines, err := s.getMergeRequestPipelines(ctx, project, mr)
	if err != nil {
		return errors.Wrap(err, "retrieving pipelines")
	}

	mr.Notes = notes
	mr.Pipelines = pipelines
	mr.ResourceStateEvents = events
	return nil
}

// getMergeRequestNotes retrieves the notes attached to a merge request in
// descending time order.
func (s *GitLabSource) getMergeRequestNotes(ctx context.Context, project *gitlab.Project, mr *gitlab.MergeRequest) ([]*gitlab.Note, error) {
	// Get the forward iterator that gives us a note page at a time.
	it := s.client.GetMergeRequestNotes(ctx, project, mr.IID)

	// Now we can iterate over the pages of notes and fill in the slice to be
	// returned.
	notes, err := readSystemNotes(it)
	if err != nil {
		return nil, errors.Wrap(err, "reading note pages")
	}

	return notes, nil
}

func readSystemNotes(it func() ([]*gitlab.Note, error)) ([]*gitlab.Note, error) {
	var notes []*gitlab.Note

	for {
		page, err := it()
		if err != nil {
			return nil, errors.Wrap(err, "retrieving note page")
		}
		if len(page) == 0 {
			// The terminal condition for the iterator is returning an empty
			// slice with no error, so we can stop iterating here.
			return notes, nil
		}

		for _, note := range page {
			// We're only interested in system notes for batch changes, since they
			// include the review state changes we need; let's not even bother
			// storing the non-system ones.
			if note.System {
				notes = append(notes, note)
			}
		}
	}
}

// getMergeRequestResourceStateEvents retrieves the events attached to a merge request in
// descending time order.
func (s *GitLabSource) getMergeRequestResourceStateEvents(ctx context.Context, project *gitlab.Project, mr *gitlab.MergeRequest) ([]*gitlab.ResourceStateEvent, error) {
	// Get the forward iterator that gives us a note page at a time.
	it := s.client.GetMergeRequestResourceStateEvents(ctx, project, mr.IID)

	// Now we can iterate over the pages of notes and fill in the slice to be
	// returned.
	events, err := readMergeRequestResourceStateEvents(it)
	if err != nil {
		return nil, errors.Wrap(err, "reading resource state events pages")
	}

	return events, nil
}

func readMergeRequestResourceStateEvents(it func() ([]*gitlab.ResourceStateEvent, error)) ([]*gitlab.ResourceStateEvent, error) {
	var events []*gitlab.ResourceStateEvent

	for {
		page, err := it()
		if err != nil {
			return nil, errors.Wrap(err, "retrieving resource state events page")
		}
		if len(page) == 0 {
			// The terminal condition for the iterator is returning an empty
			// slice with no error, so we can stop iterating here.
			return events, nil
		}

		events = append(events, page...)
	}
}

// getMergeRequestPipelines retrieves the pipelines attached to a merge request
// in descending time order.
func (s *GitLabSource) getMergeRequestPipelines(ctx context.Context, project *gitlab.Project, mr *gitlab.MergeRequest) ([]*gitlab.Pipeline, error) {
	// Get the forward iterator that gives us a pipeline page at a time.
	it := s.client.GetMergeRequestPipelines(ctx, project, mr.IID)

	// Now we can iterate over the pages of pipelines and fill in the slice to
	// be returned.
	pipelines, err := readPipelines(it)
	if err != nil {
		return nil, errors.Wrap(err, "reading pipeline pages")
	}
	return pipelines, nil
}

func readPipelines(it func() ([]*gitlab.Pipeline, error)) ([]*gitlab.Pipeline, error) {
	var pipelines []*gitlab.Pipeline

	for {
		page, err := it()
		if err != nil {
			return nil, errors.Wrap(err, "retrieving pipeline page")
		}
		if len(page) == 0 {
			// The terminal condition for the iterator is returning an empty
			// slice with no error, so we can stop iterating here.
			return pipelines, nil
		}

		pipelines = append(pipelines, page...)
	}
}

// UpdateChangeset updates the merge request on GitLab to reflect the local
// state of the Changeset.
func (s *GitLabSource) UpdateChangeset(ctx context.Context, c *Changeset) error {
	mr, ok := c.Changeset.Metadata.(*gitlab.MergeRequest)
	if !ok {
		return errors.New("Changeset is not a GitLab merge request")
	}
	project := c.Repo.Metadata.(*gitlab.Project)

	updated, err := s.client.UpdateMergeRequest(ctx, project, mr, gitlab.UpdateMergeRequestOpts{
		Title:        c.Title,
		Description:  c.Body,
		TargetBranch: git.AbbreviateRef(c.BaseRef),
	})
	if err != nil {
		return errors.Wrap(err, "updating GitLab merge request")
	}

	// These additional API calls can go away once we can use the GraphQL API.
	if err := s.decorateMergeRequestData(ctx, project, mr); err != nil {
		return errors.Wrapf(err, "retrieving additional data for merge request %d", mr.IID)
	}

	return c.Changeset.SetMetadata(updated)
}

// UndraftChangeset marks the changeset as *not* work in progress anymore.
func (s *GitLabSource) UndraftChangeset(ctx context.Context, c *Changeset) error {
	c.Title = gitlab.UnsetWIP(c.Title)
	return s.UpdateChangeset(ctx, c)
}

// CreateComment posts a comment on the Changeset.
func (s *GitLabSource) CreateComment(ctx context.Context, c *Changeset, text string) error {
	project := c.Repo.Metadata.(*gitlab.Project)
	mr, ok := c.Changeset.Metadata.(*gitlab.MergeRequest)
	if !ok {
		return errors.New("Changeset is not a GitLab merge request")
	}

	return s.client.CreateMergeRequestNote(ctx, project, mr, text)
}

// MergeChangeset merges a Changeset on the code host, if in a mergeable state.
// If squash is true, a squash-then-merge merge will be performed.
func (s *GitLabSource) MergeChangeset(ctx context.Context, c *Changeset, squash bool) error {
	mr, ok := c.Changeset.Metadata.(*gitlab.MergeRequest)
	if !ok {
		return errors.New("Changeset is not a GitLab merge request")
	}
	project := c.Repo.Metadata.(*gitlab.Project)

	updated, err := s.client.MergeMergeRequest(ctx, project, mr, squash)
	if err != nil {
		if errors.Is(err, gitlab.ErrNotMergeable) {
			return ChangesetNotMergeableError{ErrorMsg: err.Error()}
		}
		return errors.Wrap(err, "merging GitLab merge request")
	}

	// These additional API calls can go away once we can use the GraphQL API.
	if err := s.decorateMergeRequestData(ctx, project, mr); err != nil {
		return errors.Wrapf(err, "retrieving additional data for merge request %d", mr.IID)
	}

	return c.Changeset.SetMetadata(updated)
}
