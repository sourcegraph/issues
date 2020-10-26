package graphqlbackend

import (
	"context"
	"errors"

	"github.com/graph-gophers/graphql-go"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/externallink"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
)

type CreateCampaignArgs struct {
	CampaignSpec graphql.ID
}

type ApplyCampaignArgs struct {
	CampaignSpec   graphql.ID
	EnsureCampaign *graphql.ID
}

type MoveCampaignArgs struct {
	Campaign     graphql.ID
	NewName      *string
	NewNamespace *graphql.ID
}

type ListCampaignsArgs struct {
	First               int32
	After               *string
	State               *string
	ViewerCanAdminister *bool

	Namespace *graphql.ID
}

type CloseCampaignArgs struct {
	Campaign        graphql.ID
	CloseChangesets bool
}

type DeleteCampaignArgs struct {
	Campaign graphql.ID
}

type SyncChangesetArgs struct {
	Changeset graphql.ID
}

type CreateChangesetSpecArgs struct {
	ChangesetSpec string
}

type CreateCampaignSpecArgs struct {
	Namespace graphql.ID

	CampaignSpec   string
	ChangesetSpecs []graphql.ID
}

type ChangesetSpecsConnectionArgs struct {
	First int32
	After *string
}

type CampaignArgs struct {
	Namespace string
	Name      string
}

type ChangesetEventsConnectionArgs struct {
	First int32
	After *string
}

type CampaignsResolver interface {
	// Mutations
	CreateCampaign(ctx context.Context, args *CreateCampaignArgs) (CampaignResolver, error)
	ApplyCampaign(ctx context.Context, args *ApplyCampaignArgs) (CampaignResolver, error)
	MoveCampaign(ctx context.Context, args *MoveCampaignArgs) (CampaignResolver, error)
	CloseCampaign(ctx context.Context, args *CloseCampaignArgs) (CampaignResolver, error)
	DeleteCampaign(ctx context.Context, args *DeleteCampaignArgs) (*EmptyResponse, error)
	CreateChangesetSpec(ctx context.Context, args *CreateChangesetSpecArgs) (ChangesetSpecResolver, error)
	CreateCampaignSpec(ctx context.Context, args *CreateCampaignSpecArgs) (CampaignSpecResolver, error)
	SyncChangeset(ctx context.Context, args *SyncChangesetArgs) (*EmptyResponse, error)

	// Queries
	Campaigns(ctx context.Context, args *ListCampaignsArgs) (CampaignsConnectionResolver, error)
	Campaign(ctx context.Context, args *CampaignArgs) (CampaignResolver, error)
	CampaignByID(ctx context.Context, id graphql.ID) (CampaignResolver, error)
	ChangesetByID(ctx context.Context, id graphql.ID) (ChangesetResolver, error)

	CampaignSpecByID(ctx context.Context, id graphql.ID) (CampaignSpecResolver, error)
	ChangesetSpecByID(ctx context.Context, id graphql.ID) (ChangesetSpecResolver, error)
}

type CampaignSpecResolver interface {
	ID() graphql.ID

	OriginalInput() (string, error)
	ParsedInput() (JSONValue, error)
	ChangesetSpecs(ctx context.Context, args *ChangesetSpecsConnectionArgs) (ChangesetSpecConnectionResolver, error)

	Description() CampaignDescriptionResolver

	Creator(context.Context) (*UserResolver, error)
	CreatedAt() DateTime
	Namespace(context.Context) (*NamespaceResolver, error)

	ExpiresAt() *DateTime

	ApplyURL(ctx context.Context) (string, error)

	ViewerCanAdminister(context.Context) (bool, error)

	DiffStat(ctx context.Context) (*DiffStat, error)

	AppliesToCampaign(ctx context.Context) (CampaignResolver, error)
}

type CampaignDescriptionResolver interface {
	Name() string
	Description() string
}

type ChangesetSpecConnectionResolver interface {
	TotalCount(ctx context.Context) (int32, error)
	PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error)
	Nodes(ctx context.Context) ([]ChangesetSpecResolver, error)
}

type ChangesetSpecResolver interface {
	ID() graphql.ID

	Type() campaigns.ChangesetSpecDescriptionType

	ExpiresAt() *DateTime

	ToHiddenChangesetSpec() (HiddenChangesetSpecResolver, bool)
	ToVisibleChangesetSpec() (VisibleChangesetSpecResolver, bool)
}

type HiddenChangesetSpecResolver interface {
	ChangesetSpecResolver
}

type VisibleChangesetSpecResolver interface {
	ChangesetSpecResolver

	Description(ctx context.Context) (ChangesetDescription, error)
}

type ChangesetDescription interface {
	ToExistingChangesetReference() (ExistingChangesetReferenceResolver, bool)
	ToGitBranchChangesetDescription() (GitBranchChangesetDescriptionResolver, bool)
}

type ExistingChangesetReferenceResolver interface {
	BaseRepository() *RepositoryResolver
	ExternalID() string
}

type GitBranchChangesetDescriptionResolver interface {
	BaseRepository() *RepositoryResolver
	BaseRef() string
	BaseRev() string

	HeadRepository() *RepositoryResolver
	HeadRef() string

	Title() string
	Body() string

	Diff(ctx context.Context) (PreviewRepositoryComparisonResolver, error)
	DiffStat() *DiffStat

	Commits() []GitCommitDescriptionResolver

	Published() campaigns.PublishedValue
}

type GitCommitDescriptionResolver interface {
	Message() string
	Subject() string
	Body() *string
	Author() *PersonResolver
	Diff() string
}

type ChangesetCountsArgs struct {
	From *DateTime
	To   *DateTime
}

type ListChangesetsArgs struct {
	First                       int32
	After                       *string
	PublicationState            *campaigns.ChangesetPublicationState
	ReconcilerState             *[]campaigns.ReconcilerState
	ExternalState               *campaigns.ChangesetExternalState
	ReviewState                 *campaigns.ChangesetReviewState
	CheckState                  *campaigns.ChangesetCheckState
	OnlyPublishedByThisCampaign *bool
}

type CampaignResolver interface {
	ID() graphql.ID
	Name() string
	Description() *string
	InitialApplier(ctx context.Context) (*UserResolver, error)
	LastApplier(ctx context.Context) (*UserResolver, error)
	LastAppliedAt() DateTime
	SpecCreator(ctx context.Context) (*UserResolver, error)
	ViewerCanAdminister(ctx context.Context) (bool, error)
	URL(ctx context.Context) (string, error)
	Namespace(ctx context.Context) (n NamespaceResolver, err error)
	CreatedAt() DateTime
	UpdatedAt() DateTime
	Changesets(ctx context.Context, args *ListChangesetsArgs) (ChangesetsConnectionResolver, error)
	ChangesetCountsOverTime(ctx context.Context, args *ChangesetCountsArgs) ([]ChangesetCountsResolver, error)
	ClosedAt() *DateTime
	DiffStat(ctx context.Context) (*DiffStat, error)
	CurrentSpec(ctx context.Context) (CampaignSpecResolver, error)
}

type CampaignsConnectionResolver interface {
	Nodes(ctx context.Context) ([]CampaignResolver, error)
	TotalCount(ctx context.Context) (int32, error)
	PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error)
}

type ChangesetsConnectionStatsResolver interface {
	Unpublished() int32
	Draft() int32
	Open() int32
	Merged() int32
	Closed() int32
	Deleted() int32
	Total() int32
}

type ChangesetsConnectionResolver interface {
	Nodes(ctx context.Context) ([]ChangesetResolver, error)
	TotalCount(ctx context.Context) (int32, error)
	PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error)
	Stats(ctx context.Context) (ChangesetsConnectionStatsResolver, error)
}

type ChangesetLabelResolver interface {
	Text() string
	Color() string
	Description() *string
}

// ChangesetResolver is the "interface Changeset" in the GraphQL schema and is
// implemented by ExternalChangesetResolver and HiddenExternalChangesetResolver.
type ChangesetResolver interface {
	ID() graphql.ID

	CreatedAt() DateTime
	UpdatedAt() DateTime
	NextSyncAt(ctx context.Context) (*DateTime, error)
	PublicationState() campaigns.ChangesetPublicationState
	ReconcilerState() campaigns.ReconcilerState
	ExternalState() *campaigns.ChangesetExternalState
	Campaigns(ctx context.Context, args *ListCampaignsArgs) (CampaignsConnectionResolver, error)

	ToExternalChangeset() (ExternalChangesetResolver, bool)
	ToHiddenExternalChangeset() (HiddenExternalChangesetResolver, bool)
}

// HiddenExternalChangesetResolver implements only the common interface,
// ChangesetResolver, to not reveal information to unauthorized users.
//
// Theoretically this type is not necessary, but it's easier to understand the
// implementation of the GraphQL schema if we have a mapping between GraphQL
// types and Go types.
type HiddenExternalChangesetResolver interface {
	ChangesetResolver
}

// ExternalChangesetResolver implements the ChangesetResolver interface and
// additional data.
type ExternalChangesetResolver interface {
	ChangesetResolver

	ExternalID() *string
	Title(context.Context) (*string, error)
	Body(context.Context) (*string, error)
	ExternalURL() (*externallink.Resolver, error)
	ReviewState(context.Context) *campaigns.ChangesetReviewState
	CheckState() *campaigns.ChangesetCheckState
	Repository(ctx context.Context) *RepositoryResolver

	Events(ctx context.Context, args *ChangesetEventsConnectionArgs) (ChangesetEventsConnectionResolver, error)
	Diff(ctx context.Context) (RepositoryComparisonInterface, error)
	DiffStat(ctx context.Context) (*DiffStat, error)
	Labels(ctx context.Context) ([]ChangesetLabelResolver, error)

	Error() *string

	CurrentSpec(ctx context.Context) (VisibleChangesetSpecResolver, error)
}

type ChangesetEventsConnectionResolver interface {
	Nodes(ctx context.Context) ([]ChangesetEventResolver, error)
	TotalCount(ctx context.Context) (int32, error)
	PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error)
}

type ChangesetEventResolver interface {
	ID() graphql.ID
	Changeset() ExternalChangesetResolver
	CreatedAt() DateTime
}

type ChangesetCountsResolver interface {
	Date() DateTime
	Total() int32
	Merged() int32
	Closed() int32
	Draft() int32
	Open() int32
	OpenApproved() int32
	OpenChangesRequested() int32
	OpenPending() int32
}

var campaignsOnlyInEnterprise = errors.New("campaigns and changesets are only available in enterprise")

type defaultCampaignsResolver struct{}

var DefaultCampaignsResolver CampaignsResolver = defaultCampaignsResolver{}

// Mutations
func (defaultCampaignsResolver) CreateCampaign(ctx context.Context, args *CreateCampaignArgs) (CampaignResolver, error) {
	return nil, campaignsOnlyInEnterprise
}

func (defaultCampaignsResolver) ApplyCampaign(ctx context.Context, args *ApplyCampaignArgs) (CampaignResolver, error) {
	return nil, campaignsOnlyInEnterprise
}

func (defaultCampaignsResolver) CreateChangesetSpec(ctx context.Context, args *CreateChangesetSpecArgs) (ChangesetSpecResolver, error) {
	return nil, campaignsOnlyInEnterprise
}

func (defaultCampaignsResolver) CreateCampaignSpec(ctx context.Context, args *CreateCampaignSpecArgs) (CampaignSpecResolver, error) {
	return nil, campaignsOnlyInEnterprise
}

func (defaultCampaignsResolver) MoveCampaign(ctx context.Context, args *MoveCampaignArgs) (CampaignResolver, error) {
	return nil, campaignsOnlyInEnterprise
}

func (defaultCampaignsResolver) CloseCampaign(ctx context.Context, args *CloseCampaignArgs) (CampaignResolver, error) {
	return nil, campaignsOnlyInEnterprise
}

func (defaultCampaignsResolver) SyncChangeset(ctx context.Context, args *SyncChangesetArgs) (*EmptyResponse, error) {
	return nil, campaignsOnlyInEnterprise
}

func (defaultCampaignsResolver) DeleteCampaign(ctx context.Context, args *DeleteCampaignArgs) (*EmptyResponse, error) {
	return nil, campaignsOnlyInEnterprise
}

// Queries
func (defaultCampaignsResolver) CampaignByID(ctx context.Context, id graphql.ID) (CampaignResolver, error) {
	return nil, campaignsOnlyInEnterprise
}

func (defaultCampaignsResolver) Campaign(ctx context.Context, args *CampaignArgs) (CampaignResolver, error) {
	return nil, campaignsOnlyInEnterprise
}

func (defaultCampaignsResolver) Campaigns(ctx context.Context, args *ListCampaignsArgs) (CampaignsConnectionResolver, error) {
	return nil, campaignsOnlyInEnterprise
}

func (defaultCampaignsResolver) ChangesetByID(ctx context.Context, id graphql.ID) (ChangesetResolver, error) {
	return nil, campaignsOnlyInEnterprise
}

func (defaultCampaignsResolver) CampaignSpecByID(ctx context.Context, id graphql.ID) (CampaignSpecResolver, error) {
	return nil, campaignsOnlyInEnterprise
}

func (defaultCampaignsResolver) ChangesetSpecByID(ctx context.Context, id graphql.ID) (ChangesetSpecResolver, error) {
	return nil, campaignsOnlyInEnterprise
}
