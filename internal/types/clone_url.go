package types

import (
	"context"
	"fmt"
	"net/url"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/awscodecommit"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketcloud"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitolite"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/perforce"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/phabricator"
	"github.com/sourcegraph/sourcegraph/schema"
)

// TODO: GrantedScopes and RepoCloneURL don't feel like they should belong in
// this package. It causes our internal/types package to pull in code host
// specific packages like sourcegraph/sourcegraph/internal/extsvc/github.
// Ideally, they should belong in the extsvc package but that also doesn't work
// since that package can't imported from the code host specific sub-packages
// which as it leads to a cyclic import.

// GrantedScopes returns a slice of scopes granted by the service based on the token
// provided in the config.
//
// Currently only GitHub is supported.
func GrantedScopes(ctx context.Context, kind string, rawConfig string) ([]string, error) {
	if kind != extsvc.KindGitHub {
		return nil, fmt.Errorf("only GitHub supported")
	}
	config, err := extsvc.ParseConfig(kind, rawConfig)
	if err != nil {
		return nil, errors.Wrap(err, "parsing config")
	}
	switch v := config.(type) {
	case *schema.GitHubConnection:
		u, err := url.Parse(v.Url)
		if err != nil {
			return nil, errors.Wrap(err, "parsing URL")
		}
		client := github.NewV3Client(u, &auth.OAuthBearerToken{Token: v.Token}, nil)
		return client.GetAuthenticatedUserOAuthScopes(ctx)
	default:
		return nil, fmt.Errorf("unsupported config type: %T", v)
	}
}

// RepoCloneURL builds a cloneURL for the given repo based on the
// external service configuration.
// If authentication information is found in the configuration,
// it returns an authenticated URL for the selected code host.
func RepoCloneURL(kind, config string, repo *Repo) (string, error) {
	parsed, err := extsvc.ParseConfig(kind, config)
	if err != nil {
		return "", errors.Wrap(err, "loading service configuration")
	}

	switch t := parsed.(type) {
	case *schema.AWSCodeCommitConnection:
		if r, ok := repo.Metadata.(*awscodecommit.Repository); ok {
			return awsCodeCloneURL(r, t), nil
		}
	case *schema.BitbucketServerConnection:
		if r, ok := repo.Metadata.(*bitbucketserver.Repo); ok {
			return bitbucketServerCloneURL(r, t), nil
		}
	case *schema.BitbucketCloudConnection:
		if r, ok := repo.Metadata.(*bitbucketcloud.Repo); ok {
			return bitbucketCloudCloneURL(r, t), nil
		}
	case *schema.GitHubConnection:
		if r, ok := repo.Metadata.(*github.Repository); ok {
			return githubCloneURL(r, t)
		}
	case *schema.GitLabConnection:
		if r, ok := repo.Metadata.(*gitlab.Project); ok {
			return gitlabCloneURL(r, t), nil
		}
	case *schema.GitoliteConnection:
		if r, ok := repo.Metadata.(*gitolite.Repo); ok {
			return r.URL, nil
		}
	case *schema.PerforceConnection:
		if r, ok := repo.Metadata.(*perforce.Depot); ok {
			return perforceCloneURL(r, t), nil
		}
	case *schema.PhabricatorConnection:
		if r, ok := repo.Metadata.(*phabricator.Repo); ok {
			return phabricatorCloneURL(r, t), nil
		}
	case *schema.OtherExternalServiceConnection:
		if r, ok := repo.Metadata.(*OtherRepoMetadata); ok {
			return otherCloneURL(repo, r), nil
		}
	default:
		return "", errors.Errorf("unknown external service kind %q for repo %d", kind, repo.ID)
	}
	return "", errors.Errorf("unknown repo.Metadata type %T for repo %d", repo.Metadata, repo.ID)
}

func awsCodeCloneURL(repo *awscodecommit.Repository, cfg *schema.AWSCodeCommitConnection) string {
	u, err := url.Parse(repo.HTTPCloneURL)
	if err != nil {
		log15.Warn("Error adding authentication to AWS CodeCommit repository Git remote URL.", "url", repo.HTTPCloneURL, "error", err)
		return repo.HTTPCloneURL
	}

	username := cfg.GitCredentials.Username
	password := cfg.GitCredentials.Password

	u.User = url.UserPassword(username, password)
	return u.String()
}

func bitbucketServerCloneURL(repo *bitbucketserver.Repo, cfg *schema.BitbucketServerConnection) string {
	var cloneURL string
	for _, l := range repo.Links.Clone {
		if l.Name == "ssh" && cfg.GitURLType == "ssh" {
			cloneURL = l.Href
			break
		}
		if l.Name == "http" {
			var password string
			if cfg.Token != "" {
				password = cfg.Token // prefer personal access token
			} else {
				password = cfg.Password
			}
			cloneURL = setUserinfoBestEffort(l.Href, cfg.Username, password)
			// No break, so that we fallback to http in case of ssh missing
			// with GitURLType == "ssh"
		}
	}

	return cloneURL
}

// bitbucketCloudCloneURL returns the repository's Git remote URL with the configured
// Bitbucket Cloud app password inserted in the URL userinfo.
func bitbucketCloudCloneURL(repo *bitbucketcloud.Repo, cfg *schema.BitbucketCloudConnection) string {
	if cfg.GitURLType == "ssh" {
		return fmt.Sprintf("git@%s:%s.git", cfg.Url, repo.FullName)
	}

	fallbackURL := (&url.URL{
		Scheme: "https",
		Host:   cfg.Url,
		Path:   "/" + repo.FullName,
	}).String()

	httpsURL, err := repo.Links.Clone.HTTPS()
	if err != nil {
		log15.Warn("Error adding authentication to Bitbucket Cloud repository Git remote URL.", "url", repo.Links.Clone, "error", err)
		return fallbackURL
	}
	u, err := url.Parse(httpsURL)
	if err != nil {
		log15.Warn("Error adding authentication to Bitbucket Cloud repository Git remote URL.", "url", httpsURL, "error", err)
		return fallbackURL
	}

	u.User = url.UserPassword(cfg.Username, cfg.AppPassword)
	return u.String()
}

func githubCloneURL(repo *github.Repository, cfg *schema.GitHubConnection) (string, error) {
	if cfg.GitURLType == "ssh" {
		baseURL, err := url.Parse(cfg.Url)
		if err != nil {
			return "", err
		}
		baseURL = extsvc.NormalizeBaseURL(baseURL)
		originalHostname := baseURL.Hostname()
		url := fmt.Sprintf("git@%s:%s.git", originalHostname, repo.NameWithOwner)
		return url, nil
	}

	if cfg.Token == "" {
		return repo.URL, nil
	}
	u, err := url.Parse(repo.URL)
	if err != nil {
		log15.Warn("Error adding authentication to GitHub repository Git remote URL.", "url", repo.URL, "error", err)
		return repo.URL, nil
	}
	u.User = url.User(cfg.Token)
	return u.String(), nil
}

// authenticatedRemoteURL returns the GitLab projects's Git remote URL with the
// configured GitLab personal access token inserted in the URL userinfo.
func gitlabCloneURL(repo *gitlab.Project, cfg *schema.GitLabConnection) string {
	if cfg.GitURLType == "ssh" {
		return repo.SSHURLToRepo // SSH authentication must be provided out-of-band
	}
	if cfg.Token == "" {
		return repo.HTTPURLToRepo
	}
	u, err := url.Parse(repo.HTTPURLToRepo)
	if err != nil {
		log15.Warn("Error adding authentication to GitLab repository Git remote URL.", "url", repo.HTTPURLToRepo, "error", err)
		return repo.HTTPURLToRepo
	}
	// Any username works; "git" is not special.
	u.User = url.UserPassword("git", cfg.Token)
	return u.String()
}

// perforceCloneURL composes a clone URL for a Perforce depot based on
// given information. e.g.
// perforce://admin:password@ssl:111.222.333.444:1666//Sourcegraph/
func perforceCloneURL(depot *perforce.Depot, cfg *schema.PerforceConnection) string {
	cloneURL := url.URL{
		Scheme: "perforce",
		User:   url.UserPassword(cfg.P4User, cfg.P4Passwd),
		Host:   cfg.P4Port,
		Path:   depot.Depot,
	}
	return cloneURL.String()
}

func phabricatorCloneURL(repo *phabricator.Repo, _ *schema.PhabricatorConnection) string {
	var external []*phabricator.URI
	builtin := make(map[string]*phabricator.URI)

	for _, u := range repo.URIs {
		if u.Disabled || u.Normalized == "" {
			continue
		} else if u.BuiltinIdentifier != "" {
			builtin[u.BuiltinProtocol+"+"+u.BuiltinIdentifier] = u
		} else {
			external = append(external, u)
		}
	}

	var name string
	if len(external) > 0 {
		name = external[0].Normalized
	}

	var cloneURL string
	for _, alt := range [...]struct {
		protocol, identifier string
	}{ // Ordered by priority.
		{"https", "shortname"},
		{"https", "callsign"},
		{"https", "id"},
		{"ssh", "shortname"},
		{"ssh", "callsign"},
		{"ssh", "id"},
	} {
		if u, ok := builtin[alt.protocol+"+"+alt.identifier]; ok {
			cloneURL = u.Effective
			// TODO(tsenart): Authenticate the cloneURL with the user's
			// VCS password once we have that setting in the config. The
			// Conduit token can't be used for cloning.
			// cloneURL = setUserinfoBestEffort(cloneURL, conn.VCSPassword, "")

			if name == "" {
				name = u.Normalized
			}
		}
	}

	if cloneURL == "" {
		log15.Warn("unable to construct clone URL for repo", "name", name, "phabricator_id", repo.PHID)
	}

	return cloneURL
}

// TODO: this will be moved to the right package once we refactor the RepoCloneURL function.
type OtherRepoMetadata struct {
	// RelativePath is relative to ServiceID which is usually the host URL.
	// Joining them gives you the clone url.
	RelativePath string
}

func otherCloneURL(repo *Repo, m *OtherRepoMetadata) string {
	return repo.ExternalRepo.ServiceID + m.RelativePath
}

// setUserinfoBestEffort adds the username and password to rawurl. If user is
// not set in rawurl, username is used. If password is not set and there is a
// user, password is used. If anything fails, the original rawurl is returned.
func setUserinfoBestEffort(rawurl, username, password string) string {
	u, err := url.Parse(rawurl)
	if err != nil {
		return rawurl
	}

	passwordSet := password != ""

	// Update username and password if specified in rawurl
	if u.User != nil {
		if u.User.Username() != "" {
			username = u.User.Username()
		}
		if p, ok := u.User.Password(); ok {
			password = p
			passwordSet = true
		}
	}

	if username == "" {
		return rawurl
	}

	if passwordSet {
		u.User = url.UserPassword(username, password)
	} else {
		u.User = url.User(username)
	}

	return u.String()
}
