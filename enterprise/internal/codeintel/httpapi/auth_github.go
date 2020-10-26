package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
)

var githubURL = url.URL{Scheme: "https", Host: "api.github.com"}

func enforceAuthGithub(ctx context.Context, w http.ResponseWriter, r *http.Request, repoName string) (int, error) {
	nameWithOwner := strings.TrimPrefix(repoName, "github.com/")
	owner, name, err := github.SplitRepositoryNameWithOwner(nameWithOwner)
	if err != nil {
		return http.StatusNotFound, errors.New("invalid GitHub repository: nameWithOwner=" + nameWithOwner)
	}

	q := r.URL.Query()
	githubToken := q.Get("github_token")
	if githubToken == "" {
		return http.StatusUnauthorized, errors.New("must provide github_token")
	}

	client := github.NewClient(&githubURL, githubToken, nil)

	// There are 2 supported ways to authenticate the upload:
	//
	// 1. If the given token is a GitHub App installation token, then we use the
	//
	//    https://developer.github.com/v3/apps/installations/#list-repositories
	//
	//    endpoint to see if the associated GitHub App has been installed on the given repository.
	//
	//    One example of this is the built-in GITHUB_TOKEN in GitHub Actions:
	//
	//    https://help.github.com/en/actions/automating-your-workflow-with-github-actions/authenticating-with-the-github_token#about-the-github_token-secret
	//
	// 2. If the given token is a personal access token, then we use the
	//
	//    https://developer.github.com/v3/repos/#get
	//
	//    endpoint to see if the user has write access to the given repository.
	//
	// We don't know which kind of token was provided, so we try authenticating
	// the user via each in turn.

	authViaGithubApp := func() error {
		repos, err := client.ListInstallationRepositories(ctx)
		if err != nil {
			return err
		}
		for _, repo := range repos {
			if repo.NameWithOwner == nameWithOwner {
				return nil
			}
		}
		return fmt.Errorf("given repository %s not listed in installed repositories", nameWithOwner)
	}

	authViaReposEndpoint := func() error {
		repo, err := client.GetRepository(ctx, owner, name)
		if err != nil {
			return errors.Wrap(err, "unable to get repository permissions")
		}

		switch repo.ViewerPermission {
		case "ADMIN", "MAINTAIN", "WRITE":
			return nil
		default:
			return errors.New("you do not have write permission to the repository")
		}
	}

	err = nil
	var authErr error

	// Must try authenticating via GitHub App before the repos endpoint because
	// the repos endpoint always reports no permissions with a GitHub App
	// installation token.
	authErr = authViaGithubApp()
	if authErr == nil {
		return 0, nil
	}
	err = multierror.Append(err, authErr)

	authErr = authViaReposEndpoint()
	if authErr == nil {
		return 0, nil
	}
	err = multierror.Append(err, authErr)

	return http.StatusUnauthorized, err
}
