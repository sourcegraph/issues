// Package types defines types used by the frontend.
package types

import (
	"database/sql"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

// RepoFields are lazy loaded data fields on a Repo (from the DB).
type RepoFields struct {
	// URI is the full name for this repository (e.g.,
	// "github.com/user/repo"). See the documentation for the Name field.
	URI string

	// Description is a brief description of the repository.
	Description string

	// Fork is whether this repository is a fork of another repository.
	Fork bool

	// Archived is whether this repository has been archived.
	Archived bool

	// Cloned is whether this repository is cloned.
	Cloned bool

	// CreatedAt indicates when the repository record was created.
	CreatedAt time.Time

	// UpdatedAt is when this repository's metadata was last updated on Sourcegraph.
	UpdatedAt time.Time

	// DeletedAt is when this repository was soft-deleted from Sourcegraph.
	DeletedAt time.Time

	// Metadata contains the raw source code host JSON metadata.
	Metadata interface{}

	// Sources identifies all the repo sources this Repo belongs to.
	// The key is a URN created by extsvc.URN
	Sources map[string]*SourceInfo
}

// A SourceInfo represents a source a Repo belongs to (such as an external service).
type SourceInfo struct {
	ID       string
	CloneURL string
}

// Repo represents a source code repository.
type Repo struct {
	// ID is the unique numeric ID for this repository.
	ID api.RepoID
	// ExternalRepo identifies this repository by its ID on the external service where it resides (and the external
	// service itself).
	ExternalRepo api.ExternalRepoSpec
	// Name is the name for this repository (e.g., "github.com/user/repo"). It
	// is the same as URI, unless the user configures a non-default
	// repositoryPathPattern.
	//
	// Previously, this was called RepoURI.
	Name api.RepoName

	// Private is whether the repository is private on the code host.
	Private bool

	// RepoFields contains fields that are loaded from the DB only when necessary.
	// This is to reduce memory usage when loading thousands of repos.
	*RepoFields
}

// Repos is an utility type of a list of repos.
type Repos []*Repo

func (rs Repos) Len() int           { return len(rs) }
func (rs Repos) Less(i, j int) bool { return rs[i].ID < rs[j].ID }
func (rs Repos) Swap(i, j int)      { rs[i], rs[j] = rs[j], rs[i] }

// ExternalService is a connection to an external service.
type ExternalService struct {
	ID              int64
	Kind            string
	DisplayName     string
	Config          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	LastSyncAt      *time.Time
	NextSyncAt      *time.Time
	NamespaceUserID *int32
}

// URN returns a unique resource identifier of this external service.
func (e *ExternalService) URN() string {
	return extsvc.URN(e.Kind, e.ID)
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
type CampaignsUsageStatistics struct {
	CampaignsCount              int32
	ActionChangesetsCount       int32
	ActionChangesetsMergedCount int32
	ManualChangesetsCount       int32
	ManualChangesetsMergedCount int32
}

// NOTE: DO NOT alter this struct without making a symmetric change
// to the updatecheck handler. This struct is marshalled and sent to
// BigQuery, which requires the input match its schema exactly.
type CodeIntelUsageStatistics struct {
	Daily   []*CodeIntelUsagePeriod
	Weekly  []*CodeIntelUsagePeriod
	Monthly []*CodeIntelUsagePeriod
}

// NOTE: DO NOT alter this struct without making a symmetric change
// to the updatecheck handler. This struct is marshalled and sent to
// BigQuery, which requires the input match its schema exactly.
type CodeIntelUsagePeriod struct {
	StartTime   time.Time
	Hover       *CodeIntelEventCategoryStatistics
	Definitions *CodeIntelEventCategoryStatistics
	References  *CodeIntelEventCategoryStatistics
}

// NOTE: DO NOT alter this struct without making a symmetric change
// to the updatecheck handler. This struct is marshalled and sent to
// BigQuery, which requires the input match its schema exactly.
type CodeIntelEventCategoryStatistics struct {
	LSIF   *CodeIntelEventStatistics
	LSP    *CodeIntelEventStatistics
	Search *CodeIntelEventStatistics
}

// NOTE: DO NOT alter this struct without making a symmetric change
// to the updatecheck handler. This struct is marshalled and sent to
// BigQuery, which requires the input match its schema exactly.
type CodeIntelEventStatistics struct {
	UsersCount  int32
	EventsCount *int32
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
	StartTime          time.Time
	TotalUsers         int32
	Literal            *SearchEventStatistics
	Regexp             *SearchEventStatistics
	After              *SearchCountStatistics
	Archived           *SearchCountStatistics
	Author             *SearchCountStatistics
	Before             *SearchCountStatistics
	Case               *SearchCountStatistics
	Commit             *SearchEventStatistics
	Committer          *SearchCountStatistics
	Content            *SearchCountStatistics
	Count              *SearchCountStatistics
	Diff               *SearchEventStatistics
	File               *SearchEventStatistics
	Fork               *SearchCountStatistics
	Index              *SearchCountStatistics
	Lang               *SearchCountStatistics
	Message            *SearchCountStatistics
	PatternType        *SearchCountStatistics
	Repo               *SearchEventStatistics
	Repohascommitafter *SearchCountStatistics
	Repohasfile        *SearchCountStatistics
	Repogroup          *SearchCountStatistics
	Structural         *SearchEventStatistics
	Symbol             *SearchEventStatistics
	Timeout            *SearchCountStatistics
	Type               *SearchCountStatistics
	SearchModes        *SearchModeUsageStatistics
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

// AggregatedEvent represents the total events, unique users, and
// latencies over the current month, week, and day for a single event.
type AggregatedEvent struct {
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
	RecentFilesClickedPercentage           float64
	RecentSearchClickedPercentage          float64
	RecentRepositoriesClickedPercentage    float64
	SavedSearchesClickedPercentage         float64
	NewSavedSearchesClickedPercentage      float64
	TotalPanelViews                        float64
	UsersFilesClickedPercentage            float64
	UsersSearchClickedPercentage           float64
	UsersRepositoriesClickedPercentage     float64
	UsersSavedSearchesClickedPercentage    float64
	UsersNewSavedSearchesClickedPercentage float64
	PercentUsersShown                      float64
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
