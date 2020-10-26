package resolvers

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/externallink"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	ee "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
)

type changesetResolver struct {
	store       *ee.Store
	httpFactory *httpcli.Factory

	changeset *campaigns.Changeset

	// When repo is nil, this resolver resolves to a `HiddenExternalChangeset` in the API.
	repo         *types.Repo
	repoResolver *graphqlbackend.RepositoryResolver

	// cache changeset events as they are used more than once
	eventsOnce sync.Once
	events     ee.ChangesetEvents
	eventsErr  error

	attemptedPreloadNextSyncAt bool
	// When the next sync is scheduled
	preloadedNextSyncAt *time.Time
	nextSyncAtOnce      sync.Once
	nextSyncAt          time.Time
	nextSyncAtErr       error

	// cache the current ChangesetSpec as it's accessed by multiple methods
	specOnce sync.Once
	spec     *campaigns.ChangesetSpec
	specErr  error
}

func NewChangesetResolverWithNextSync(store *ee.Store, httpFactory *httpcli.Factory, changeset *campaigns.Changeset, repo *types.Repo, nextSyncAt *time.Time) *changesetResolver {
	r := NewChangesetResolver(store, httpFactory, changeset, repo)
	r.attemptedPreloadNextSyncAt = true
	r.preloadedNextSyncAt = nextSyncAt
	return r
}

func NewChangesetResolver(store *ee.Store, httpFactory *httpcli.Factory, changeset *campaigns.Changeset, repo *types.Repo) *changesetResolver {
	return &changesetResolver{
		store:        store,
		httpFactory:  httpFactory,
		repo:         repo,
		repoResolver: graphqlbackend.NewRepositoryResolver(repo),
		changeset:    changeset,
	}
}

const changesetIDKind = "Changeset"

func marshalChangesetID(id int64) graphql.ID {
	return relay.MarshalID(changesetIDKind, id)
}

func unmarshalChangesetID(id graphql.ID) (cid int64, err error) {
	err = relay.UnmarshalSpec(id, &cid)
	return
}

func (r *changesetResolver) ToExternalChangeset() (graphqlbackend.ExternalChangesetResolver, bool) {
	if !r.repoAccessible() {
		return nil, false
	}

	return r, true
}

func (r *changesetResolver) ToHiddenExternalChangeset() (graphqlbackend.HiddenExternalChangesetResolver, bool) {
	if r.repoAccessible() {
		return nil, false
	}

	return r, true
}

func (r *changesetResolver) repoAccessible() bool {
	// If the repository is not nil, it's accessible
	return r.repo != nil
}

func (r *changesetResolver) computeSpec(ctx context.Context) (*campaigns.ChangesetSpec, error) {
	r.specOnce.Do(func() {
		if r.changeset.CurrentSpecID == 0 {
			r.specErr = errors.New("Changeset has no ChangesetSpec")
			return
		}

		r.spec, r.specErr = r.store.GetChangesetSpecByID(ctx, r.changeset.CurrentSpecID)
	})
	return r.spec, r.specErr
}

func (r *changesetResolver) computeEvents(ctx context.Context) ([]*campaigns.ChangesetEvent, error) {
	r.eventsOnce.Do(func() {
		opts := ee.ListChangesetEventsOpts{
			ChangesetIDs: []int64{r.changeset.ID},
		}
		es, _, err := r.store.ListChangesetEvents(ctx, opts)

		r.events = es
		sort.Sort(r.events)

		r.eventsErr = err
	})
	return r.events, r.eventsErr
}

func (r *changesetResolver) computeNextSyncAt(ctx context.Context) (time.Time, error) {
	r.nextSyncAtOnce.Do(func() {
		if r.attemptedPreloadNextSyncAt {
			if r.preloadedNextSyncAt != nil {
				r.nextSyncAt = *r.preloadedNextSyncAt
			}
			return
		}
		syncData, err := r.store.ListChangesetSyncData(ctx, ee.ListChangesetSyncDataOpts{ChangesetIDs: []int64{r.changeset.ID}})
		if err != nil {
			r.nextSyncAtErr = err
			return
		}
		for _, d := range syncData {
			if d.ChangesetID == r.changeset.ID {
				r.nextSyncAt = ee.NextSync(r.store.Clock(), d)
				return
			}
		}
	})
	return r.nextSyncAt, r.nextSyncAtErr
}

func (r *changesetResolver) ID() graphql.ID {
	return marshalChangesetID(r.changeset.ID)
}

func (r *changesetResolver) ExternalID() *string {
	if r.changeset.PublicationState.Unpublished() {
		return nil
	}
	return &r.changeset.ExternalID
}

func (r *changesetResolver) Repository(ctx context.Context) *graphqlbackend.RepositoryResolver {
	return r.repoResolver
}

func (r *changesetResolver) Campaigns(ctx context.Context, args *graphqlbackend.ListCampaignsArgs) (graphqlbackend.CampaignsConnectionResolver, error) {
	opts := ee.ListCampaignsOpts{
		ChangesetID: r.changeset.ID,
	}

	state, err := parseCampaignState(args.State)
	if err != nil {
		return nil, err
	}
	opts.State = state
	if err := validateFirstParamDefaults(args.First); err != nil {
		return nil, err
	}
	opts.Limit = int(args.First)
	if args.After != nil {
		cursor, err := strconv.ParseInt(*args.After, 10, 32)
		if err != nil {
			return nil, err
		}
		opts.Cursor = cursor
	}

	authErr := backend.CheckCurrentUserIsSiteAdmin(ctx)
	if authErr != nil && authErr != backend.ErrMustBeSiteAdmin {
		return nil, err
	}
	isSiteAdmin := authErr != backend.ErrMustBeSiteAdmin
	if !isSiteAdmin {
		if args.ViewerCanAdminister != nil && *args.ViewerCanAdminister {
			actor := actor.FromContext(ctx)
			opts.InitialApplierID = actor.UID
		}
	}

	return &campaignsConnectionResolver{store: r.store, httpFactory: r.httpFactory, opts: opts}, nil
}

func (r *changesetResolver) CreatedAt() graphqlbackend.DateTime {
	return graphqlbackend.DateTime{Time: r.changeset.CreatedAt}
}

func (r *changesetResolver) UpdatedAt() graphqlbackend.DateTime {
	return graphqlbackend.DateTime{Time: r.changeset.UpdatedAt}
}

func (r *changesetResolver) NextSyncAt(ctx context.Context) (*graphqlbackend.DateTime, error) {
	nextSyncAt, err := r.computeNextSyncAt(ctx)
	if err != nil {
		return nil, err
	}
	if nextSyncAt.IsZero() {
		return nil, nil
	}
	return &graphqlbackend.DateTime{Time: nextSyncAt}, nil
}

func (r *changesetResolver) Title(ctx context.Context) (*string, error) {
	if r.changeset.PublishedAndSynced() {
		t, err := r.changeset.Title()
		if err != nil {
			return nil, err
		}
		return &t, nil
	}

	if r.changeset.Unpublished() {
		desc, err := r.getBranchSpecDescription(ctx)
		if err != nil {
			return nil, err
		}

		return &desc.Title, nil
	}

	// Marked as published, but unsynced
	return nil, nil
}

func (r *changesetResolver) Body(ctx context.Context) (*string, error) {
	if r.changeset.PublishedAndSynced() {
		b, err := r.changeset.Body()
		if err != nil {
			return nil, err
		}
		return &b, nil
	}

	if r.changeset.Unpublished() {
		desc, err := r.getBranchSpecDescription(ctx)
		if err != nil {
			return nil, err
		}

		return &desc.Body, nil
	}

	// Marked as published, but unsynced
	return nil, nil
}

func (r *changesetResolver) getBranchSpecDescription(ctx context.Context) (*campaigns.ChangesetSpecDescription, error) {
	spec, err := r.computeSpec(ctx)
	if err != nil {
		return nil, err
	}

	if spec.Spec.IsImportingExisting() {
		return nil, errors.New("ChangesetSpec imports a changeset")
	}

	return spec.Spec, nil
}

func (r *changesetResolver) PublicationState() campaigns.ChangesetPublicationState {
	return r.changeset.PublicationState
}

func (r *changesetResolver) ReconcilerState() campaigns.ReconcilerState {
	return r.changeset.ReconcilerState
}

func (r *changesetResolver) ExternalState() *campaigns.ChangesetExternalState {
	if !r.changeset.PublishedAndSynced() {
		return nil
	}
	return &r.changeset.ExternalState
}

func (r *changesetResolver) ExternalURL() (*externallink.Resolver, error) {
	if !r.changeset.PublishedAndSynced() {
		return nil, nil
	}
	url, err := r.changeset.URL()
	if err != nil {
		return nil, err
	}
	return externallink.NewResolver(url, r.changeset.ExternalServiceType), nil
}

func (r *changesetResolver) ReviewState(ctx context.Context) *campaigns.ChangesetReviewState {
	if !r.changeset.PublishedAndSynced() {
		return nil
	}
	return &r.changeset.ExternalReviewState
}

func (r *changesetResolver) CheckState() *campaigns.ChangesetCheckState {
	if !r.changeset.PublishedAndSynced() {
		return nil
	}

	state := r.changeset.ExternalCheckState
	if state == campaigns.ChangesetCheckStateUnknown {
		return nil
	}

	return &state
}

func (r *changesetResolver) Error() *string { return r.changeset.FailureMessage }

func (r *changesetResolver) CurrentSpec(ctx context.Context) (graphqlbackend.VisibleChangesetSpecResolver, error) {
	if r.changeset.CurrentSpecID == 0 {
		return nil, nil
	}

	spec, err := r.computeSpec(ctx)
	if err != nil {
		return nil, err
	}

	return &changesetSpecResolver{
		store:                r.store,
		httpFactory:          r.httpFactory,
		changesetSpec:        spec,
		preloadedRepo:        r.repo,
		attemptedPreloadRepo: true,
	}, nil
}

func (r *changesetResolver) Labels(ctx context.Context) ([]graphqlbackend.ChangesetLabelResolver, error) {
	if !r.changeset.PublishedAndSynced() {
		return []graphqlbackend.ChangesetLabelResolver{}, nil
	}

	// Not every code host supports labels on changesets so don't make a DB call unless we need to.
	if ok := r.changeset.SupportsLabels(); !ok {
		return []graphqlbackend.ChangesetLabelResolver{}, nil
	}

	es, err := r.computeEvents(ctx)
	if err != nil {
		return nil, err
	}

	// We use changeset labels as the source of truth as they can be renamed
	// or removed but we'll also take into account any changeset events that
	// have happened since the last sync in order to reflect changes that
	// have come in via webhooks
	labels := ee.ComputeLabels(r.changeset, es)
	resolvers := make([]graphqlbackend.ChangesetLabelResolver, 0, len(labels))
	for _, l := range labels {
		resolvers = append(resolvers, &changesetLabelResolver{label: l})
	}
	return resolvers, nil
}

func (r *changesetResolver) Events(ctx context.Context, args *graphqlbackend.ChangesetEventsConnectionArgs) (graphqlbackend.ChangesetEventsConnectionResolver, error) {
	if err := validateFirstParamDefaults(args.First); err != nil {
		return nil, err
	}
	var cursor int64
	if args.After != nil {
		var err error
		cursor, err = strconv.ParseInt(*args.After, 10, 32)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse after cursor")
		}
	}
	// TODO: We already need to fetch all events for ReviewState and Labels
	// perhaps we can use the cached data here
	return &changesetEventsConnectionResolver{
		store:             r.store,
		changesetResolver: r,
		first:             int(args.First),
		cursor:            cursor,
	}, nil
}

func (r *changesetResolver) Diff(ctx context.Context) (graphqlbackend.RepositoryComparisonInterface, error) {
	if r.changeset.Unpublished() {
		desc, err := r.getBranchSpecDescription(ctx)
		if err != nil {
			return nil, err
		}

		diff, err := desc.Diff()
		if err != nil {
			return nil, errors.New("ChangesetSpec has no diff")
		}

		return graphqlbackend.NewPreviewRepositoryComparisonResolver(
			ctx,
			r.repoResolver,
			desc.BaseRev,
			diff,
		)
	}

	// If it's published but not synced, we don't have enough information to
	// return a diff yet.
	if !r.changeset.PublishedAndSynced() {
		return nil, nil
	}

	if !r.changeset.HasDiff() {
		return nil, nil
	}

	base, err := r.changeset.BaseRefOid()
	if err != nil {
		return nil, err
	}
	if base == "" {
		// Fallback to the ref if we can't get the OID
		base, err = r.changeset.BaseRef()
		if err != nil {
			return nil, err
		}
	}

	head, err := r.changeset.HeadRefOid()
	if err != nil {
		return nil, err
	}
	if head == "" {
		// Fallback to the ref if we can't get the OID
		head, err = r.changeset.HeadRef()
		if err != nil {
			return nil, err
		}
	}

	return graphqlbackend.NewRepositoryComparison(ctx, r.repoResolver, &graphqlbackend.RepositoryComparisonInput{
		Base:         &base,
		Head:         &head,
		FetchMissing: true,
	})
}

func (r *changesetResolver) DiffStat(ctx context.Context) (*graphqlbackend.DiffStat, error) {
	if stat := r.changeset.DiffStat(); stat != nil {
		return graphqlbackend.NewDiffStat(*stat), nil
	}
	return nil, nil
}

type changesetLabelResolver struct {
	label campaigns.ChangesetLabel
}

func (r *changesetLabelResolver) Text() string {
	return r.label.Name
}

func (r *changesetLabelResolver) Color() string {
	return r.label.Color
}

func (r *changesetLabelResolver) Description() *string {
	if r.label.Description == "" {
		return nil
	}
	return &r.label.Description
}
