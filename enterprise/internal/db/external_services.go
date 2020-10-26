package db

import (
	"github.com/sourcegraph/sourcegraph/internal/authz/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/authz/github"
	"github.com/sourcegraph/sourcegraph/internal/authz/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/schema"
)

// NewExternalServicesStore returns an OSS db.ExternalServicesStore set with
// enterprise validators.
func NewExternalServicesStore() *db.ExternalServicesStore {
	return &db.ExternalServicesStore{
		GitHubValidators: []func(*schema.GitHubConnection) error{
			github.ValidateAuthz,
		},
		GitLabValidators: []func(*schema.GitLabConnection, []schema.AuthProviders) error{
			gitlab.ValidateAuthz,
		},
		BitbucketServerValidators: []func(*schema.BitbucketServerConnection) error{
			bitbucketserver.ValidateAuthz,
		},
	}
}
