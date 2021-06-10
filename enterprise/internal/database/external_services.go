package database

import (
	"github.com/sourcegraph/sourcegraph/internal/authz/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/authz/github"
	"github.com/sourcegraph/sourcegraph/internal/authz/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/authz/perforce"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/schema"
)

// InitExternalServicesStoreValidators returns an OSS database.ExternalServicesStore set with
// enterprise validators.
func InitExternalServicesStoreValidators() {
	database.GitHubValidators = []func(*schema.GitHubConnection) error{
		github.ValidateAuthz,
	}
	database.GitLabValidators = []func(*schema.GitLabConnection, []schema.AuthProviders) error{
		gitlab.ValidateAuthz,
	}
	database.BitbucketServerValidators = []func(*schema.BitbucketServerConnection) error{
		bitbucketserver.ValidateAuthz,
	}
	database.PerforceValidators = []func(connection *schema.PerforceConnection) error{
		perforce.ValidateAuthz,
	}
}
