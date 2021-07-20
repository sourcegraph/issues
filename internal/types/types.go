// Package types defines types used by the frontend.
package types

import (
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

// A SourceInfo represents a source a Repo belongs to (such as an external service).
type SourceInfo struct {
	ID       string
	CloneURL string
}

// ExternalServiceID returns the ID of the external service this
// SourceInfo refers to.
func (i SourceInfo) ExternalServiceID() int64 {
	_, id := extsvc.DecodeURN(i.ID)
	return id
}

// Repo represents a source code repository.
type Repo struct {
	// ID is the unique numeric ID for this repository.
	ID api.RepoID
	// Name is the name for this repository (e.g., "github.com/user/repo"). It
	// is the same as URI, unless the user configures a non-default
	// repositoryPathPattern.
	//
	// Previously, this was called RepoURI.
	Name api.RepoName
	// URI is the full name for this repository (e.g.,
	// "github.com/user/repo"). See the documentation for the Name field.
	URI string
	// Description is a brief description of the repository.
	Description string
	// Fork is whether this repository is a fork of another repository.
	Fork bool
	// Archived is whether the repository has been archived.
	Archived bool
	// Stars is the star count the repository has in the code host.
	Stars int `json:",omitempty"`
	// Private is whether the repository is private.
	Private bool
	// CreatedAt is when this repository was created on Sourcegraph.
	CreatedAt time.Time
	// UpdatedAt is when this repository's metadata was last updated on Sourcegraph.
	UpdatedAt time.Time
	// DeletedAt is when this repository was soft-deleted from Sourcegraph.
	DeletedAt time.Time
	// ExternalRepo identifies this repository by its ID on the external service where it resides (and the external
	// service itself).
	ExternalRepo api.ExternalRepoSpec
	// Sources identifies all the repo sources this Repo belongs to.
	// The key is a URN created by extsvc.URN
	Sources map[string]*SourceInfo
	// Metadata contains the raw source code host JSON metadata.
	Metadata interface{}
	// Blocked contains the reason this repository was blocked and the timestamp of when it happened.
	Blocked *RepoBlock `json:",omitempty"`
}

// RepoBlock contains data about a repo that has been blocked. Blocked repos aren't returned by store methods by default.
type RepoBlock struct {
	At     int64 // Unix timestamp
	Reason string
}

// CloneURLs returns all the clone URLs this repo is clonable from.
func (r *Repo) CloneURLs() []string {
	urls := make([]string, 0, len(r.Sources))
	for _, src := range r.Sources {
		if src != nil && src.CloneURL != "" {
			urls = append(urls, src.CloneURL)
		}
	}
	return urls
}

// IsDeleted returns true if the repo is deleted.
func (r *Repo) IsDeleted() bool { return !r.DeletedAt.IsZero() }

// ExternalServiceIDs returns the IDs of the external services this
// repo belongs to.
func (r *Repo) ExternalServiceIDs() []int64 {
	ids := make([]int64, 0, len(r.Sources))
	for _, src := range r.Sources {
		ids = append(ids, src.ExternalServiceID())
	}
	return ids
}

// BlockedRepoError is returned by a Repo IsBlocked method.
type BlockedRepoError struct {
	Name   api.RepoName
	Reason string
}

func (e BlockedRepoError) Error() string {
	return fmt.Sprintf("repository %s has been blocked. reason: %s", e.Name, e.Reason)
}

// Blocked implements the blocker interface in the errcode package.
func (e BlockedRepoError) Blocked() bool { return true }

// IsBlocked returns a non nil error if the repo has been blocked.
func (r *Repo) IsBlocked() error {
	if r.Blocked != nil {
		return &BlockedRepoError{Name: r.Name, Reason: r.Blocked.Reason}
	}
	return nil
}

// Update updates Repo r with the fields from the given newer Repo n,
// returning true if modified.
func (r *Repo) Update(n *Repo) (modified bool) {
	if !r.Name.Equal(n.Name) {
		r.Name, modified = n.Name, true
	}

	if r.URI != n.URI {
		r.URI, modified = n.URI, true
	}

	if r.Description != n.Description {
		r.Description, modified = n.Description, true
	}

	if n.ExternalRepo != (api.ExternalRepoSpec{}) &&
		!r.ExternalRepo.Equal(&n.ExternalRepo) {
		r.ExternalRepo, modified = n.ExternalRepo, true
	}

	if r.Archived != n.Archived {
		r.Archived, modified = n.Archived, true
	}

	if r.Fork != n.Fork {
		r.Fork, modified = n.Fork, true
	}

	if r.Private != n.Private {
		r.Private, modified = n.Private, true
	}

	if r.Stars != n.Stars {
		r.Stars, modified = n.Stars, true
	}

	if !reflect.DeepEqual(r.Metadata, n.Metadata) {
		r.Metadata, modified = n.Metadata, true
	}

	for urn, info := range n.Sources {
		if old, ok := r.Sources[urn]; !ok || !reflect.DeepEqual(info, old) {
			r.Sources[urn] = info
			modified = true
		}
	}

	return modified
}

// Clone returns a clone of the given repo.
func (r *Repo) Clone() *Repo {
	if r == nil {
		return nil
	}
	clone := *r
	if r.Sources != nil {
		clone.Sources = make(map[string]*SourceInfo, len(r.Sources))
		for k, v := range r.Sources {
			clone.Sources[k] = v
		}
	}
	return &clone
}

// Apply applies the given functional options to the Repo.
func (r *Repo) Apply(opts ...func(*Repo)) {
	if r == nil {
		return
	}

	for _, opt := range opts {
		opt(r)
	}
}

// With returns a clone of the given repo with the given functional options applied.
func (r *Repo) With(opts ...func(*Repo)) *Repo {
	clone := r.Clone()
	clone.Apply(opts...)
	return clone
}

// Less compares Repos by the important fields (fields with constraints in our
// DB). Additionally it will compare on Sources to give a deterministic order
// on repos returned from a sourcer.
//
// NewDiff relies on Less to deterministically decide on the order to merge
// repositories, as well as which repository to keep on conflicts.
//
// Context on using other fields such as timestamps to order/resolve
// conflicts: We only want to rely on values that have constraints in our
// database. Timestamps have the following downsides:
//
//   - We need to assume the upstream codehost has reasonable values for them
//   - Not all codehosts set them to relevant values (eg gitolite or other)
//   - They could change often for codehosts that do set them.
func (r *Repo) Less(s *Repo) bool {
	if r.ID != s.ID {
		return r.ID < s.ID
	}
	if r.Name != s.Name {
		return r.Name < s.Name
	}
	if cmp := r.ExternalRepo.Compare(s.ExternalRepo); cmp != 0 {
		return cmp == -1
	}

	return sortedSliceLess(sourcesKeys(r.Sources), sourcesKeys(s.Sources))
}

func (r *Repo) String() string {
	eid := fmt.Sprintf("{%s %s %s}", r.ExternalRepo.ServiceID, r.ExternalRepo.ServiceType, r.ExternalRepo.ID)
	if r.IsDeleted() {
		return fmt.Sprintf("Repo{ID: %d, Name: %q, EID: %s, IsDeleted: true}", r.ID, r.Name, eid)
	}
	return fmt.Sprintf("Repo{ID: %d, Name: %q, EID: %s}", r.ID, r.Name, eid)
}

func sourcesKeys(m map[string]*SourceInfo) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// sortedSliceLess returns true if a < b
func sortedSliceLess(a, b []string) bool {
	for i, v := range a {
		if i == len(b) {
			return false
		}
		if v != b[i] {
			return v < b[i]
		}
	}
	return len(a) != len(b)
}

// Repos is an utility type with convenience methods for operating on lists of Repos.
type Repos []*Repo

func (rs Repos) Len() int           { return len(rs) }
func (rs Repos) Less(i, j int) bool { return rs[i].Less(rs[j]) }
func (rs Repos) Swap(i, j int)      { rs[i], rs[j] = rs[j], rs[i] }

// IDs returns the list of ids from all Repos.
func (rs Repos) IDs() []api.RepoID {
	ids := make([]api.RepoID, len(rs))
	for i := range rs {
		ids[i] = rs[i].ID
	}
	return ids
}

// Names returns the list of names from all Repos.
func (rs Repos) Names() []string {
	names := make([]string, len(rs))
	for i := range rs {
		names[i] = string(rs[i].Name)
	}
	return names
}

// NamesSummary caps the number of repos to 20 when composing a space-separated list string.
// Used in logging statements.
func (rs Repos) NamesSummary() string {
	if len(rs) > 20 {
		return strings.Join(rs[:20].Names(), " ") + "..."
	}
	return strings.Join(rs.Names(), " ")
}

// Kinds returns the unique set of kinds from all Repos.
func (rs Repos) Kinds() (kinds []string) {
	set := map[string]bool{}
	for _, r := range rs {
		kind := strings.ToUpper(r.ExternalRepo.ServiceType)
		if !set[kind] {
			kinds = append(kinds, kind)
			set[kind] = true
		}
	}
	return kinds
}

// ExternalRepos returns the list of set ExternalRepoSpecs from all Repos.
func (rs Repos) ExternalRepos() []api.ExternalRepoSpec {
	specs := make([]api.ExternalRepoSpec, 0, len(rs))
	for _, r := range rs {
		specs = append(specs, r.ExternalRepo)
	}
	return specs
}

// Sources returns a map of all the sources per repo id.
func (rs Repos) Sources() map[api.RepoID][]SourceInfo {
	sources := make(map[api.RepoID][]SourceInfo)
	for i := range rs {
		for _, info := range rs[i].Sources {
			sources[rs[i].ID] = append(sources[rs[i].ID], *info)
		}
	}

	return sources
}

// Concat adds the given Repos to the end of rs.
func (rs *Repos) Concat(others ...Repos) {
	for _, o := range others {
		*rs = append(*rs, o...)
	}
}

// Clone returns a clone of Repos.
func (rs Repos) Clone() Repos {
	o := make(Repos, 0, len(rs))
	for _, r := range rs {
		o = append(o, r.Clone())
	}
	return o
}

// Apply applies the given functional options to the Repo.
func (rs Repos) Apply(opts ...func(*Repo)) {
	for _, r := range rs {
		r.Apply(opts...)
	}
}

// With returns a clone of the given repos with the given functional options applied.
func (rs Repos) With(opts ...func(*Repo)) Repos {
	clone := rs.Clone()
	clone.Apply(opts...)
	return clone
}

// Filter returns all the Repos that match the given predicate.
func (rs Repos) Filter(pred func(*Repo) bool) (fs Repos) {
	for _, r := range rs {
		if pred(r) {
			fs = append(fs, r)
		}
	}
	return fs
}

// RepoName represents a source code repository name and its ID.
type RepoName struct {
	ID   api.RepoID
	Name api.RepoName
}

func (r *RepoName) ToRepo() *Repo {
	return &Repo{
		ID:   r.ID,
		Name: r.Name,
	}
}

// RepoNames is an utility type with convenience methods for operating on lists of repo names
type RepoNames []RepoName

func (rs RepoNames) Len() int           { return len(rs) }
func (rs RepoNames) Less(i, j int) bool { return rs[i].ID < rs[j].ID }
func (rs RepoNames) Swap(i, j int)      { rs[i], rs[j] = rs[j], rs[i] }

type CodeHostRepository struct {
	Name       string
	CodeHostID int64
	Private    bool
}

// RepoGitserverStatus includes basic repo data along with the current gitserver
// status for the repo, which may be unknown.
type RepoGitserverStatus struct {
	// ID is the unique numeric ID for this repository.
	ID api.RepoID
	// Name is the name for this repository (e.g., "github.com/user/repo").
	Name api.RepoName

	// GitserverRepo data if it exists
	*GitserverRepo
}

type CloneStatus string

const (
	CloneStatusUnknown   CloneStatus = ""
	CloneStatusNotCloned CloneStatus = "not_cloned"
	CloneStatusCloning   CloneStatus = "cloning"
	CloneStatusCloned    CloneStatus = "cloned"
)

func ParseCloneStatus(s string) CloneStatus {
	cs := CloneStatus(s)
	switch cs {
	case CloneStatusNotCloned, CloneStatusCloning, CloneStatusCloned:
		return cs
	default:
		return CloneStatusUnknown
	}
}

// GitserverRepo  represents the data gitserver knows about a repo
type GitserverRepo struct {
	RepoID api.RepoID
	// Usually represented by a gitserver hostname
	ShardID     string
	CloneStatus CloneStatus
	// The last external service used to sync or clone this repo
	LastExternalService int64
	// The last error that occurred or empty if the last action was successful
	LastError string
	UpdatedAt time.Time
}

// ExternalService is a connection to an external service.
type ExternalService struct {
	ID              int64
	Kind            string
	DisplayName     string
	Config          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       time.Time
	LastSyncAt      time.Time
	NextSyncAt      time.Time
	NamespaceUserID int32
	Unrestricted    bool // Whether access to repositories belong to this external service is unrestricted.
	CloudDefault    bool // Whether this external service is our default public service on Cloud
}

// ExternalServiceSyncJob represents an sync job for an external service
type ExternalServiceSyncJob struct {
	ID                int64
	State             string
	FailureMessage    string
	StartedAt         time.Time
	FinishedAt        time.Time
	ProcessAfter      time.Time
	NumResets         int
	ExternalServiceID int64
	NumFailures       int
}

// URN returns a unique resource identifier of this external service,
// used as the key in a repo's Sources map as well as the SourceInfo ID.
func (e *ExternalService) URN() string {
	return extsvc.URN(e.Kind, e.ID)
}

// IsDeleted returns true if the external service is deleted.
func (e *ExternalService) IsDeleted() bool { return !e.DeletedAt.IsZero() }

// Update updates ExternalService e with the fields from the given newer ExternalService n,
// returning true if modified.
func (e *ExternalService) Update(n *ExternalService) (modified bool) {
	if e.ID != n.ID {
		return false
	}

	if !strings.EqualFold(e.Kind, n.Kind) {
		e.Kind, modified = strings.ToUpper(n.Kind), true
	}

	if e.DisplayName != n.DisplayName {
		e.DisplayName, modified = n.DisplayName, true
	}

	if e.Config != n.Config {
		e.Config, modified = n.Config, true
	}

	if !e.UpdatedAt.Equal(n.UpdatedAt) {
		e.UpdatedAt, modified = n.UpdatedAt, true
	}

	if !e.DeletedAt.Equal(n.DeletedAt) {
		e.DeletedAt, modified = n.DeletedAt, true
	}

	return modified
}

// Configuration returns the external service config.
func (e *ExternalService) Configuration() (cfg interface{}, _ error) {
	return extsvc.ParseConfig(e.Kind, e.Config)
}

// Clone returns a clone of the given external service.
func (e *ExternalService) Clone() *ExternalService {
	clone := *e
	return &clone
}

// Apply applies the given functional options to the ExternalService.
func (e *ExternalService) Apply(opts ...func(*ExternalService)) {
	if e == nil {
		return
	}

	for _, opt := range opts {
		opt(e)
	}
}

// With returns a clone of the given repo with the given functional options applied.
func (e *ExternalService) With(opts ...func(*ExternalService)) *ExternalService {
	clone := e.Clone()
	clone.Apply(opts...)
	return clone
}

// ExternalServices is an utility type with
// convenience methods for operating on lists of ExternalServices.
type ExternalServices []*ExternalService

// IDs returns the list of ids from all ExternalServices.
func (es ExternalServices) IDs() []int64 {
	ids := make([]int64, len(es))
	for i := range es {
		ids[i] = es[i].ID
	}
	return ids
}

// DisplayNames returns the list of display names from all ExternalServices.
func (es ExternalServices) DisplayNames() []string {
	names := make([]string, len(es))
	for i := range es {
		names[i] = es[i].DisplayName
	}
	return names
}

// Kinds returns the unique set of Kinds in the given external services list.
func (es ExternalServices) Kinds() (kinds []string) {
	set := make(map[string]bool, len(es))
	for _, e := range es {
		if !set[e.Kind] {
			kinds = append(kinds, e.Kind)
			set[e.Kind] = true
		}
	}
	return kinds
}

// URNs returns the list of URNs from all ExternalServices.
func (es ExternalServices) URNs() []string {
	urns := make([]string, len(es))
	for i := range es {
		urns[i] = es[i].URN()
	}
	return urns
}

func (es ExternalServices) Len() int {
	return len(es)
}

func (es ExternalServices) Swap(i, j int) {
	es[i], es[j] = es[j], es[i]
}

func (es ExternalServices) Less(i, j int) bool {
	return es[i].ID < es[j].ID
}

// Clone returns a clone of the given external services.
func (es ExternalServices) Clone() ExternalServices {
	o := make(ExternalServices, 0, len(es))
	for _, r := range es {
		o = append(o, r.Clone())
	}
	return o
}

// Apply applies the given functional options to the ExternalService.
func (es ExternalServices) Apply(opts ...func(*ExternalService)) {
	for _, r := range es {
		r.Apply(opts...)
	}
}

// With returns a clone of the given external services with the given functional options applied.
func (es ExternalServices) With(opts ...func(*ExternalService)) ExternalServices {
	clone := es.Clone()
	clone.Apply(opts...)
	return clone
}

type GlobalState struct {
	SiteID      string
	Initialized bool // whether the initial site admin account has been created
}

// User represents a registered user.
type User struct {
	ID                    int32
	Username              string
	DisplayName           string
	AvatarURL             string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	SiteAdmin             bool
	BuiltinAuth           bool
	Tags                  []string
	InvalidatedSessionsAt time.Time
}

type Org struct {
	ID          int32
	Name        string
	DisplayName *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type OrgMembership struct {
	ID        int32
	OrgID     int32
	UserID    int32
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PhabricatorRepo struct {
	ID       int32
	Name     api.RepoName
	URL      string
	Callsign string
}

type UserUsageStatistics struct {
	UserID                      int32
	PageViews                   int32
	SearchQueries               int32
	CodeIntelligenceActions     int32
	FindReferencesActions       int32
	LastActiveTime              *time.Time
	LastCodeHostIntegrationTime *time.Time
}

// UserUsageCounts captures the usage numbers of a user in a single day.
type UserUsageCounts struct {
	Date           time.Time
	UserID         uint32
	SearchCount    int32
	CodeIntelCount int32
}

// UserDates captures the created and deleted dates of a single user.
type UserDates struct {
	UserID    int32
	CreatedAt time.Time
	DeletedAt time.Time
}

// NOTE: DO NOT alter this struct without making a symmetric change
// to the updatecheck handler. This struct is marshalled and sent to
// BigQuery, which requires the input match its schema exactly.
type SiteUsageStatistics struct {
	DAUs []*SiteActivityPeriod
	WAUs []*SiteActivityPeriod
	MAUs []*SiteActivityPeriod
}

// NOTE: DO NOT alter this struct without making a symmetric change
// to the updatecheck handler. This struct is marshalled and sent to
// BigQuery, which requires the input match its schema exactly.
type SiteActivityPeriod struct {
	StartTime            time.Time
	UserCount            int32
	RegisteredUserCount  int32
	AnonymousUserCount   int32
	IntegrationUserCount int32
}

// NOTE: DO NOT alter this struct without making a symmetric change
// to the updatecheck handler. This struct is marshalled and sent to
// BigQuery, which requires the input match its schema exactly.
type BatchChangesUsageStatistics struct {
	// ViewBatchChangeApplyPageCount is the number of page views on the apply page
	// ("preview" page).
	ViewBatchChangeApplyPageCount int32
	// ViewBatchChangeDetailsPageAfterCreateCount is the number of page views on
	// the batch changes details page *after creating* the batch change on the apply
	// page by clicking "Apply spec".
	ViewBatchChangeDetailsPageAfterCreateCount int32
	// ViewBatchChangeDetailsPageAfterUpdateCount is the number of page views on
	// the batch changes details page *after updating* a batch change on the apply page
	// by clicking "Apply spec".
	ViewBatchChangeDetailsPageAfterUpdateCount int32

	// BatchChangesCount is the number of batch changes on the instance. This can go
	// down when users delete a batch change.
	BatchChangesCount int32
	// BatchChangesClosedCount is the number of *closed* batch changes on the
	// instance. This can go down when users delete a batch change.
	BatchChangesClosedCount int32

	// BatchSpecsCreatedCount is the number of batch change specs that have been
	// created by running `src batch [preview|apply]`. This number never
	// goes down since it's based on event logs, even if the batch specs
	// were not used and cleaned up.
	BatchSpecsCreatedCount int32
	// ChangesetSpecsCreatedCount is the number of changeset specs that have
	// been created by running `src batch [preview|apply]`. This number
	// never goes down since it's based on event logs, even if the changeset
	// specs were not used and cleaned up.
	ChangesetSpecsCreatedCount int32

	// PublishedChangesetsUnpublishedCount is the number of changesets in the
	// database that have not been published but belong to a batch change.
	// This number *could* go down, since it's not
	// based on event logs, but so far (Mar 2021) we never cleaned up
	// changesets in the database.
	PublishedChangesetsUnpublishedCount int32

	// PublishedChangesetsCount is the number of changesets published on code hosts
	// by batch changes. This number *could* go down, since it's not based on
	// event logs, but so far (Mar 2021) we never cleaned up changesets in the
	// database.
	PublishedChangesetsCount int32
	// PublishedChangesetsDiffStatAddedSum is the total sum of lines added by
	// changesets published on the code host by batch changes.
	PublishedChangesetsDiffStatAddedSum int32
	// PublishedChangesetsDiffStatChangedSum is the total sum of lines changed by
	// changesets published on the code host by batch changes.
	PublishedChangesetsDiffStatChangedSum int32
	// PublishedChangesetsDiffStatDeletedSum is the total sum of lines deleted by
	// changesets published on the code host by batch changes.
	PublishedChangesetsDiffStatDeletedSum int32

	// PublishedChangesetsMergedCount is the number of changesets published on
	// code hosts by batch changes that have also been *merged*.
	// This number *could* go down, since it's not based on event logs, but
	// so far (Mar 2021) we never cleaned up changesets in the database.
	PublishedChangesetsMergedCount int32
	// PublishedChangesetsMergedDiffStatAddedSum is the total sum of lines added by
	// changesets published on the code host by batch changes and merged.
	PublishedChangesetsMergedDiffStatAddedSum int32
	// PublishedChangesetsMergedDiffStatChangedSum is the total sum of lines changed by
	// changesets published on the code host by batch changes and merged.
	PublishedChangesetsMergedDiffStatChangedSum int32
	// PublishedChangesetsMergedDiffStatDeletedSum is the total sum of lines deleted by
	// changesets published on the code host by batch changes and merged.
	PublishedChangesetsMergedDiffStatDeletedSum int32

	// ImportedChangesetsCount is the total number of changesets that have been
	// imported by a batch change to be tracked.
	// This number *could* go down, since it's not based on event logs, but
	// so far (Mar 2021) we never cleaned up changesets in the database.
	ImportedChangesetsCount int32
	// ManualChangesetsCount is the total number of *merged* changesets that
	// have been imported by a batch change to be tracked.
	// This number *could* go down, since it's not based on event logs, but
	// so far (Mar 2021) we never cleaned up changesets in the database.
	ImportedChangesetsMergedCount int32

	// CurrentMonthContributorsCount is the count of unique users that have logged a
	// "contributing" batch changes event, such as "BatchChangeCreated".
	//
	// See `contributorsEvents` in `GetBatchChangesUsageStatistics` for a full list
	// of events.
	CurrentMonthContributorsCount int64

	// CurrentMonthUsersCount is the count of unique users that have logged a
	// "using" batch changes event, such as "ViewBatchChangesListPage" and also "BatchChangeCreated".
	//
	// See `contributorsEvents` in `GetBatchChangesUsageStatistics` for a full
	// list of events.
	CurrentMonthUsersCount int64

	BatchChangesCohorts []*BatchChangesCohort
}

// NOTE: DO NOT alter this struct without making a symmetric change
// to the updatecheck handler. This struct is marshalled and sent to
// BigQuery, which requires the input match its schema exactly.
type BatchChangesCohort struct {
	// Week is the week of this cohort and is used to group batch changes by
	// their creation date.
	Week string

	// BatchChangesClosed is the number of batch changes that were created in Week and
	// are currently closed.
	BatchChangesClosed int64

	// BatchChangesOpen is the number of batch changes that were created in Week and
	// are currently open.
	BatchChangesOpen int64

	// The following are the counts of the changesets that are currently
	// attached to the batch changes in this cohort.

	ChangesetsImported        int64
	ChangesetsUnpublished     int64
	ChangesetsPublished       int64
	ChangesetsPublishedOpen   int64
	ChangesetsPublishedDraft  int64
	ChangesetsPublishedMerged int64
	ChangesetsPublishedClosed int64
}

// NOTE: DO NOT alter this struct without making a symmetric change
// to the updatecheck handler. This struct is marshalled and sent to
// BigQuery, which requires the input match its schema exactly.
type SearchUsageStatistics struct {
	Daily   []*SearchUsagePeriod
	Weekly  []*SearchUsagePeriod
	Monthly []*SearchUsagePeriod
}

// NOTE: DO NOT alter this struct without making a symmetric change
// to the updatecheck handler. This struct is marshalled and sent to
// BigQuery, which requires the input match its schema exactly.
type SearchUsagePeriod struct {
	StartTime  time.Time
	TotalUsers int32

	// Counts and latency statistics for different kinds of searches.
	Literal    *SearchEventStatistics
	Regexp     *SearchEventStatistics
	Commit     *SearchEventStatistics
	Diff       *SearchEventStatistics
	File       *SearchEventStatistics
	Structural *SearchEventStatistics
	Symbol     *SearchEventStatistics

	// Counts of search query attributes. Ref: RFC 384.
	OperatorOr              *SearchCountStatistics
	OperatorAnd             *SearchCountStatistics
	OperatorNot             *SearchCountStatistics
	SelectRepo              *SearchCountStatistics
	SelectFile              *SearchCountStatistics
	SelectContent           *SearchCountStatistics
	SelectSymbol            *SearchCountStatistics
	SelectCommitDiffAdded   *SearchCountStatistics
	SelectCommitDiffRemoved *SearchCountStatistics
	RepoContains            *SearchCountStatistics
	RepoContainsFile        *SearchCountStatistics
	RepoContainsContent     *SearchCountStatistics
	RepoContainsCommitAfter *SearchCountStatistics
	CountAll                *SearchCountStatistics
	NonGlobalContext        *SearchCountStatistics
	OnlyPatterns            *SearchCountStatistics
	OnlyPatternsThreeOrMore *SearchCountStatistics

	// DEPRECATED. Counts statistics for fields.
	After              *SearchCountStatistics
	Archived           *SearchCountStatistics
	Author             *SearchCountStatistics
	Before             *SearchCountStatistics
	Case               *SearchCountStatistics
	Committer          *SearchCountStatistics
	Content            *SearchCountStatistics
	Count              *SearchCountStatistics
	Fork               *SearchCountStatistics
	Index              *SearchCountStatistics
	Lang               *SearchCountStatistics
	Message            *SearchCountStatistics
	PatternType        *SearchCountStatistics
	Repo               *SearchEventStatistics
	Repohascommitafter *SearchCountStatistics
	Repohasfile        *SearchCountStatistics
	Repogroup          *SearchCountStatistics
	Timeout            *SearchCountStatistics
	Type               *SearchCountStatistics

	// DEPRECATED. Search modes statistics refers to removed functionality.
	SearchModes *SearchModeUsageStatistics
}

type SearchModeUsageStatistics struct {
	Interactive *SearchCountStatistics
	PlainText   *SearchCountStatistics
}

type SearchCountStatistics struct {
	UserCount   *int32
	EventsCount *int32
}

// NOTE: DO NOT alter this struct without making a symmetric change
// to the updatecheck handler. This struct is marshalled and sent to
// BigQuery, which requires the input match its schema exactly.
type SearchEventStatistics struct {
	UserCount      *int32
	EventsCount    *int32
	EventLatencies *SearchEventLatencies
}

// NOTE: DO NOT alter this struct without making a symmetric change
// to the updatecheck handler. This struct is marshalled and sent to
// BigQuery, which requires the input match its schema exactly.
type SearchEventLatencies struct {
	P50 float64
	P90 float64
	P99 float64
}

// SiteUsageSummary is an alternate view of SiteUsageStatistics which is
// calculated in the database layer.
type SiteUsageSummary struct {
	Month                   time.Time
	Week                    time.Time
	Day                     time.Time
	UniquesMonth            int32
	UniquesWeek             int32
	UniquesDay              int32
	RegisteredUniquesMonth  int32
	RegisteredUniquesWeek   int32
	RegisteredUniquesDay    int32
	IntegrationUniquesMonth int32
	IntegrationUniquesWeek  int32
	IntegrationUniquesDay   int32
	ManageUniquesMonth      int32
	CodeUniquesMonth        int32
	VerifyUniquesMonth      int32
	MonitorUniquesMonth     int32
	ManageUniquesWeek       int32
	CodeUniquesWeek         int32
	VerifyUniquesWeek       int32
	MonitorUniquesWeek      int32
}

// SearchAggregatedEvent represents the total events, unique users, and
// latencies over the current month, week, and day for a single search event.
type SearchAggregatedEvent struct {
	Name           string
	Month          time.Time
	Week           time.Time
	Day            time.Time
	TotalMonth     int32
	TotalWeek      int32
	TotalDay       int32
	UniquesMonth   int32
	UniquesWeek    int32
	UniquesDay     int32
	LatenciesMonth []float64
	LatenciesWeek  []float64
	LatenciesDay   []float64
}

type SurveyResponse struct {
	ID        int32
	UserID    *int32
	Email     *string
	Score     int32
	Reason    *string
	Better    *string
	CreatedAt time.Time
}

type Event struct {
	ID              int32
	Name            string
	URL             string
	UserID          *int32
	AnonymousUserID string
	Argument        string
	Source          string
	Version         string
	Timestamp       time.Time
}

// GrowthStatistics represents the total users that were created,
// deleted, resurrected, churned and retained over the current month.
type GrowthStatistics struct {
	DeletedUsers     int32
	CreatedUsers     int32
	ResurrectedUsers int32
	ChurnedUsers     int32
	RetainedUsers    int32
}

// SavedSearches represents the total number of saved searches, users
// using saved searches, and usage of saved searches.
type SavedSearches struct {
	TotalSavedSearches   int32
	UniqueUsers          int32
	NotificationsSent    int32
	NotificationsClicked int32
	UniqueUserPageViews  int32
	OrgSavedSearches     int32
}

// Panel homepage represents interaction data on the
// enterprise homepage panels.
type HomepagePanels struct {
	RecentFilesClickedPercentage           *float64
	RecentSearchClickedPercentage          *float64
	RecentRepositoriesClickedPercentage    *float64
	SavedSearchesClickedPercentage         *float64
	NewSavedSearchesClickedPercentage      *float64
	TotalPanelViews                        *float64
	UsersFilesClickedPercentage            *float64
	UsersSearchClickedPercentage           *float64
	UsersRepositoriesClickedPercentage     *float64
	UsersSavedSearchesClickedPercentage    *float64
	UsersNewSavedSearchesClickedPercentage *float64
	PercentUsersShown                      *float64
}

type WeeklyRetentionStats struct {
	WeekStart  time.Time
	CohortSize *int32
	Week0      *float64
	Week1      *float64
	Week2      *float64
	Week3      *float64
	Week4      *float64
	Week5      *float64
	Week6      *float64
	Week7      *float64
	Week8      *float64
	Week9      *float64
	Week10     *float64
	Week11     *float64
}

type RetentionStats struct {
	Weekly []*WeeklyRetentionStats
}

type SearchOnboarding struct {
	TotalOnboardingTourViews   *int32
	ViewedLangStep             *int32
	ViewedFilterRepoStep       *int32
	ViewedAddQueryTermStep     *int32
	ViewedSubmitSearchStep     *int32
	ViewedSearchReferenceStep  *int32
	CloseOnboardingTourClicked *int32
}

// Weekly usage statistics for the extensions platform
type ExtensionsUsageStatistics struct {
	WeekStart                  time.Time
	UsageStatisticsByExtension []*ExtensionUsageStatistics
	// Average number of non-default extensions used by users
	// that have used at least one non-default extension
	AverageNonDefaultExtensions *float64
	// The count of users that have activated a non-default extension this week
	NonDefaultExtensionUsers *int32
}

// Weekly statistics for an individual extension
type ExtensionUsageStatistics struct {
	// The count of users that have activated this extension
	UserCount *int32
	// The average number of activations for users that have
	// used this extension at least once
	AverageActivations *float64
	ExtensionID        *string
}

type CodeInsightsUsageStatistics struct {
	WeeklyUsageStatisticsByInsight []*InsightUsageStatistics
	WeeklyInsightsPageViews        *int32
	WeeklyInsightsUniquePageViews  *int32
	WeeklyInsightConfigureClick    *int32
	WeeklyInsightAddMoreClick      *int32
	WeekStart                      time.Time
	WeeklyInsightCreators          *int32
	WeeklyFirstTimeInsightCreators *int32
	WeeklyAggregatedUsage          []AggregatedPingStats
	InsightTimeIntervals           []InsightTimeIntervalPing
	InsightOrgVisible              []OrgVisibleInsightPing
}

// Usage statistics for a type of code insight
type InsightUsageStatistics struct {
	InsightType      *string
	Additions        *int32
	Edits            *int32
	Removals         *int32
	Hovers           *int32
	UICustomizations *int32
	DataPointClicks  *int32
}

type PingName string

// AggregatedPingStats is a generic representation of an aggregated ping statistic
type AggregatedPingStats struct {
	Name        PingName
	TotalCount  int
	UniqueCount int
}

type InsightTimeIntervalPing struct {
	IntervalDays int
	TotalCount   int
}

type OrgVisibleInsightPing struct {
	Type       string
	TotalCount int
}

type CodeMonitoringUsageStatistics struct {
	CodeMonitoringPageViews                       *int32
	CreateCodeMonitorPageViews                    *int32
	CreateCodeMonitorPageViewsWithTriggerQuery    *int32
	CreateCodeMonitorPageViewsWithoutTriggerQuery *int32
	ManageCodeMonitorPageViews                    *int32
	CodeMonitorEmailLinkClicks                    *int32
}

// Secret represents the secrets table
type Secret struct {
	ID int32

	// The table containing an object whose token is being encrypted.
	SourceType sql.NullString

	// The ID of the object in the SourceType table.
	SourceID sql.NullInt32

	// KeyName represents a unique key for the case where we're storing key-value pairs.
	KeyName sql.NullString

	// Value contains the encrypted string
	Value string
}

type SearchContext struct {
	ID int64
	// Name contains the non-prefixed part of the search context spec.
	// The name is a substring of the spec and it should NOT be used as the spec itself.
	// The spec contains additional information (such as the @ prefix and the context namespace)
	// that helps differentiate between different search contexts.
	// Example mappings from context spec to context name:
	// global -> global, @user -> user, @org -> org,
	// @user/ctx1 -> ctx1, @org/ctx2 -> ctx2.
	Name        string
	Description string
	// Public property controls the visibility of the search context. Public search context is available to
	// any user on the instance. If a public search context contains private repositories, those are filtered out
	// for unauthorized users. Private search contexts are only available to their owners. Private user search context
	// is available only to the user, private org search context is available only to the members of the org, and private
	// instance-level search contexts is available only to site-admins.
	Public          bool
	NamespaceUserID int32 // if non-zero, the owner is this user. NamespaceUserID/NamespaceOrgID are mutually exclusive.
	NamespaceOrgID  int32 // if non-zero, the owner is this organization. NamespaceUserID/NamespaceOrgID are mutually exclusive.
	UpdatedAt       time.Time

	// We cache namespace names to avoid separate database lookups when constructing the search context spec

	// NamespaceUserName is the name of the user if NamespaceUserID is present.
	NamespaceUserName string
	// NamespaceUserName is the name of the org if NamespaceOrgID is present.
	NamespaceOrgName string
}

// SearchContextRepositoryRevisions is a simple wrapper for a repository and its revisions
// contained in a search context. It is made compatible with search.RepositoryRevisions, so it can be easily
// converted when needed. We could use search.RepositoryRevisions directly instead, but it
// introduces an import cycle with `internal/vcs/git` package when used in `internal/database/search_contexts.go`.
type SearchContextRepositoryRevisions struct {
	Repo      RepoName
	Revisions []string
}
