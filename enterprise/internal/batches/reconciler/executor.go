package reconciler

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/sources"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/state"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

// executePlan executes the given reconciler plan.
func executePlan(ctx context.Context, gitserverClient GitserverClient, sourcer sources.Sourcer, noSleepBeforeSync bool, tx *store.Store, plan *Plan) (err error) {
	e := &executor{
		gitserverClient:   gitserverClient,
		sourcer:           sourcer,
		noSleepBeforeSync: noSleepBeforeSync,
		tx:                tx,
		ch:                plan.Changeset,
		spec:              plan.ChangesetSpec,
	}

	return e.Run(ctx, plan)
}

type executor struct {
	gitserverClient   GitserverClient
	sourcer           sources.Sourcer
	noSleepBeforeSync bool
	tx                *store.Store
	ch                *btypes.Changeset
	spec              *btypes.ChangesetSpec

	css  sources.ChangesetSource
	repo *types.Repo
}

func (e *executor) Run(ctx context.Context, plan *Plan) (err error) {
	if plan.Ops.IsNone() {
		return nil
	}

	// Load the changeset repo.
	e.repo, err = e.tx.Repos().Get(ctx, e.ch.RepoID)
	if err != nil {
		return errors.Wrap(err, "failed to load repository")
	}

	// Load the changeset source.
	e.css, err = loadChangesetSource(ctx, e.tx, e.sourcer, e.ch, e.repo)
	if err != nil {
		return err
	}

	for _, op := range plan.Ops.ExecutionOrder() {
		switch op {
		case btypes.ReconcilerOperationSync:
			err = e.syncChangeset(ctx)

		case btypes.ReconcilerOperationImport:
			err = e.importChangeset(ctx)

		case btypes.ReconcilerOperationPush:
			err = e.pushChangesetPatch(ctx)

		case btypes.ReconcilerOperationPublish:
			err = e.publishChangeset(ctx, false)

		case btypes.ReconcilerOperationPublishDraft:
			err = e.publishChangeset(ctx, true)

		case btypes.ReconcilerOperationReopen:
			err = e.reopenChangeset(ctx)

		case btypes.ReconcilerOperationUpdate:
			err = e.updateChangeset(ctx)

		case btypes.ReconcilerOperationUndraft:
			err = e.undraftChangeset(ctx)

		case btypes.ReconcilerOperationClose:
			err = e.closeChangeset(ctx)

		case btypes.ReconcilerOperationSleep:
			e.sleep()

		case btypes.ReconcilerOperationDetach:
			e.detachChangeset()

		case btypes.ReconcilerOperationArchive:
			e.archiveChangeset()

		default:
			err = errors.Errorf("executor operation %q not implemented", op)
		}

		if err != nil {
			return err
		}
	}

	events, err := e.ch.Events()
	if err != nil {
		log15.Error("Events", "err", err)
		return errcode.MakeNonRetryable(err)
	}
	state.SetDerivedState(ctx, e.tx.Repos(), e.ch, events)

	if err := e.tx.UpsertChangesetEvents(ctx, events...); err != nil {
		log15.Error("UpsertChangesetEvents", "err", err)
		return err
	}

	return e.tx.UpdateChangeset(ctx, e.ch)
}

// pushChangesetPatch creates the commits for the changeset on its codehost.
func (e *executor) pushChangesetPatch(ctx context.Context) (err error) {
	existingSameBranch, err := e.tx.GetChangeset(ctx, store.GetChangesetOpts{
		ExternalServiceType: e.ch.ExternalServiceType,
		RepoID:              e.ch.RepoID,
		ExternalBranch:      e.spec.Spec.HeadRef,
		// TODO: Do we need to check whether it's published or not?
	})
	if err != nil && err != store.ErrNoResults {
		return err
	}

	if existingSameBranch != nil && existingSameBranch.ID != e.ch.ID {
		return errPublishSameBranch{}
	}

	// Create a commit and push it
	// Figure out which authenticator we should use to modify the changeset.
	// au is nil if we want to use the global credentials stored in the external
	// service configuration.
	pushConf, err := e.css.GitserverPushConfig(ctx, e.tx.ExternalServices(), e.repo)
	if err != nil {
		return err
	}
	opts, err := buildCommitOpts(e.repo, e.spec, pushConf)
	if err != nil {
		return err
	}
	return e.pushCommit(ctx, opts)
}

// publishChangeset creates the given changeset on its code host.
func (e *executor) publishChangeset(ctx context.Context, asDraft bool) (err error) {
	cs := &sources.Changeset{
		Title:     e.spec.Spec.Title,
		Body:      e.spec.Spec.Body,
		BaseRef:   e.spec.Spec.BaseRef,
		HeadRef:   e.spec.Spec.HeadRef,
		Repo:      e.repo,
		Changeset: e.ch,
	}

	// Depending on the changeset, we may want to add to the body (for example,
	// to add a backlink to Sourcegraph).
	if err := decorateChangesetBody(ctx, e.tx, database.NamespacesWith(e.tx), cs); err != nil {
		return errors.Wrapf(err, "decorating body for changeset %d", e.ch.ID)
	}

	var exists bool
	if asDraft {
		// If the changeset shall be published in draft mode, make sure the changeset source implements DraftChangesetSource.
		draftCss, err := sources.ToDraftChangesetSource(e.css)
		if err != nil {
			return err
		}
		exists, err = draftCss.CreateDraftChangeset(ctx, cs)
		if err != nil {
			return errors.Wrap(err, "creating draft changeset")
		}
	} else {
		// If we're running this method a second time, because we failed due to an
		// ephemeral error, there's a race condition here.
		// It's possible that `CreateChangeset` doesn't return the newest head ref
		// commit yet, because the API of the codehost doesn't return it yet.
		exists, err = e.css.CreateChangeset(ctx, cs)
		if err != nil {
			return errors.Wrap(err, "creating changeset")
		}
	}

	// If the Changeset already exists and our source can update it, we try to update it
	if exists {
		outdated, err := cs.IsOutdated()
		if err != nil {
			return errors.Wrap(err, "could not determine whether changeset needs update")
		}

		if outdated {
			if err := e.css.UpdateChangeset(ctx, cs); err != nil {
				return errors.Wrap(err, "updating changeset")
			}
		}
	}
	// Set the changeset to published.
	e.ch.PublicationState = btypes.ChangesetPublicationStatePublished
	return nil
}

func (e *executor) syncChangeset(ctx context.Context) error {
	if err := e.loadChangeset(ctx); err != nil {
		if !errors.HasType(err, sources.ChangesetNotFoundError{}) {
			return err
		}

		// If we're syncing a changeset and it can't be found anymore, we mark
		// it as deleted.
		if !e.ch.IsDeleted() {
			e.ch.SetDeleted()
		}
	}

	return nil
}

func (e *executor) importChangeset(ctx context.Context) error {
	if err := e.loadChangeset(ctx); err != nil {
		return err
	}

	// The changeset finished importing, so it is published now.
	e.ch.PublicationState = btypes.ChangesetPublicationStatePublished

	return nil
}

func (e *executor) loadChangeset(ctx context.Context) error {
	repoChangeset := &sources.Changeset{Repo: e.repo, Changeset: e.ch}
	return e.css.LoadChangeset(ctx, repoChangeset)
}

// updateChangeset updates the given changeset's attribute on the code host
// according to its ChangesetSpec and the delta previously computed.
func (e *executor) updateChangeset(ctx context.Context) (err error) {
	cs := sources.Changeset{
		Title:     e.spec.Spec.Title,
		Body:      e.spec.Spec.Body,
		BaseRef:   e.spec.Spec.BaseRef,
		HeadRef:   e.spec.Spec.HeadRef,
		Repo:      e.repo,
		Changeset: e.ch,
	}

	// Depending on the changeset, we may want to add to the body (for example,
	// to add a backlink to Sourcegraph).
	if err := decorateChangesetBody(ctx, e.tx, database.NamespacesWith(e.tx), &cs); err != nil {
		return errors.Wrapf(err, "decorating body for changeset %d", e.ch.ID)
	}

	if err := e.css.UpdateChangeset(ctx, &cs); err != nil {
		return errors.Wrap(err, "updating changeset")
	}

	return nil
}

// reopenChangeset reopens the given changeset attribute on the code host.
func (e *executor) reopenChangeset(ctx context.Context) (err error) {
	cs := sources.Changeset{Repo: e.repo, Changeset: e.ch}
	if err := e.css.ReopenChangeset(ctx, &cs); err != nil {
		return errors.Wrap(err, "updating changeset")
	}
	return nil
}

func (e *executor) detachChangeset() {
	for _, assoc := range e.ch.BatchChanges {
		if assoc.Detach {
			e.ch.RemoveBatchChangeID(assoc.BatchChangeID)
		}
	}
}

// archiveChangeset sets all associations to archived that are marked as "to-be-archived".
func (e *executor) archiveChangeset() {
	for i, assoc := range e.ch.BatchChanges {
		if assoc.Archive {
			e.ch.BatchChanges[i].IsArchived = true
			e.ch.BatchChanges[i].Archive = false
		}
	}
}

// closeChangeset closes the given changeset on its code host if its ExternalState is OPEN or DRAFT.
func (e *executor) closeChangeset(ctx context.Context) (err error) {
	e.ch.Closing = false

	if e.ch.ExternalState != btypes.ChangesetExternalStateDraft && e.ch.ExternalState != btypes.ChangesetExternalStateOpen {
		return nil
	}

	cs := &sources.Changeset{Changeset: e.ch, Repo: e.repo}

	if err := e.css.CloseChangeset(ctx, cs); err != nil {
		return errors.Wrap(err, "closing changeset")
	}
	return nil
}

// undraftChangeset marks the given changeset on its code host as ready for review.
func (e *executor) undraftChangeset(ctx context.Context) (err error) {
	draftCss, err := sources.ToDraftChangesetSource(e.css)
	if err != nil {
		return err
	}

	cs := &sources.Changeset{
		Title:     e.spec.Spec.Title,
		Body:      e.spec.Spec.Body,
		BaseRef:   e.spec.Spec.BaseRef,
		HeadRef:   e.spec.Spec.HeadRef,
		Repo:      e.repo,
		Changeset: e.ch,
	}

	if err := draftCss.UndraftChangeset(ctx, cs); err != nil {
		return errors.Wrap(err, "undrafting changeset")
	}
	return nil
}

// sleep sleeps for 3 seconds.
func (e *executor) sleep() {
	if !e.noSleepBeforeSync {
		time.Sleep(3 * time.Second)
	}
}

func loadChangesetSource(ctx context.Context, s *store.Store, sourcer sources.Sourcer, ch *btypes.Changeset, repo *types.Repo) (sources.ChangesetSource, error) {
	// This is a changeset source using the external service config for authentication,
	// based on our heuristic in the sources package.
	css, err := sourcer.ForRepo(ctx, s, repo)
	if err != nil {
		return nil, err
	}
	if ch.OwnedByBatchChangeID != 0 {
		// If the changeset is owned by a batch change, we want to reconcile using
		// the user's credentials, which means we need to know which user last
		// applied the owning batch change. Let's go find out.
		batchChange, err := loadBatchChange(ctx, s, ch.OwnedByBatchChangeID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load owning batch change")
		}
		css, err = sources.WithAuthenticatorForUser(ctx, s, css, batchChange.LastApplierID, repo)
		if err != nil {
			switch err {
			case sources.ErrMissingCredentials:
				return nil, &errMissingCredentials{repo: string(repo.Name)}
			case sources.ErrNoSSHCredential:
				return nil, &errNoSSHCredential{}
			default:
				var e sources.ErrNoPushCredentials
				if errors.As(err, &e) {
					return nil, &errNoPushCredentials{credentialsType: e.CredentialsType}
				}
				return nil, err
			}
		}
	} else {
		// This retains the external service token, when no site credential is found.
		// Unowned changesets are imported, and therefore don't need to use a user
		// credential, since reconciliation isn't a mutating process. We try to use
		// a site-credential, but it's ok if it doesn't exist.
		// TODO: This code-path will fail once the site credentials are the only
		// fallback we want to use.
		css, err = sources.WithSiteAuthenticator(ctx, s, css, repo)
		if err != nil {
			return nil, err
		}
	}
	return css, nil
}

func (e *executor) pushCommit(ctx context.Context, opts protocol.CreateCommitFromPatchRequest) error {
	_, err := e.gitserverClient.CreateCommitFromPatch(ctx, opts)
	if err != nil {
		var e *protocol.CreateCommitFromPatchError
		if errors.As(err, &e) {
			return errors.Errorf(
				"creating commit from patch for repository %q: %s\n"+
					"```\n"+
					"$ %s\n"+
					"%s\n"+
					"```",
				e.RepositoryName, e.InternalError, e.Command, strings.TrimSpace(e.CombinedOutput))
		}
		return err
	}

	return nil
}

func buildCommitOpts(repo *types.Repo, spec *btypes.ChangesetSpec, pushOpts *protocol.PushConfig) (opts protocol.CreateCommitFromPatchRequest, err error) {
	desc := spec.Spec

	diff, err := desc.Diff()
	if err != nil {
		return opts, err
	}

	commitMessage, err := desc.CommitMessage()
	if err != nil {
		return opts, err
	}

	commitAuthorName, err := desc.AuthorName()
	if err != nil {
		return opts, err
	}

	commitAuthorEmail, err := desc.AuthorEmail()
	if err != nil {
		return opts, err
	}

	opts = protocol.CreateCommitFromPatchRequest{
		Repo:       repo.Name,
		BaseCommit: api.CommitID(desc.BaseRev),
		// IMPORTANT: We add a trailing newline here, otherwise `git apply`
		// will fail with "corrupt patch at line <N>" where N is the last line.
		Patch:     diff + "\n",
		TargetRef: desc.HeadRef,

		// CAUTION: `UniqueRef` means that we'll push to the branch even if it
		// already exists.
		// So when we retry publishing a changeset, this will overwrite what we
		// pushed before.
		UniqueRef: false,

		CommitInfo: protocol.PatchCommitInfo{
			Message:     commitMessage,
			AuthorName:  commitAuthorName,
			AuthorEmail: commitAuthorEmail,
			Date:        spec.CreatedAt,
		},
		// We use unified diffs, not git diffs, which means they're missing the
		// `a/` and `b/` filename prefixes. `-p0` tells `git apply` to not
		// expect and strip prefixes.
		GitApplyArgs: []string{"-p0"},
		Push:         pushOpts,
	}

	return opts, nil
}

type getBatchChanger interface {
	GetBatchChange(ctx context.Context, opts store.GetBatchChangeOpts) (*btypes.BatchChange, error)
}

func loadBatchChange(ctx context.Context, tx getBatchChanger, id int64) (*btypes.BatchChange, error) {
	if id == 0 {
		return nil, errors.New("changeset has no owning batch change")
	}

	batchChange, err := tx.GetBatchChange(ctx, store.GetBatchChangeOpts{ID: id})
	if err != nil && err != store.ErrNoResults {
		return nil, errors.Wrapf(err, "retrieving owning batch change: %d", id)
	} else if batchChange == nil {
		return nil, errors.Errorf("batch change not found: %d", id)
	}

	return batchChange, nil
}

type getNamespacer interface {
	GetByID(ctx context.Context, orgID, userID int32) (*database.Namespace, error)
}

func decorateChangesetBody(ctx context.Context, tx getBatchChanger, nsStore getNamespacer, cs *sources.Changeset) error {
	batchChange, err := loadBatchChange(ctx, tx, cs.OwnedByBatchChangeID)
	if err != nil {
		return errors.Wrap(err, "failed to load batch change")
	}

	// We need to get the namespace, since external batch change URLs are
	// namespaced.
	ns, err := nsStore.GetByID(ctx, batchChange.NamespaceOrgID, batchChange.NamespaceUserID)
	if err != nil {
		return errors.Wrap(err, "retrieving namespace")
	}

	u, err := batchChangeURL(ctx, ns, batchChange)
	if err != nil {
		return errors.Wrap(err, "building URL")
	}

	cs.Body = fmt.Sprintf(
		"%s\n\n[_Created by Sourcegraph batch change `%s/%s`._](%s)",
		cs.Body, ns.Name, batchChange.Name, u,
	)

	return nil
}

// internalClient is here for mocking reasons.
var internalClient interface {
	ExternalURL(context.Context) (string, error)
} = api.InternalClient

func batchChangeURL(ctx context.Context, ns *database.Namespace, c *btypes.BatchChange) (string, error) {
	// To build the absolute URL, we need to know where Sourcegraph is!
	extStr, err := internalClient.ExternalURL(ctx)
	if err != nil {
		return "", errors.Wrap(err, "getting external Sourcegraph URL")
	}

	extURL, err := url.Parse(extStr)
	if err != nil {
		return "", errors.Wrap(err, "parsing external Sourcegraph URL")
	}

	// This needs to be kept consistent with resolvers.batchChangeURL().
	// (Refactoring the resolver to use the same function is difficult due to
	// the different querying and caching behaviour in GraphQL resolvers, so we
	// simply replicate the logic here.)
	u := extURL.ResolveReference(&url.URL{Path: namespaceURL(ns) + "/batch-changes/" + c.Name})

	return u.String(), nil
}

func namespaceURL(ns *database.Namespace) string {
	prefix := "/users/"
	if ns.Organization != 0 {
		prefix = "/organizations/"
	}

	return prefix + ns.Name
}

// errPublishSameBranch is returned by publish changeset if a changeset with
// the same external branch already exists in the database and is owned by
// another batch change.
// It is a terminal error that won't be fixed by retrying to publish the
// changeset with the same spec.
type errPublishSameBranch struct{}

func (e errPublishSameBranch) Error() string {
	return "cannot create changeset on the same branch in multiple batch changes"
}

func (e errPublishSameBranch) NonRetryable() bool { return true }

// errNoSSHCredential is returned, if the  clone URL of the repository uses the
// ssh:// scheme, but the authenticator doesn't support SSH pushes.
type errNoSSHCredential struct{}

func (e errNoSSHCredential) Error() string {
	return "The used credential doesn't support SSH pushes, but the repo requires pushing over SSH."
}

func (e errNoSSHCredential) NonRetryable() bool { return true }

// errMissingCredentials is returned if the user that applied the last batch change
// /changeset spec doesn't have a user credential for the given repository and is
// not a site-admin (so no fallback to the global credentials is possible).
type errMissingCredentials struct{ repo string }

func (e errMissingCredentials) Error() string {
	return fmt.Sprintf("user does not have a valid credential for repository %q", e.repo)
}

func (e errMissingCredentials) NonRetryable() bool { return true }

func (e errMissingCredentials) Is(target error) bool {
	if t, ok := target.(errMissingCredentials); ok && t.repo == e.repo {
		return true
	}
	return false
}

// errNoPushCredentials is returned if the authenticator cannot be used by git to
// authenticate a `git push`.
type errNoPushCredentials struct{ credentialsType string }

func (e errNoPushCredentials) Error() string {
	return fmt.Sprintf("cannot use credentials of type %s to push commits", e.credentialsType)
}

func (e errNoPushCredentials) NonRetryable() bool { return true }
