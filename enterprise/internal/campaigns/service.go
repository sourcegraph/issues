package campaigns

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

// NewService returns a Service.
func NewService(store *Store, cf *httpcli.Factory) *Service {
	return NewServiceWithClock(store, cf, store.Clock())
}

// NewServiceWithClock returns a Service the given clock used
// to generate timestamps.
func NewServiceWithClock(store *Store, cf *httpcli.Factory, clock func() time.Time) *Service {
	svc := &Service{store: store, cf: cf, clock: clock}

	return svc
}

type Service struct {
	store *Store
	cf    *httpcli.Factory

	sourcer repos.Sourcer

	clock func() time.Time
}

// CreateCampaign creates the Campaign. When a PatchSetID is set on the
// Campaign it validates that the PatchSet contains Patches.
func (s *Service) CreateCampaign(ctx context.Context, c *campaigns.Campaign) error {
	var err error
	tr, ctx := trace.New(ctx, "Service.CreateCampaign", fmt.Sprintf("Name: %q", c.Name))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	if c.Name == "" {
		return ErrCampaignNameBlank
	}

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return err
	}
	defer tx.Done(&err)

	if c.PatchSetID != 0 {
		_, err = tx.GetCampaign(ctx, GetCampaignOpts{PatchSetID: c.PatchSetID})
		if err != nil && err != ErrNoResults {
			return err
		}
		if err != ErrNoResults {
			err = ErrPatchSetDuplicate
			return err
		}
	}

	c.CreatedAt = s.clock()
	c.UpdatedAt = c.CreatedAt

	err = tx.CreateCampaign(ctx, c)
	if err != nil {
		return err
	}

	if c.PatchSetID == 0 {
		return nil
	}
	err = validateCampaignBranch(c.Branch)
	if err != nil {
		return err
	}
	// Validate we don't have an empty patchset.
	var patchCount int64
	patchCount, err = tx.CountPatches(ctx, CountPatchesOpts{PatchSetID: c.PatchSetID, OnlyWithDiff: true, OnlyUnpublishedInCampaign: c.ID})
	if err != nil {
		return err
	}
	if patchCount == 0 {
		err = ErrNoPatches
		return err
	}

	return nil
}

// CreateCampaignSpec creates the CampaignSpec.
func (s *Service) CreateCampaignSpec(
	ctx context.Context,
	c *campaigns.CampaignSpec,
	changesetSpecRandIDs []string,
) (err error) {
	tr, ctx := trace.New(ctx, "Service.CreateCampaignSpec", fmt.Sprintf("User %d", c.UserID))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	opts := ListChangesetSpecsOpts{Limit: -1, RandIDs: changesetSpecRandIDs}
	cs, _, err := s.store.ListChangesetSpecs(ctx, opts)
	if err != nil {
		return err
	}

	repoIDs := make([]api.RepoID, 0, len(cs))
	for _, c := range cs {
		repoIDs = append(repoIDs, c.RepoID)
	}

	accessibleReposByID, err := accessibleRepos(ctx, repoIDs)
	if err != nil {
		return err
	}

	byRandID := make(map[string]*campaigns.ChangesetSpec, len(cs))
	for _, changesetSpec := range cs {
		// 🚨 SECURITY: We return an error if the user doesn't have access to one
		// of the repositories associated with a ChangesetSpec.
		if _, ok := accessibleReposByID[changesetSpec.RepoID]; !ok {
			return &db.RepoNotFoundErr{ID: changesetSpec.RepoID}
		}
		byRandID[changesetSpec.RandID] = changesetSpec
	}

	// Check if a changesetSpec was not found
	for _, randID := range changesetSpecRandIDs {
		if _, ok := byRandID[randID]; !ok {
			return &changesetSpecNotFoundErr{RandID: randID}
		}
	}

	// TODO: Handle YAML
	if err := json.Unmarshal([]byte(c.RawSpec), &c.Spec); err != nil {
		return err
	}
	// TODO: Validate that c.Spec is valid

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return err
	}
	defer tx.Done(&err)

	if err := tx.CreateCampaignSpec(ctx, c); err != nil {
		return err
	}

	for _, changesetSpec := range cs {
		changesetSpec.CampaignSpecID = c.ID

		if err := tx.UpdateChangesetSpec(ctx, changesetSpec); err != nil {
			return err
		}
	}

	return nil
}

// CreateChangesetSpec creates the ChangesetSpec.
func (s *Service) CreateChangesetSpec(ctx context.Context, c *campaigns.ChangesetSpec) (err error) {
	tr, ctx := trace.New(ctx, "Service.CreateChangesetSpec", fmt.Sprintf("User %d", c.UserID))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	if err := json.Unmarshal([]byte(c.RawSpec), &c.Spec); err != nil {
		return err
	}

	// TODO: Validate that c.Spec is valid

	c.RepoID, err = graphqlbackend.UnmarshalRepositoryID(c.Spec.RepoID)
	if err != nil {
		return err
	}

	// 🚨 SECURITY: We use db.Repos.Get to check whether the user has access to
	// the repository or not.
	if _, err = db.Repos.Get(ctx, c.RepoID); err != nil {
		return err
	}

	return s.store.CreateChangesetSpec(ctx, c)
}

// changesetSpecNotFoundErr is returned by CreateCampaignSpec if a
// ChangesetSpec with the given RandID doesn't exist.
// It fulfills the interface required by errcode.IsNotFound.
type changesetSpecNotFoundErr struct {
	RandID string
}

func (e *changesetSpecNotFoundErr) Error() string {
	if e.RandID != "" {
		return fmt.Sprintf("changesetSpec not found: id=%s", e.RandID)
	}
	return "changesetSpec not found"
}

func (e *changesetSpecNotFoundErr) NotFound() bool { return true }

type ApplyCampaignOpts struct {
	CampaignSpecRandID string
	EnsureCampaignID   int64
}

func (o ApplyCampaignOpts) String() string {
	return fmt.Sprintf(
		"CampaignSpec %s, EnsureCampaignID %d",
		o.CampaignSpecRandID,
		o.EnsureCampaignID,
	)
}

// ApplyCampaign creates the CampaignSpec.
func (s *Service) ApplyCampaign(ctx context.Context, opts ApplyCampaignOpts) (campaign *campaigns.Campaign, err error) {
	tr, ctx := trace.New(ctx, "Service.ApplyCampaign", opts.String())
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Done(&err)

	campaignSpec, err := tx.GetCampaignSpec(ctx, GetCampaignSpecOpts{
		RandID: opts.CampaignSpecRandID,
	})
	if err != nil {
		return nil, err
	}

	getOpts := GetCampaignOpts{
		CampaignSpecName: campaignSpec.Spec.Name,
		NamespaceUserID:  campaignSpec.NamespaceUserID,
		NamespaceOrgID:   campaignSpec.NamespaceOrgID,
	}

	campaign, err = tx.GetCampaign(ctx, getOpts)
	if err != nil {
		if err != ErrNoResults {
			return nil, err
		}
		err = nil
	}
	if campaign == nil {
		campaign = &campaigns.Campaign{}
	}

	if opts.EnsureCampaignID != 0 && campaign.ID != opts.EnsureCampaignID {
		return nil, ErrEnsureCampaignFailed
	}

	if campaign.CampaignSpecID == campaignSpec.ID {
		return campaign, nil
	}

	campaign.CampaignSpecID = campaignSpec.ID

	// Do we still need AuthorID on Campaign?
	campaign.AuthorID = campaignSpec.UserID

	// TODO Do we need these fields on Campaign or is it enough that
	// we have them on CampaignSpec?
	campaign.NamespaceOrgID = campaignSpec.NamespaceOrgID
	campaign.NamespaceUserID = campaignSpec.NamespaceUserID
	campaign.Branch = campaignSpec.Spec.ChangesetTemplate.Branch
	campaign.Name = campaignSpec.Spec.Name
	campaign.Description = campaignSpec.Spec.Description

	if campaign.ID == 0 {
		return campaign, tx.CreateCampaign(ctx, campaign)
	}

	return campaign, tx.UpdateCampaign(ctx, campaign)
}

// ErrEnsureCampaignFailed is returned by ApplyCampaign when a ensureCampaignID
// is provided but a campaign with the name specified the campaignSpec exists
// in the given namespace but has a different ID.
var ErrEnsureCampaignFailed = errors.New("a campaign in the given namespace and with the given name exists but does not match the given ID")

// ErrNoPatches is returned by CreateCampaign or UpdateCampaign if a
// PatchSetID was specified but the PatchSet does not have any
// (finished) Patches.
var ErrNoPatches = errors.New("cannot create or update a Campaign without any changesets")

// ErrCloseProcessingCampaign is returned by CloseCampaign if the Campaign has
// been published at the time of closing but its ChangesetJobs have not
// finished execution.
var ErrCloseProcessingCampaign = errors.New("cannot close a Campaign while changesets are being created on codehosts")

// ErrUnsupportedCodehost is returned by EnqueueChangesetJobForPatch if the target repo of a patch is an unsupported repo.
var ErrUnsupportedCodehost = errors.New("cannot publish patch for unsupported codehost")

// CloseCampaign closes the Campaign with the given ID if it has not been closed yet.
func (s *Service) CloseCampaign(ctx context.Context, id int64, closeChangesets bool) (campaign *campaigns.Campaign, err error) {
	traceTitle := fmt.Sprintf("campaign: %d, closeChangesets: %t", id, closeChangesets)
	tr, ctx := trace.New(ctx, "service.CloseCampaign", traceTitle)
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	transaction := func() (err error) {
		tx, err := s.store.Transact(ctx)
		if err != nil {
			return err
		}
		defer tx.Done(&err)

		campaign, err = tx.GetCampaign(ctx, GetCampaignOpts{ID: id})
		if err != nil {
			return errors.Wrap(err, "getting campaign")
		}

		if err := backend.CheckSiteAdminOrSameUser(ctx, campaign.AuthorID); err != nil {
			return err
		}

		processing, err := campaignIsProcessing(ctx, tx, id)
		if err != nil {
			return err
		}
		if processing {
			err = ErrCloseProcessingCampaign
			return err
		}

		if !campaign.ClosedAt.IsZero() {
			return nil
		}

		campaign.ClosedAt = time.Now().UTC()

		return tx.UpdateCampaign(ctx, campaign)
	}

	err = transaction()
	if err != nil {
		return nil, err
	}

	if closeChangesets {
		go func() {
			ctx := trace.ContextWithTrace(context.Background(), tr)

			cs, _, err := s.store.ListChangesets(ctx, ListChangesetsOpts{
				CampaignID: campaign.ID,
				Limit:      -1,
			})
			if err != nil {
				log15.Error("ListChangesets", "err", err)
				return
			}

			// Close only the changesets that are open
			err = s.CloseOpenChangesets(ctx, cs)
			if err != nil {
				log15.Error("CloseCampaignChangesets", "err", err)
			}
		}()
	}

	return campaign, nil
}

// ErrDeleteProcessingCampaign is returned by DeleteCampaign if the Campaign
// has been published at the time of deletion but its ChangesetJobs have not
// finished execution.
var ErrDeleteProcessingCampaign = errors.New("cannot delete a Campaign while changesets are being created on codehosts")

// DeleteCampaign deletes the Campaign with the given ID if it hasn't been
// deleted yet. If closeChangesets is true, the changesets associated with the
// Campaign will be closed on the codehosts.
func (s *Service) DeleteCampaign(ctx context.Context, id int64) (err error) {
	traceTitle := fmt.Sprintf("campaign: %d", id)
	tr, ctx := trace.New(ctx, "service.DeleteCampaign", traceTitle)
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	campaign, err := s.store.GetCampaign(ctx, GetCampaignOpts{ID: id})
	if err != nil {
		return err
	}

	if err := backend.CheckSiteAdminOrSameUser(ctx, campaign.AuthorID); err != nil {
		return err
	}

	transaction := func() (err error) {
		tx, err := s.store.Transact(ctx)
		if err != nil {
			return err
		}
		defer tx.Done(&err)

		processing, err := campaignIsProcessing(ctx, tx, id)
		if err != nil {
			return err
		}
		if processing {
			return ErrDeleteProcessingCampaign
		}

		return tx.DeleteCampaign(ctx, id)
	}

	return transaction()
}

// CloseOpenChangesets closes the given Changesets on their respective codehosts and syncs them.
func (s *Service) CloseOpenChangesets(ctx context.Context, cs campaigns.Changesets) (err error) {
	cs = cs.Filter(func(c *campaigns.Changeset) bool {
		return c.ExternalState == campaigns.ChangesetStateOpen
	})

	if len(cs) == 0 {
		return nil
	}

	accessibleReposByID, err := accessibleRepos(ctx, cs.RepoIDs())
	if err != nil {
		return err
	}

	reposStore := repos.NewDBStore(s.store.DB(), sql.TxOptions{})
	bySource, err := groupChangesetsBySource(ctx, reposStore, s.cf, s.sourcer, cs...)
	if err != nil {
		return err
	}

	errs := &multierror.Error{}
	for _, group := range bySource {
		for _, c := range group.Changesets {
			if _, ok := accessibleReposByID[c.RepoID]; !ok {
				continue
			}

			if err := group.CloseChangeset(ctx, c); err != nil {
				errs = multierror.Append(errs, err)
			}
		}
	}

	if len(errs.Errors) != 0 {
		return errs
	}

	// Here we need to sync the just-closed changesets (even though
	// CloseChangesets updates the given Changesets too), because closing a
	// Changeset often produces a ChangesetEvent on the codehost and if we were
	// to close the Changesets and not update the events (which is what
	// syncChangesetsWithSources does) our burndown chart will be outdated
	// until the next run of campaigns.Syncer.
	return syncChangesetsWithSources(ctx, s.store, bySource)
}

// EnqueueChangesetSync loads the given changeset from the database, checks
// whether the actor in the context has permission to enqueue a sync and then
// enqueues a sync by calling the repoupdater client.
func (s *Service) EnqueueChangesetSync(ctx context.Context, id int64) (err error) {
	traceTitle := fmt.Sprintf("changeset: %d", id)
	tr, ctx := trace.New(ctx, "service.EnqueueChangesetSync", traceTitle)
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	// Check for existence of changeset so we don't swallow that error.
	changeset, err := s.store.GetChangeset(ctx, GetChangesetOpts{ID: id})
	if err != nil {
		return err
	}

	// 🚨 SECURITY: We use db.Repos.Get to check whether the user has access to
	// the repository or not.
	if _, err = db.Repos.Get(ctx, changeset.RepoID); err != nil {
		return err
	}

	campaigns, _, err := s.store.ListCampaigns(ctx, ListCampaignsOpts{ChangesetID: id})
	if err != nil {
		return err
	}

	// Check whether the user has admin rights for one of the campaigns.
	var (
		authErr        error
		hasAdminRights bool
	)

	for _, c := range campaigns {
		err := backend.CheckSiteAdminOrSameUser(ctx, c.AuthorID)
		if err != nil {
			authErr = err
		} else {
			hasAdminRights = true
			break
		}
	}

	if !hasAdminRights {
		return authErr
	}

	if err := repoupdater.DefaultClient.EnqueueChangesetSync(ctx, []int64{id}); err != nil {
		return err
	}

	return nil
}

// GetCampaignStatus returns the BackgroundProcessStatus for the given campaign.
func (s *Service) GetCampaignStatus(ctx context.Context, c *campaigns.Campaign) (status *campaigns.BackgroundProcessStatus, err error) {
	traceTitle := fmt.Sprintf("campaign: %d", c.ID)
	tr, ctx := trace.New(ctx, "service.GetCampaignStatus", traceTitle)
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	canAdmin, err := hasCampaignAdminPermissions(ctx, c)
	if err != nil {
		return nil, err
	}

	if !canAdmin {
		// If the user doesn't have admin permissions for this campaign, we
		// don't need to filter out specific errors, but can simply exclude
		// _all_ errors.
		return s.store.GetCampaignStatus(ctx, GetCampaignStatusOpts{
			ID:            c.ID,
			ExcludeErrors: true,
		})
	}

	// We need to filter out error messages the user is not allowed to see,
	// because they don't have permissions to access the repository associated
	// with a given patch/changesetJob.

	// First we load the repo IDs of the failed changesetJobs
	repoIDs, err := s.store.GetRepoIDsForFailedChangesetJobs(ctx, c.ID)
	if err != nil {
		return nil, err
	}

	// 🚨 SECURITY: We use db.Repos.GetByIDs to filter out repositories the
	// user doesn't have access to.
	accessibleRepos, err := db.Repos.GetByIDs(ctx, repoIDs...)
	if err != nil {
		return nil, err
	}

	accessibleRepoIDs := make(map[api.RepoID]struct{}, len(accessibleRepos))
	for _, r := range accessibleRepos {
		accessibleRepoIDs[r.ID] = struct{}{}
	}

	// We now check which repositories in `repoIDs` are not in `accessibleRepoIDs`.
	// We have to filter the error messages associated with those out.
	excludedRepos := make([]api.RepoID, 0, len(accessibleRepoIDs))
	for _, id := range repoIDs {
		if _, ok := accessibleRepoIDs[id]; !ok {
			excludedRepos = append(excludedRepos, id)
		}
	}

	return s.store.GetCampaignStatus(ctx, GetCampaignStatusOpts{
		ID:                   c.ID,
		ExcludeErrorsInRepos: excludedRepos,
	})
}

// ErrCampaignNameBlank is returned by CreateCampaign or UpdateCampaign if the
// specified Campaign name is blank.
var ErrCampaignNameBlank = errors.New("Campaign title cannot be blank")

// ErrCampaignBranchBlank is returned by CreateCampaign or UpdateCampaign if the specified Campaign's
// branch is blank. This is only enforced when creating published campaigns with a patch set.
var ErrCampaignBranchBlank = errors.New("Campaign branch cannot be blank")

// ErrCampaignBranchInvalid is returned by CreateCampaign or UpdateCampaign if the specified Campaign's
// branch is invalid. This is only enforced when creating published campaigns with a patch set.
var ErrCampaignBranchInvalid = errors.New("Campaign branch is invalid")

// ErrPatchSetDuplicate is return by CreateCampaign or UpdateCampaign if the
// specified patch set is already attached to another campaign.
var ErrPatchSetDuplicate = errors.New("Campaign cannot use the same patch set as another campaign")

func validateCampaignBranch(branch string) error {
	if branch == "" {
		return ErrCampaignBranchBlank
	}
	if !git.ValidateBranchName(branch) {
		return ErrCampaignBranchInvalid
	}
	return nil
}

func campaignIsProcessing(ctx context.Context, store *Store, campaign int64) (bool, error) {
	status, err := store.GetCampaignStatus(ctx, GetCampaignStatusOpts{ID: campaign})
	if err != nil {
		return false, err
	}
	return status.Processing(), nil
}

// hasCampaignAdminPermissions returns true when the actor in the given context
// is either a site-admin or the author of the given campaign.
func hasCampaignAdminPermissions(ctx context.Context, c *campaigns.Campaign) (bool, error) {
	// 🚨 SECURITY: Only site admins or the authors of a campaign have campaign admin rights.
	if err := backend.CheckSiteAdminOrSameUser(ctx, c.AuthorID); err != nil {
		if _, ok := err.(*backend.InsufficientAuthorizationError); ok {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

// accessibleRepos collects the RepoIDs of the changesets and returns a set of
// the api.RepoID for which the subset of repositories for which the actor in
// ctx has read permissions.
func accessibleRepos(ctx context.Context, ids []api.RepoID) (map[api.RepoID]*types.Repo, error) {
	// 🚨 SECURITY: We use db.Repos.GetByIDs to filter out repositories the
	// user doesn't have access to.
	accessibleRepos, err := db.Repos.GetByIDs(ctx, ids...)
	if err != nil {
		return nil, err
	}

	accessibleRepoIDs := make(map[api.RepoID]*types.Repo, len(accessibleRepos))
	for _, r := range accessibleRepos {
		accessibleRepoIDs[r.ID] = r
	}

	return accessibleRepoIDs, nil
}
