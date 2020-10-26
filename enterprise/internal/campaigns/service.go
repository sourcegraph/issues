package campaigns

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

// NewService returns a Service.
func NewService(store *Store, cf *httpcli.Factory) *Service {
	return NewServiceWithClock(store, cf, store.Clock())
}

// NewServiceWithClock returns a Service the given clock used
// to generate timestamps.
func NewServiceWithClock(store *Store, cf *httpcli.Factory, clock func() time.Time) *Service {
	svc := &Service{store: store, sourcer: repos.NewSourcer(cf), clock: clock}

	return svc
}

type Service struct {
	store *Store

	sourcer repos.Sourcer

	clock func() time.Time
}

// WithStore returns a copy of the Service with its store attribute set to the
// given Store.
func (s *Service) WithStore(store *Store) *Service {
	return &Service{store: store, sourcer: s.sourcer, clock: s.clock}
}

type CreateCampaignSpecOpts struct {
	RawSpec string

	NamespaceUserID int32
	NamespaceOrgID  int32

	ChangesetSpecRandIDs []string
}

// CreateCampaignSpec creates the CampaignSpec.
func (s *Service) CreateCampaignSpec(ctx context.Context, opts CreateCampaignSpecOpts) (spec *campaigns.CampaignSpec, err error) {
	actor := actor.FromContext(ctx)
	tr, ctx := trace.New(ctx, "Service.CreateCampaignSpec", fmt.Sprintf("Actor %s", actor))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	spec, err = campaigns.NewCampaignSpecFromRaw(opts.RawSpec)
	if err != nil {
		return nil, err
	}

	// Check whether the current user has access to either one of the namespaces.
	err = checkNamespaceAccess(ctx, opts.NamespaceUserID, opts.NamespaceOrgID)
	if err != nil {
		return nil, err
	}
	spec.NamespaceOrgID = opts.NamespaceOrgID
	spec.NamespaceUserID = opts.NamespaceUserID
	spec.UserID = actor.UID

	if len(opts.ChangesetSpecRandIDs) == 0 {
		return spec, s.store.CreateCampaignSpec(ctx, spec)
	}

	listOpts := ListChangesetSpecsOpts{RandIDs: opts.ChangesetSpecRandIDs}
	cs, _, err := s.store.ListChangesetSpecs(ctx, listOpts)
	if err != nil {
		return nil, err
	}

	// 🚨 SECURITY: db.Repos.GetRepoIDsSet uses the authzFilter under the hood and
	// filters out repositories that the user doesn't have access to.
	accessibleReposByID, err := db.Repos.GetReposSetByIDs(ctx, cs.RepoIDs()...)
	if err != nil {
		return nil, err
	}

	byRandID := make(map[string]*campaigns.ChangesetSpec, len(cs))
	for _, changesetSpec := range cs {
		// 🚨 SECURITY: We return an error if the user doesn't have access to one
		// of the repositories associated with a ChangesetSpec.
		if _, ok := accessibleReposByID[changesetSpec.RepoID]; !ok {
			return nil, &db.RepoNotFoundErr{ID: changesetSpec.RepoID}
		}
		byRandID[changesetSpec.RandID] = changesetSpec
	}

	// Check if a changesetSpec was not found
	for _, randID := range opts.ChangesetSpecRandIDs {
		if _, ok := byRandID[randID]; !ok {
			return nil, &changesetSpecNotFoundErr{RandID: randID}
		}
	}

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = tx.Done(err) }()

	if err := tx.CreateCampaignSpec(ctx, spec); err != nil {
		return nil, err
	}

	for _, changesetSpec := range cs {
		changesetSpec.CampaignSpecID = spec.ID

		if err := tx.UpdateChangesetSpec(ctx, changesetSpec); err != nil {
			return nil, err
		}
	}

	return spec, nil
}

// CreateChangesetSpec validates the given raw spec input and creates the ChangesetSpec.
func (s *Service) CreateChangesetSpec(ctx context.Context, rawSpec string, userID int32) (spec *campaigns.ChangesetSpec, err error) {
	tr, ctx := trace.New(ctx, "Service.CreateChangesetSpec", fmt.Sprintf("User %d", userID))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	spec, err = campaigns.NewChangesetSpecFromRaw(rawSpec)
	if err != nil {
		return nil, err
	}
	spec.UserID = userID
	spec.RepoID, err = graphqlbackend.UnmarshalRepositoryID(spec.Spec.BaseRepository)
	if err != nil {
		return nil, err
	}

	// 🚨 SECURITY: We use db.Repos.Get to check whether the user has access to
	// the repository or not.
	if _, err = db.Repos.Get(ctx, spec.RepoID); err != nil {
		return nil, err
	}

	return spec, s.store.CreateChangesetSpec(ctx, spec)
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

// GetCampaignMatchingCampaignSpec returns the Campaign that the CampaignSpec
// applies to, if that Campaign already exists.
// If it doesn't exist yet, both return values are nil.
// It accepts a *Store so that it can be used inside a transaction.
func (s *Service) GetCampaignMatchingCampaignSpec(ctx context.Context, tx *Store, spec *campaigns.CampaignSpec) (*campaigns.Campaign, error) {
	opts := GetCampaignOpts{
		Name:            spec.Spec.Name,
		NamespaceUserID: spec.NamespaceUserID,
		NamespaceOrgID:  spec.NamespaceOrgID,
	}

	campaign, err := tx.GetCampaign(ctx, opts)
	if err != nil {
		if err != ErrNoResults {
			return nil, err
		}
		err = nil
	}
	return campaign, err
}

type MoveCampaignOpts struct {
	CampaignID int64

	NewName string

	NewNamespaceUserID int32
	NewNamespaceOrgID  int32
}

func (o MoveCampaignOpts) String() string {
	return fmt.Sprintf(
		"CampaignID %d, NewName %q, NewNamespaceUserID %d, NewNamespaceOrgID %d",
		o.CampaignID,
		o.NewName,
		o.NewNamespaceUserID,
		o.NewNamespaceOrgID,
	)
}

// MoveCampaign moves the campaign from one namespace to another and/or renames
// the campaign.
func (s *Service) MoveCampaign(ctx context.Context, opts MoveCampaignOpts) (campaign *campaigns.Campaign, err error) {
	tr, ctx := trace.New(ctx, "Service.MoveCampaign", opts.String())
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = tx.Done(err) }()

	campaign, err = tx.GetCampaign(ctx, GetCampaignOpts{ID: opts.CampaignID})
	if err != nil {
		return nil, err
	}

	// 🚨 SECURITY: Only the Author of the campaign can move it.
	if err := backend.CheckSiteAdminOrSameUser(ctx, campaign.InitialApplierID); err != nil {
		return nil, err
	}
	// Check if current user has access to target namespace if set.
	if opts.NewNamespaceOrgID != 0 || opts.NewNamespaceUserID != 0 {
		err = checkNamespaceAccess(ctx, opts.NewNamespaceUserID, opts.NewNamespaceOrgID)
		if err != nil {
			return nil, err
		}
	}

	if opts.NewNamespaceOrgID != 0 {
		campaign.NamespaceOrgID = opts.NewNamespaceOrgID
		campaign.NamespaceUserID = 0
	} else if opts.NewNamespaceUserID != 0 {
		campaign.NamespaceUserID = opts.NewNamespaceUserID
		campaign.NamespaceOrgID = 0
	}

	if opts.NewName != "" {
		campaign.Name = opts.NewName
	}

	return campaign, tx.UpdateCampaign(ctx, campaign)
}

// ErrEnsureCampaignFailed is returned by ApplyCampaign when a ensureCampaignID
// is provided but a campaign with the name specified the campaignSpec exists
// in the given namespace but has a different ID.
var ErrEnsureCampaignFailed = errors.New("a campaign in the given namespace and with the given name exists but does not match the given ID")

// CloseCampaign closes the Campaign with the given ID if it has not been closed yet.
func (s *Service) CloseCampaign(ctx context.Context, id int64, closeChangesets bool) (campaign *campaigns.Campaign, err error) {
	traceTitle := fmt.Sprintf("campaign: %d, closeChangesets: %t", id, closeChangesets)
	tr, ctx := trace.New(ctx, "service.CloseCampaign", traceTitle)
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	campaign, err = s.store.GetCampaign(ctx, GetCampaignOpts{ID: id})
	if err != nil {
		return nil, errors.Wrap(err, "getting campaign")
	}

	if campaign.Closed() {
		return campaign, nil
	}

	if err := backend.CheckSiteAdminOrSameUser(ctx, campaign.InitialApplierID); err != nil {
		return nil, err
	}

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = tx.Done(err) }()

	campaign.ClosedAt = s.clock()
	if err := tx.UpdateCampaign(ctx, campaign); err != nil {
		return nil, err
	}

	if !closeChangesets {
		return campaign, nil
	}

	// At this point we don't know which changesets have ExternalStateOpen,
	// since some might still be being processed in the background by the
	// reconciler.
	// So enqueue all, except the ones that are completed and closed/merged,
	// for closing. If after being processed they're not open, it'll be a noop.
	if err := tx.EnqueueChangesetsToClose(ctx, campaign.ID); err != nil {
		return nil, err
	}

	return campaign, nil
}

// DeleteCampaign deletes the Campaign with the given ID if it hasn't been
// deleted yet.
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

	if err := backend.CheckSiteAdminOrSameUser(ctx, campaign.InitialApplierID); err != nil {
		return err
	}

	return s.store.DeleteCampaign(ctx, id)
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
		err := backend.CheckSiteAdminOrSameUser(ctx, c.InitialApplierID)
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

// checkNamespaceAccess checks whether the current user in the ctx has access
// to either the user ID or the org ID as a namespace.
// If the userID is non-zero that will be checked. Otherwise the org ID will be
// checked.
// If the current user is an admin, true will be returned.
// Otherwise it checks whether the current user _is_ the namespace user or has
// access to the namespace org.
// If both values are zero, an error is returned.
func checkNamespaceAccess(ctx context.Context, namespaceUserID, namespaceOrgID int32) error {
	if namespaceOrgID != 0 {
		return backend.CheckOrgAccess(ctx, namespaceOrgID)
	} else if namespaceUserID != 0 {
		return backend.CheckSiteAdminOrSameUser(ctx, namespaceUserID)
	} else {
		return ErrNoNamespace
	}
}

// ErrNoNamespace is returned by checkNamespaceAccess if no valid namespace ID is given.
var ErrNoNamespace = errors.New("no namespace given")

// checkRepoSupported checks whether the given repository is supported by campaigns
// and if not it returns an error.
func checkRepoSupported(repo *types.Repo) error {
	if campaigns.IsRepoSupported(&repo.ExternalRepo) {
		return nil
	}

	return errors.Errorf(
		"External service type %s of repository %q is currently not supported for use with campaigns",
		repo.ExternalRepo.ServiceType,
		repo.Name,
	)
}
