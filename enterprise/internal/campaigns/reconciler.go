package campaigns

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
	"github.com/sourcegraph/sourcegraph/schema"
)

type GitserverClient interface {
	CreateCommitFromPatch(ctx context.Context, req protocol.CreateCommitFromPatchRequest) (string, error)
}

// reconciler processes changesets and reconciles their current state — in
// Sourcegraph or on the code host — with that described in the current
// ChangesetSpec associated with the changeset.
type reconciler struct {
	gitserverClient GitserverClient
	sourcer         repos.Sourcer
	store           *Store

	// This is used to disable a time.Sleep for operationSleep so that the
	// tests don't run slower.
	noSleepBeforeSync bool
}

// HandlerFunc returns a dbworker.HandlerFunc that can be passed to a
// workerutil.Worker to process queued changesets.
func (r *reconciler) HandlerFunc() dbworker.HandlerFunc {
	return func(ctx context.Context, tx dbworkerstore.Store, record workerutil.Record) error {
		return r.process(ctx, r.store.With(tx), record.(*campaigns.Changeset))
	}
}

// process is the main entry point of the reconciler and processes changesets
// that were marked as queued in the database.
//
// For each changeset, the reconciler computes an execution plan to run to reconcile a
// possible divergence between the changeset's current state and the desired
// state (for example expressed in a changeset spec).
//
// To do that, the reconciler looks at the changeset's current state
// (publication state, external state, sync state, ...), its (if set) current
// ChangesetSpec, and (if it exists) its previous ChangesetSpec.
//
// If an error is returned, the workerutil.Worker that called this function
// (through the HandlerFunc) will set the changeset's ReconcilerState to
// errored and set its FailureMessage to the error.
func (r *reconciler) process(ctx context.Context, tx *Store, ch *campaigns.Changeset) error {
	// Reset the error message.
	ch.FailureMessage = nil

	prev, curr, err := loadChangesetSpecs(ctx, tx, ch)
	if err != nil {
		return nil
	}

	plan, err := determinePlan(prev, curr, ch)
	if err != nil {
		return err
	}

	log15.Info("Reconciler processing changeset", "changeset", ch.ID, "operations", plan.ops)

	e := &executor{
		sourcer:           r.sourcer,
		gitserverClient:   r.gitserverClient,
		noSleepBeforeSync: r.noSleepBeforeSync,

		tx: tx,
		ch: ch,

		spec:  curr,
		delta: plan.delta,
	}

	return e.ExecutePlan(ctx, plan)
}

// ErrPublishSameBranch is returned by publish changeset if a changeset with the same external branch
// already exists in the database and is owned by another campaign.
var ErrPublishSameBranch = errors.New("cannot create changeset on the same branch in multiple campaigns")

type executor struct {
	gitserverClient   GitserverClient
	sourcer           repos.Sourcer
	noSleepBeforeSync bool

	tx  *Store
	ccs repos.ChangesetSource

	repo *repos.Repo

	ch    *campaigns.Changeset
	spec  *campaigns.ChangesetSpec
	delta *changesetSpecDelta
}

// ExecutePlan executes the given reconciler plan.
func (e *executor) ExecutePlan(ctx context.Context, plan *plan) (err error) {
	if plan.ops.IsNone() {
		return nil
	}

	reposStore := repos.NewDBStore(e.tx.Handle().DB(), sql.TxOptions{})

	e.repo, err = loadRepo(ctx, reposStore, e.ch.RepoID)
	if err != nil {
		return errors.Wrap(err, "failed to load repository")
	}

	extSvc, err := loadExternalService(ctx, reposStore, e.repo)
	if err != nil {
		return errors.Wrap(err, "failed to load external service")
	}

	// Set up a source with which we can modify the changeset.
	e.ccs, err = e.buildChangesetSource(e.repo, extSvc)
	if err != nil {
		return err
	}

	upsertChangesetEvents := true
	for _, op := range plan.ops.ExecutionOrder() {
		switch op {
		case operationSync:
			err = e.syncChangeset(ctx)

		case operationImport:
			var notFound bool
			notFound, err = e.importChangeset(ctx)
			if notFound {
				upsertChangesetEvents = false
			}

		case operationPush:
			err = e.pushChangesetPatch(ctx)

		case operationPublish:
			err = e.publishChangeset(ctx, false)

		case operationPublishDraft:
			err = e.publishChangeset(ctx, true)

		case operationReopen:
			err = e.reopenChangeset(ctx)

		case operationUpdate:
			err = e.updateChangeset(ctx)

		case operationUndraft:
			err = e.undraftChangeset(ctx)

		case operationClose:
			err = e.closeChangeset(ctx)

		case operationSleep:
			e.sleep()

		default:
			err = fmt.Errorf("executor operation %q not implemented", op)
		}

		if err != nil {
			return err
		}
	}

	if upsertChangesetEvents {
		events := e.ch.Events()
		SetDerivedState(ctx, e.ch, events)

		if err := e.tx.UpsertChangesetEvents(ctx, events...); err != nil {
			log15.Error("UpsertChangesetEvents", "err", err)
			return err
		}
	}

	return e.tx.UpdateChangeset(ctx, e.ch)
}

func (e *executor) buildChangesetSource(repo *repos.Repo, extSvc *repos.ExternalService) (repos.ChangesetSource, error) {
	sources, err := e.sourcer(extSvc)
	if err != nil {
		return nil, err
	}
	if len(sources) != 1 {
		return nil, errors.New("invalid number of sources for external service")
	}
	src := sources[0]
	ccs, ok := src.(repos.ChangesetSource)
	if !ok {
		return nil, errors.Errorf("creating changesets on code host of repo %q is not implemented", repo.Name)
	}

	return ccs, nil
}

// pushChangesetPatch creates the commits for the changeset on its codehost.
func (e *executor) pushChangesetPatch(ctx context.Context) (err error) {
	existingSameBranch, err := e.tx.GetChangeset(ctx, GetChangesetOpts{
		ExternalServiceType: e.ch.ExternalServiceType,
		RepoID:              e.ch.RepoID,
		ExternalBranch:      git.AbbreviateRef(e.spec.Spec.HeadRef),
	})
	if err != nil && err != ErrNoResults {
		return err
	}

	if existingSameBranch != nil && existingSameBranch.ID != e.ch.ID {
		return ErrPublishSameBranch
	}

	// Create a commit and push it
	opts, err := buildCommitOpts(e.repo, e.spec)
	if err != nil {
		return err
	}
	return e.pushCommit(ctx, opts)
}

// publishChangeset creates the given changeset on its code host.
func (e *executor) publishChangeset(ctx context.Context, asDraft bool) (err error) {
	cs := &repos.Changeset{
		Title:     e.spec.Spec.Title,
		Body:      e.spec.Spec.Body,
		BaseRef:   e.spec.Spec.BaseRef,
		HeadRef:   git.EnsureRefPrefix(e.spec.Spec.HeadRef),
		Repo:      e.repo,
		Changeset: e.ch,
	}

	// Depending on the changeset, we may want to add to the body (for example,
	// to add a backlink to Sourcegraph).
	if err := decorateChangesetBody(ctx, e.tx, cs); err != nil {
		return errors.Wrapf(err, "decorating body for changeset %d", e.ch.ID)
	}

	var exists bool
	if asDraft {
		// If the changeset shall be published in draft mode, make sure the changeset source implements DraftChangesetSource.
		draftCcs, ok := e.ccs.(repos.DraftChangesetSource)
		if !ok {
			return errors.New("changeset operation is publish-draft, but changeset source doesn't implement DraftChangesetSource")
		}
		exists, err = draftCcs.CreateDraftChangeset(ctx, cs)
	} else {
		// If we're running this method a second time, because we failed due to an
		// ephemeral error, there's a race condition here.
		// It's possible that `CreateChangeset` doesn't return the newest head ref
		// commit yet, because the API of the codehost doesn't return it yet.
		exists, err = e.ccs.CreateChangeset(ctx, cs)
	}
	if err != nil {
		return errors.Wrap(err, "creating changeset")
	}
	// If the Changeset already exists and our source can update it, we try to update it
	if exists {
		outdated, err := cs.IsOutdated()
		if err != nil {
			return errors.Wrap(err, "could not determine whether changeset needs update")
		}

		if outdated {
			if err := e.ccs.UpdateChangeset(ctx, cs); err != nil {
				return errors.Wrap(err, "updating changeset")
			}
		}
	}
	// Set the changeset to published.
	e.ch.PublicationState = campaigns.ChangesetPublicationStatePublished
	return nil
}

func (e *executor) syncChangeset(ctx context.Context) error {
	if err := e.loadChangeset(ctx); err != nil {
		_, ok := err.(repos.ChangesetNotFoundError)
		if !ok {
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

func (e *executor) importChangeset(ctx context.Context) (bool, error) {
	if err := e.loadChangeset(ctx); err != nil {
		_, ok := err.(repos.ChangesetNotFoundError)
		if !ok {
			return false, err
		}

		// If we're importing and it can't be found, we want to mark the
		// changeset as "dead" and never retry:

		msg := err.Error()
		e.ch.FailureMessage = &msg
		e.ch.ReconcilerState = campaigns.ReconcilerStateErrored
		e.ch.NumFailures = reconcilerMaxNumRetries + 999
		return true, nil
	}

	e.ch.Unsynced = false

	return false, nil
}

func (e *executor) loadChangeset(ctx context.Context) error {
	repoChangeset := &repos.Changeset{Repo: e.repo, Changeset: e.ch}
	return e.ccs.LoadChangeset(ctx, repoChangeset)
}

// updateChangeset updates the given changeset's attribute on the code host
// according to its ChangesetSpec and the delta previously computed.
func (e *executor) updateChangeset(ctx context.Context) (err error) {
	cs := repos.Changeset{
		Title:     e.spec.Spec.Title,
		Body:      e.spec.Spec.Body,
		BaseRef:   e.spec.Spec.BaseRef,
		HeadRef:   git.EnsureRefPrefix(e.spec.Spec.HeadRef),
		Repo:      e.repo,
		Changeset: e.ch,
	}

	// Depending on the changeset, we may want to add to the body (for example,
	// to add a backlink to Sourcegraph).
	if err := decorateChangesetBody(ctx, e.tx, &cs); err != nil {
		return errors.Wrapf(err, "decorating body for changeset %d", e.ch.ID)
	}

	if err := e.ccs.UpdateChangeset(ctx, &cs); err != nil {
		return errors.Wrap(err, "updating changeset")
	}

	return nil
}

// reopenChangeset reopens the given changeset attribute on the code host.
func (e *executor) reopenChangeset(ctx context.Context) (err error) {
	cs := repos.Changeset{Repo: e.repo, Changeset: e.ch}
	if err := e.ccs.ReopenChangeset(ctx, &cs); err != nil {
		return errors.Wrap(err, "updating changeset")
	}
	return nil
}

// closeChangeset closes the given changeset on its code host if its ExternalState is OPEN or DRAFT.
func (e *executor) closeChangeset(ctx context.Context) (err error) {
	e.ch.Closing = false

	if e.ch.ExternalState != campaigns.ChangesetExternalStateDraft && e.ch.ExternalState != campaigns.ChangesetExternalStateOpen {
		return nil
	}

	cs := &repos.Changeset{Changeset: e.ch, Repo: e.repo}

	if err := e.ccs.CloseChangeset(ctx, cs); err != nil {
		return errors.Wrap(err, "closing changeset")
	}
	return nil
}

// undraftChangeset marks the given changeset on its code host as ready for review.
func (e *executor) undraftChangeset(ctx context.Context) (err error) {
	draftCcs, ok := e.ccs.(repos.DraftChangesetSource)
	if !ok {
		return errors.New("changeset operation is undraft, but changeset source doesn't implement DraftChangesetSource")
	}

	cs := &repos.Changeset{
		Title:     e.spec.Spec.Title,
		Body:      e.spec.Spec.Body,
		BaseRef:   e.spec.Spec.BaseRef,
		HeadRef:   git.EnsureRefPrefix(e.spec.Spec.HeadRef),
		Repo:      e.repo,
		Changeset: e.ch,
	}

	if err := draftCcs.UndraftChangeset(ctx, cs); err != nil {
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

func (e *executor) pushCommit(ctx context.Context, opts protocol.CreateCommitFromPatchRequest) error {
	_, err := e.gitserverClient.CreateCommitFromPatch(ctx, opts)
	if err != nil {
		if diffErr, ok := err.(*protocol.CreateCommitFromPatchError); ok {
			return errors.Errorf(
				"creating commit from patch for repository %q: %s\n"+
					"```\n"+
					"$ %s\n"+
					"%s\n"+
					"```",
				diffErr.RepositoryName, diffErr.InternalError, diffErr.Command, strings.TrimSpace(diffErr.CombinedOutput))
		}
		return err
	}

	return nil
}

func buildCommitOpts(repo *repos.Repo, spec *campaigns.ChangesetSpec) (protocol.CreateCommitFromPatchRequest, error) {
	var opts protocol.CreateCommitFromPatchRequest

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
		Repo:       api.RepoName(repo.Name),
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
		Push:         true,
	}

	return opts, nil
}

// operation is an enum to distinguish between different reconciler operations.
type operation string

const (
	operationPush         operation = "push"
	operationUpdate       operation = "update"
	operationUndraft      operation = "undraft"
	operationPublish      operation = "publish"
	operationPublishDraft operation = "publish-draft"
	operationSync         operation = "sync"
	operationImport       operation = "import"
	operationClose        operation = "close"
	operationReopen       operation = "reopen"
	operationSleep        operation = "sleep"
)

var operationPrecedence = map[operation]int{
	operationPush:         0,
	operationImport:       1,
	operationPublish:      1,
	operationPublishDraft: 1,
	operationClose:        1,
	operationReopen:       2,
	operationUndraft:      3,
	operationUpdate:       4,
	operationSleep:        5,
	operationSync:         6,
}

type operations []operation

func (ops operations) IsNone() bool {
	return len(ops) == 0
}

func (ops operations) Equal(b operations) bool {
	if len(ops) != len(b) {
		return false
	}
	bEntries := make(map[operation]struct{})
	for _, e := range b {
		bEntries[e] = struct{}{}
	}

	for _, op := range ops {
		if _, ok := bEntries[op]; !ok {
			return false
		}
	}

	return true
}

func (ops operations) String() string {
	if ops.IsNone() {
		return "No operations required"
	}
	eo := ops.ExecutionOrder()
	ss := make([]string, len(eo))
	for i, val := range eo {
		ss[i] = string(val)
	}
	return strings.Join(ss, " => ")
}

func (ops operations) ExecutionOrder() []operation {
	uniqueOps := []operation{}

	// Make sure ops are unique.
	seenOps := make(map[operation]struct{})
	for _, op := range ops {
		if _, ok := seenOps[op]; ok {
			continue
		}

		seenOps[op] = struct{}{}
		uniqueOps = append(uniqueOps, op)
	}

	sort.Slice(uniqueOps, func(i, j int) bool {
		return operationPrecedence[uniqueOps[i]] < operationPrecedence[uniqueOps[j]]
	})

	return uniqueOps
}

// plan represents the possible operations the reconciler needs to do
// to reconcile the current and the desired state of a changeset.
type plan struct {
	// The operations that need to be done to reconcile the changeset.
	ops operations

	// The delta between a possible previous ChangesetSpec and the current
	// ChangesetSpec.
	delta *changesetSpecDelta
}

func (p *plan) AddOp(op operation) { p.ops = append(p.ops, op) }
func (p *plan) SetOp(op operation) { p.ops = operations{op} }

// determinePlan looks at the given changeset to determine what action the
// reconciler should take.
// It loads the current ChangesetSpec and if it exists also the previous one.
// If the current ChangesetSpec is not applied to a campaign, it returns an
// error.
func determinePlan(previousSpec, currentSpec *campaigns.ChangesetSpec, ch *campaigns.Changeset) (*plan, error) {
	pl := &plan{}

	// If it doesn't have a spec, it's an imported changeset and we can't do
	// anything.
	if currentSpec == nil {
		if ch.Unsynced {
			pl.SetOp(operationImport)
		}
		return pl, nil
	}

	// If it's marked as closing, we don't need to look at the specs.
	if ch.Closing {
		pl.SetOp(operationClose)
		return pl, nil
	}

	delta, err := compareChangesetSpecs(previousSpec, currentSpec)
	if err != nil {
		return pl, nil
	}
	pl.delta = delta

	switch ch.PublicationState {
	case campaigns.ChangesetPublicationStateUnpublished:
		if currentSpec.Spec.Published.True() {
			pl.SetOp(operationPublish)
			pl.AddOp(operationPush)
		} else if currentSpec.Spec.Published.Draft() && ch.SupportsDraft() {
			// If configured to be opened as draft, and the changeset supports
			// draft mode, publish as draft. Otherwise, take no action.
			pl.SetOp(operationPublishDraft)
			pl.AddOp(operationPush)
		}

	case campaigns.ChangesetPublicationStatePublished:
		if reopenAfterDetach(ch) {
			pl.SetOp(operationReopen)
		}

		// Only do undraft, when the codehost supports draft changesets.
		if delta.undraft && campaigns.ExternalServiceSupports(ch.ExternalServiceType, campaigns.CodehostCapabilityDraftChangesets) {
			pl.AddOp(operationUndraft)
		}

		if delta.AttributesChanged() {
			if delta.NeedCommitUpdate() {
				pl.AddOp(operationPush)
			}

			// If we only need to update the diff and we didn't change the state of the changeset,
			// we're done, because we already pushed the commit. We don't need to
			// update anything on the codehost.
			if !delta.NeedCodeHostUpdate() {
				// But we need to sync the changeset so that it has the new commit.
				//
				// The problem: the code host might not have updated the changeset to
				// have the new commit SHA as its head ref oid (and the check states,
				// ...).
				//
				// That's why we give them 3 seconds to update the changesets.
				//
				// Why 3 seconds? Well... 1 or 2 seem to be too short and 4 too long?
				pl.AddOp(operationSleep)
				pl.AddOp(operationSync)
			} else {
				// Otherwise, we need to update the pull request on the code host or, if we
				// need to reopen it, update it to make sure it has the newest state.
				pl.AddOp(operationUpdate)
			}
		}

	default:
		return pl, fmt.Errorf("unknown changeset publication state: %s", ch.PublicationState)
	}

	return pl, nil
}

func reopenAfterDetach(ch *campaigns.Changeset) bool {
	closed := ch.ExternalState == campaigns.ChangesetExternalStateClosed
	if !closed {
		return false
	}

	// Sanity check: if it's not owned by a campaign, it's simply being tracked.
	if ch.OwnedByCampaignID == 0 {
		return false
	}
	// Sanity check 2: if it's marked as to-be-closed, then we don't reopen it.
	if ch.Closing {
		return false
	}

	// Check if it's (re-)attached to the campaign that created it.
	attachedToOwner := false
	for _, campaignID := range ch.CampaignIDs {
		if campaignID == ch.OwnedByCampaignID {
			attachedToOwner = true
		}
	}

	// At this point the changeset is closed and not marked as to-be-closed and
	// attached to the owning campaign.
	return attachedToOwner

	// TODO: What if somebody closed the changeset on purpose on the codehost?
}

func loadRepo(ctx context.Context, tx repos.Store, id api.RepoID) (*repos.Repo, error) {
	rs, err := tx.ListRepos(ctx, repos.StoreListReposArgs{IDs: []api.RepoID{id}})
	if err != nil {
		return nil, err
	}
	if len(rs) != 1 {
		return nil, errors.Errorf("repo not found: %d", id)
	}
	return rs[0], nil
}

func loadExternalService(ctx context.Context, reposStore repos.Store, repo *repos.Repo) (*repos.ExternalService, error) {
	var externalService *repos.ExternalService
	{
		args := repos.StoreListExternalServicesArgs{IDs: repo.ExternalServiceIDs()}

		es, err := reposStore.ListExternalServices(ctx, args)
		if err != nil {
			return nil, err
		}

		for _, e := range es {
			cfg, err := e.Configuration()
			if err != nil {
				return nil, err
			}

			switch cfg := cfg.(type) {
			case *schema.GitHubConnection:
				if cfg.Token != "" {
					externalService = e
				}
			case *schema.BitbucketServerConnection:
				if cfg.Token != "" {
					externalService = e
				}
			case *schema.GitLabConnection:
				if cfg.Token != "" {
					externalService = e
				}
			}
			if externalService != nil {
				break
			}
		}
	}

	if externalService == nil {
		return nil, errors.Errorf("no external services found for repo %q", repo.Name)
	}

	return externalService, nil
}

func loadCampaign(ctx context.Context, tx *Store, id int64) (*campaigns.Campaign, error) {
	if id == 0 {
		return nil, errors.New("changeset has no owning campaign")
	}

	campaign, err := tx.GetCampaign(ctx, GetCampaignOpts{ID: id})
	if err != nil && err != ErrNoResults {
		return nil, errors.Wrapf(err, "retrieving owning campaign: %d", id)
	} else if campaign == nil {
		return nil, errors.Errorf("campaign not found: %d", id)
	}

	return campaign, nil
}

func loadChangesetSpecs(ctx context.Context, tx *Store, ch *campaigns.Changeset) (prev, curr *campaigns.ChangesetSpec, err error) {
	if ch.CurrentSpecID != 0 {
		curr, err = tx.GetChangesetSpecByID(ctx, ch.CurrentSpecID)
		if err != nil {
			return
		}
	}
	if ch.PreviousSpecID != 0 {
		prev, err = tx.GetChangesetSpecByID(ctx, ch.PreviousSpecID)
		if err != nil {
			return
		}
	}
	return
}

func decorateChangesetBody(ctx context.Context, tx *Store, cs *repos.Changeset) error {
	campaign, err := loadCampaign(ctx, tx, cs.OwnedByCampaignID)
	if err != nil {
		return errors.Wrap(err, "failed to load campaign")
	}

	// We need to get the namespace, since external campaign URLs are
	// namespaced.
	ns, err := db.Namespaces.GetByID(ctx, campaign.NamespaceOrgID, campaign.NamespaceUserID)
	if err != nil {
		return errors.Wrap(err, "retrieving namespace")
	}

	url, err := campaignURL(ctx, ns, campaign)
	if err != nil {
		return errors.Wrap(err, "building URL")
	}

	cs.Body = fmt.Sprintf(
		"%s\n\n[_Created by Sourcegraph campaign `%s/%s`._](%s)",
		cs.Body, ns.Name, campaign.Name, url,
	)

	return nil
}

// internalClient is here for mocking reasons.
var internalClient interface {
	ExternalURL(context.Context) (string, error)
} = api.InternalClient

func campaignURL(ctx context.Context, ns *db.Namespace, c *campaigns.Campaign) (string, error) {
	// To build the absolute URL, we need to know where Sourcegraph is!
	extStr, err := internalClient.ExternalURL(ctx)
	if err != nil {
		return "", errors.Wrap(err, "getting external Sourcegraph URL")
	}

	extURL, err := url.Parse(extStr)
	if err != nil {
		return "", errors.Wrap(err, "parsing external Sourcegraph URL")
	}

	// This needs to be kept consistent with resolvers.campaignURL().
	// (Refactoring the resolver to use the same function is difficult due to
	// the different querying and caching behaviour in GraphQL resolvers, so we
	// simply replicate the logic here.)
	u := extURL.ResolveReference(&url.URL{Path: namespaceURL(ns) + "/campaigns/" + c.Name})

	return u.String(), nil
}

func namespaceURL(ns *db.Namespace) string {
	prefix := "/users/"
	if ns.Organization != 0 {
		prefix = "/organizations/"
	}

	return prefix + ns.Name
}

func compareChangesetSpecs(previous, current *campaigns.ChangesetSpec) (*changesetSpecDelta, error) {
	delta := &changesetSpecDelta{}

	if previous == nil {
		return delta, nil
	}

	if previous.Spec.Title != current.Spec.Title {
		delta.titleChanged = true
	}
	if previous.Spec.Body != current.Spec.Body {
		delta.bodyChanged = true
	}
	if previous.Spec.BaseRef != current.Spec.BaseRef {
		delta.baseRefChanged = true
	}

	// If was set to "draft" and now "true", need to undraft the changeset.
	// We currently ignore going from "true" to "draft".
	if previous.Spec.Published.Draft() && current.Spec.Published.True() {
		delta.undraft = true
	}

	// Diff
	currentDiff, err := current.Spec.Diff()
	if err != nil {
		return nil, nil
	}
	previousDiff, err := previous.Spec.Diff()
	if err != nil {
		return nil, err
	}
	if previousDiff != currentDiff {
		delta.diffChanged = true
	}

	// CommitMessage
	currentCommitMessage, err := current.Spec.CommitMessage()
	if err != nil {
		return nil, nil
	}
	previousCommitMessage, err := previous.Spec.CommitMessage()
	if err != nil {
		return nil, err
	}
	if previousCommitMessage != currentCommitMessage {
		delta.commitMessageChanged = true
	}

	// AuthorName
	currentAuthorName, err := current.Spec.AuthorName()
	if err != nil {
		return nil, nil
	}
	previousAuthorName, err := previous.Spec.AuthorName()
	if err != nil {
		return nil, err
	}
	if previousAuthorName != currentAuthorName {
		delta.authorNameChanged = true
	}

	// AuthorEmail
	currentAuthorEmail, err := current.Spec.AuthorEmail()
	if err != nil {
		return nil, nil
	}
	previousAuthorEmail, err := previous.Spec.AuthorEmail()
	if err != nil {
		return nil, err
	}
	if previousAuthorEmail != currentAuthorEmail {
		delta.authorEmailChanged = true
	}

	return delta, nil
}

type changesetSpecDelta struct {
	titleChanged         bool
	bodyChanged          bool
	undraft              bool
	baseRefChanged       bool
	diffChanged          bool
	commitMessageChanged bool
	authorNameChanged    bool
	authorEmailChanged   bool
}

func (d *changesetSpecDelta) String() string { return fmt.Sprintf("%#v", d) }

func (d *changesetSpecDelta) NeedCommitUpdate() bool {
	return d.diffChanged || d.commitMessageChanged || d.authorNameChanged || d.authorEmailChanged
}

func (d *changesetSpecDelta) NeedCodeHostUpdate() bool {
	return d.titleChanged || d.bodyChanged || d.baseRefChanged
}

func (d *changesetSpecDelta) AttributesChanged() bool {
	return d.NeedCommitUpdate() || d.NeedCodeHostUpdate()
}
