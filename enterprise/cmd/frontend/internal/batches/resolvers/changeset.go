package resolvers

import (
	"context"
	"fmt"
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
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/state"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/syncer"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types/scheduler/config"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

type changesetResolver struct {
	store *store.Store

	changeset *btypes.Changeset

	// When repo is nil, this resolver resolves to a `HiddenExternalChangeset` in the API.
	repo         *types.Repo
	repoResolver *graphqlbackend.RepositoryResolver

	attemptedPreloadNextSyncAt bool
	// When the next sync is scheduled
	preloadedNextSyncAt time.Time
	nextSyncAtOnce      sync.Once
	nextSyncAt          time.Time
	nextSyncAtErr       error

	// cache the current ChangesetSpec as it's accessed by multiple methods
	specOnce sync.Once
	spec     *btypes.ChangesetSpec
	specErr  error
}

func NewChangesetResolverWithNextSync(store *store.Store, changeset *btypes.Changeset, repo *types.Repo, nextSyncAt time.Time) *changesetResolver {
	r := NewChangesetResolver(store, changeset, repo)
	r.attemptedPreloadNextSyncAt = true
	r.preloadedNextSyncAt = nextSyncAt
	return r
}

func NewChangesetResolver(store *store.Store, changeset *btypes.Changeset, repo *types.Repo) *changesetResolver {
	return &changesetResolver{
		store:        store,
		repo:         repo,
		repoResolver: graphqlbackend.NewRepositoryResolver(store.DB(), repo),
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

func (r *changesetResolver) computeSpec(ctx context.Context) (*btypes.ChangesetSpec, error) {
	r.specOnce.Do(func() {
		if r.changeset.CurrentSpecID == 0 {
			r.specErr = errors.New("Changeset has no ChangesetSpec")
			return
		}

		r.spec, r.specErr = r.store.GetChangesetSpecByID(ctx, r.changeset.CurrentSpecID)
	})
	return r.spec, r.specErr
}

func (r *changesetResolver) computeNextSyncAt(ctx context.Context) (time.Time, error) {
	r.nextSyncAtOnce.Do(func() {
		if r.attemptedPreloadNextSyncAt {
			r.nextSyncAt = r.preloadedNextSyncAt
			return
		}
		syncData, err := r.store.ListChangesetSyncData(ctx, store.ListChangesetSyncDataOpts{ChangesetIDs: []int64{r.changeset.ID}})
		if err != nil {
			r.nextSyncAtErr = err
			return
		}
		for _, d := range syncData {
			if d.ChangesetID == r.changeset.ID {
				r.nextSyncAt = syncer.NextSync(r.store.Clock(), d)
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
	if r.changeset.ExternalID == "" {
		return nil
	}
	return &r.changeset.ExternalID
}

func (r *changesetResolver) Repository(ctx context.Context) *graphqlbackend.RepositoryResolver {
	return r.repoResolver
}

// TODO(campaigns-deprecation): This should be removed once we remove campaigns completely.
func (r *changesetResolver) Campaigns(ctx context.Context, args *graphqlbackend.ListBatchChangesArgs) (graphqlbackend.BatchChangesConnectionResolver, error) {
	return r.BatchChanges(ctx, args)
}

func (r *changesetResolver) BatchChanges(ctx context.Context, args *graphqlbackend.ListBatchChangesArgs) (graphqlbackend.BatchChangesConnectionResolver, error) {
	opts := store.ListBatchChangesOpts{
		ChangesetID: r.changeset.ID,
	}

	state, err := parseBatchChangeState(args.State)
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

	return &batchChangesConnectionResolver{store: r.store, opts: opts}, nil
}

func (r *changesetResolver) CreatedAt() graphqlbackend.DateTime {
	return graphqlbackend.DateTime{Time: r.changeset.CreatedAt}
}

func (r *changesetResolver) UpdatedAt() graphqlbackend.DateTime {
	return graphqlbackend.DateTime{Time: r.changeset.UpdatedAt}
}

func (r *changesetResolver) NextSyncAt(ctx context.Context) (*graphqlbackend.DateTime, error) {
	// If code host syncs are disabled, the syncer is not actively syncing
	// changesets and the next sync time cannot be determined.
	if conf.Get().DisableAutoCodeHostSyncs {
		return nil, nil
	}

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
	if r.changeset.IsImporting() {
		return nil, nil
	}

	if r.changeset.Published() {
		t, err := r.changeset.Title()
		if err != nil {
			return nil, err
		}
		return &t, nil
	}

	desc, err := r.getBranchSpecDescription(ctx)
	if err != nil {
		return nil, err
	}

	return &desc.Title, nil
}

func (r *changesetResolver) Author() (*graphqlbackend.PersonResolver, error) {
	if r.changeset.IsImporting() {
		return nil, nil
	}

	if !r.changeset.Published() {
		return nil, nil
	}

	name, err := r.changeset.AuthorName()
	if err != nil {
		return nil, err
	}
	email, err := r.changeset.AuthorEmail()
	if err != nil {
		return nil, err
	}

	return graphqlbackend.NewPersonResolver(
		r.store.DB(),
		name,
		email,
		// Try to find the corresponding Sourcegraph user.
		true,
	), nil
}

func (r *changesetResolver) Body(ctx context.Context) (*string, error) {
	if r.changeset.IsImporting() {
		return nil, nil
	}

	if r.changeset.Published() {
		b, err := r.changeset.Body()
		if err != nil {
			return nil, err
		}
		return &b, nil
	}

	desc, err := r.getBranchSpecDescription(ctx)
	if err != nil {
		return nil, err
	}

	return &desc.Body, nil
}

func (r *changesetResolver) getBranchSpecDescription(ctx context.Context) (*btypes.ChangesetSpecDescription, error) {
	spec, err := r.computeSpec(ctx)
	if err != nil {
		return nil, err
	}

	if spec.Spec.IsImportingExisting() {
		return nil, errors.New("ChangesetSpec imports a changeset")
	}

	return spec.Spec, nil
}

func (r *changesetResolver) PublicationState() string {
	return string(r.changeset.PublicationState)
}

func (r *changesetResolver) ReconcilerState() string {
	return string(r.changeset.ReconcilerState)
}

func (r *changesetResolver) ExternalState() *string {
	if !r.changeset.Published() {
		return nil
	}
	state := string(r.changeset.ExternalState)
	return &state
}

func (r *changesetResolver) State() (string, error) {
	// Note that there's an inverse version of this function in
	// getRewirerMappingCurrentState(): if one changes, so should the other.

	switch r.changeset.ReconcilerState {
	case btypes.ReconcilerStateErrored:
		return string(btypes.ChangesetStateRetrying), nil
	case btypes.ReconcilerStateFailed:
		return string(btypes.ChangesetStateFailed), nil
	case btypes.ReconcilerStateScheduled:
		return string(btypes.ChangesetStateScheduled), nil
	default:
		if r.changeset.ReconcilerState != btypes.ReconcilerStateCompleted {
			return string(btypes.ChangesetStateProcessing), nil
		}
	}

	if r.changeset.PublicationState == btypes.ChangesetPublicationStateUnpublished {
		return string(btypes.ChangesetStateUnpublished), nil
	}

	switch r.changeset.ExternalState {
	case btypes.ChangesetExternalStateDraft:
		return string(btypes.ChangesetStateDraft), nil
	case btypes.ChangesetExternalStateOpen:
		return string(btypes.ChangesetStateOpen), nil
	case btypes.ChangesetExternalStateClosed:
		return string(btypes.ChangesetStateClosed), nil
	case btypes.ChangesetExternalStateMerged:
		return string(btypes.ChangesetStateMerged), nil
	case btypes.ChangesetExternalStateDeleted:
		return string(btypes.ChangesetStateDeleted), nil
	default:
		return "", fmt.Errorf("invalid ExternalState %q for state calculation", r.changeset.ExternalState)
	}
}

func (r *changesetResolver) ExternalURL() (*externallink.Resolver, error) {
	if !r.changeset.Published() {
		return nil, nil
	}
	if r.changeset.ExternalState == btypes.ChangesetExternalStateDeleted {
		return nil, nil
	}
	url, err := r.changeset.URL()
	if err != nil {
		return nil, err
	}
	if url == "" {
		return nil, nil
	}
	return externallink.NewResolver(url, r.changeset.ExternalServiceType), nil
}

func (r *changesetResolver) ReviewState(ctx context.Context) *string {
	if !r.changeset.Published() {
		return nil
	}
	state := string(r.changeset.ExternalReviewState)
	return &state
}

func (r *changesetResolver) CheckState() *string {
	if !r.changeset.Published() {
		return nil
	}

	state := string(r.changeset.ExternalCheckState)
	if state == string(btypes.ChangesetCheckStateUnknown) {
		return nil
	}

	return &state
}

func (r *changesetResolver) Error() *string { return r.changeset.FailureMessage }

func (r *changesetResolver) SyncerError() *string { return r.changeset.SyncErrorMessage }

func (r *changesetResolver) ScheduleEstimateAt(ctx context.Context) (*graphqlbackend.DateTime, error) {
	// We need to find out how deep in the queue this changeset is.
	place, err := r.store.GetChangesetPlaceInSchedulerQueue(ctx, r.changeset.ID)
	if err == store.ErrNoResults {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// Now we can ask the scheduler to estimate where this item would fall in
	// the schedule.
	return graphqlbackend.DateTimeOrNil(config.ActiveWindow().Estimate(r.store.Clock()(), place)), nil
}

func (r *changesetResolver) CurrentSpec(ctx context.Context) (graphqlbackend.VisibleChangesetSpecResolver, error) {
	if r.changeset.CurrentSpecID == 0 {
		return nil, nil
	}

	spec, err := r.computeSpec(ctx)
	if err != nil {
		return nil, err
	}

	return NewChangesetSpecResolverWithRepo(r.store, r.repo, spec), nil
}

func (r *changesetResolver) Labels(ctx context.Context) ([]graphqlbackend.ChangesetLabelResolver, error) {
	if !r.changeset.Published() {
		return []graphqlbackend.ChangesetLabelResolver{}, nil
	}

	// Not every code host supports labels on changesets so don't make a DB call unless we need to.
	if ok := r.changeset.SupportsLabels(); !ok {
		return []graphqlbackend.ChangesetLabelResolver{}, nil
	}

	opts := store.ListChangesetEventsOpts{
		ChangesetIDs: []int64{r.changeset.ID},
		Kinds:        state.ComputeLabelsRequiredEventTypes,
	}
	es, _, err := r.store.ListChangesetEvents(ctx, opts)
	if err != nil {
		return nil, err
	}
	// ComputeLabels expects the events to be pre-sorted.
	sort.Sort(state.ChangesetEvents(es))

	// We use changeset labels as the source of truth as they can be renamed
	// or removed but we'll also take into account any changeset events that
	// have happened since the last sync in order to reflect changes that
	// have come in via webhooks
	labels := state.ComputeLabels(r.changeset, es)
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
	if r.changeset.IsImporting() {
		return nil, nil
	}

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
			r.store.DB(),
			r.repoResolver,
			desc.BaseRev,
			diff,
		)
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

	return graphqlbackend.NewRepositoryComparison(ctx, r.store.DB(), r.repoResolver, &graphqlbackend.RepositoryComparisonInput{
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
	label btypes.ChangesetLabel
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
