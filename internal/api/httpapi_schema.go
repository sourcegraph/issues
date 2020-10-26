package api

// RepoCreateOrUpdateRequest is a request to create or update a repository.
//
// The request handler determines if the request refers to an existing repository (and should therefore update
// instead of create). If ExternalRepo is set, then it tries to find a stored repository with the same ExternalRepo
// values. If ExternalRepo is not set, then it tries to find a stored repository with the same RepoName value.
//
// NOTE: Some fields are only used during creation (and are not used to update an existing repository).
type RepoCreateOrUpdateRequest struct {
	// ExternalRepo identifies this repository by its ID on the external service where it resides (and the external
	// service itself).
	ExternalRepo ExternalRepoSpec

	// RepoName is the repository's name.
	//
	// TODO(sqs): Add a way for callers to request that this repository be renamed.
	RepoName `json:"repo"`

	// Enabled is whether the repository should be enabled when initially created.
	//
	// NOTE: If the repository already exists when this request is received, its enablement is not updated. This
	// field is used only when creating the repository.
	Enabled bool `json:"enabled"`

	// Description is the repository's description on its external origin.
	Description string `json:"description"`

	// Fork is whether this repository is a fork (according to its external origin).
	Fork bool `json:"fork"`

	// Archived is whether this repository is archived (according to its external origin).
	Archived bool `json:"archived"`
}

type PhabricatorRepoCreateRequest struct {
	RepoName `json:"repo"`
	Callsign string `json:"callsign"`
	URL      string `json:"url"`
}

type ExternalServiceConfigsRequest struct {
	Kind    string `json:"kind"`
	Limit   int    `json:"limit"`
	AfterID int    `json:"after_id"`
}

type ExternalServicesListRequest struct {
	// NOTE(tsenart): We must keep this field in addition to the
	// Kinds field until after we roll-out this change, for backwards compatibility.
	Kind    string   `json:"kind"`
	Kinds   []string `json:"kinds"`
	Limit   int      `json:"limit"`
	AfterID int      `json:"after_id"`
}
