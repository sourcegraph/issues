// Package api contains an API client and types for cross-service communication.
package api

import (
	"fmt"
	"time"
)

// RepoID is the unique identifier for a repository.
type RepoID int32

// RepoName is the name of a repository, consisting of one or more "/"-separated path components.
//
// Previously, this was called RepoURI.
type RepoName string

// CommitID is the 40-character SHA-1 hash for a Git commit.
type CommitID string

// Repo represents a source code repository.
type Repo struct {
	// ID is the unique numeric ID for this repository on Sourcegraph.
	ID RepoID

	// ExternalRepo identifies this repository by its ID on the external service where it resides (and the external
	// service itself).
	ExternalRepo *ExternalRepoSpec

	// Name is the name of the repository (such as "github.com/user/repo").
	Name RepoName
	// Enabled is whether the repository is enabled. Disabled repositories are
	// not accessible by users (except site admins).
	Enabled bool
}

func (Repo) Fork() bool {
	// TODO(sqs): update callers
	return false
}

// ExternalRepoSpec specifies a repository on an external service (such as GitHub or GitLab).
type ExternalRepoSpec struct {
	// ID is the repository's ID on the external service. Its value is opaque except to the repo-updater.
	//
	// For GitHub, this is the GitHub GraphQL API's node ID for the repository.
	ID string

	// ServiceType is the type of external service. Its value is opaque except to the repo-updater.
	//
	// Example: "github", "gitlab", etc.
	ServiceType string

	// ServiceID is the particular instance of the external service where this repository resides. Its value is
	// opaque but typically consists of the canonical base URL to the service.
	//
	// Implementations must take care to normalize this URL. For example, if different GitHub.com repository code
	// paths used slightly different values here (such as "https://github.com/" and "https://github.com", note the
	// lack of trailing slash), then the same logical repository would be incorrectly treated as multiple distinct
	// repositories depending on the code path that provided its ServiceID value.
	//
	// Example: "https://github.com/", "https://github-enterprise.example.com/"
	ServiceID string
}

// Equal returns true if r is equal to s.
func (r ExternalRepoSpec) Equal(s *ExternalRepoSpec) bool {
	return r.ID == s.ID && r.ServiceType == s.ServiceType && r.ServiceID == s.ServiceID
}

// Compare returns -1 if r < s, 0 if r == s or 1 if r > s
func (r ExternalRepoSpec) Compare(s ExternalRepoSpec) int {
	if r.ServiceType != s.ServiceType {
		return cmp(r.ServiceType, s.ServiceType)
	}
	if r.ServiceID != s.ServiceID {
		return cmp(r.ServiceID, s.ServiceID)
	}
	return cmp(r.ID, s.ID)
}

func (r ExternalRepoSpec) String() string {
	return fmt.Sprintf("ExternalRepoSpec{%s %s %s}", r.ServiceID, r.ServiceType, r.ID)
}

// A SettingsSubject is something that can have settings. Exactly 1 field must be nonzero.
type SettingsSubject struct {
	Default bool   // whether this is for default settings
	Site    bool   // whether this is for global settings
	Org     *int32 // the org's ID
	User    *int32 // the user's ID
}

func (s SettingsSubject) String() string {
	switch {
	case s.Default:
		return "DefaultSettings"
	case s.Site:
		return "site"
	case s.Org != nil:
		return fmt.Sprintf("org %d", *s.Org)
	case s.User != nil:
		return fmt.Sprintf("user %d", *s.User)
	default:
		return "unknown settings subject"
	}
}

// Settings contains settings for a subject.
type Settings struct {
	ID           int32           // the unique ID of this settings value
	Subject      SettingsSubject // the subject of these settings
	AuthorUserID *int32          // the ID of the user who authored this settings value
	Contents     string          // the raw JSON (with comments and trailing commas allowed)
	CreatedAt    time.Time       // the date when this settings value was created
}

// ExternalService represents an complete external service record.
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

func cmp(a, b string) int {
	switch {
	case a < b:
		return -1
	case b < a:
		return 1
	default:
		return 0
	}
}
