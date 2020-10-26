// +build gqltest

package main

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/gqltestutil"
)

func TestExternalService(t *testing.T) {
	if len(*githubToken) == 0 {
		t.Skip("Environment variable GITHUB_TOKEN is not set")
	}

	t.Run("repositoryPathPattern", func(t *testing.T) {
		const repo = "sgtest/go-diff" // Tiny repo, fast to clone
		const slug = "github.com/" + repo
		// Set up external service
		esID, err := client.AddExternalService(gqltestutil.AddExternalServiceInput{
			Kind:        extsvc.KindGitHub,
			DisplayName: "gqltest-github-repoPathPattern",
			Config: mustMarshalJSONString(struct {
				URL                   string   `json:"url"`
				Token                 string   `json:"token"`
				Repos                 []string `json:"repos"`
				RepositoryPathPattern string   `json:"repositoryPathPattern"`
			}{
				URL:                   "http://github.com",
				Token:                 *githubToken,
				Repos:                 []string{repo},
				RepositoryPathPattern: "foobar/{host}/{nameWithOwner}",
			}),
		})
		// The repo-updater might not be up yet but it will eventually catch up for the external
		// service we just added, thus it is OK to ignore this transient error.
		if err != nil && !strings.Contains(err.Error(), "/sync-external-service") {
			t.Fatal(err)
		}
		defer func() {
			err := client.DeleteExternalService(esID)
			if err != nil {
				t.Fatal(err)
			}
		}()

		err = client.WaitForReposToBeCloned("foobar/" + slug)
		if err != nil {
			t.Fatal(err)
		}

		// The request URL should be redirected to the new path
		origURL := *baseURL + "/" + slug
		resp, err := client.Get(origURL)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()

		wantURL := *baseURL + "/foobar/" + slug // <baseURL>/foobar/github.com/sgtest/go-diff
		if diff := cmp.Diff(wantURL, resp.Request.URL.String()); diff != "" {
			t.Fatalf("URL mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestExternalService_AWSCodeCommit(t *testing.T) {
	if len(*awsAccessKeyID) == 0 || len(*awsSecretAccessKey) == 0 ||
		len(*awsCodeCommitUsername) == 0 || len(*awsCodeCommitPassword) == 0 {
		t.Skip("Environment variable AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_CODE_COMMIT_USERNAME or AWS_CODE_COMMIT_PASSWORD is not set")
	}

	// Set up external service
	esID, err := client.AddExternalService(gqltestutil.AddExternalServiceInput{
		Kind:        extsvc.KindAWSCodeCommit,
		DisplayName: "gqltest-aws-code-commit",
		Config: mustMarshalJSONString(struct {
			Region                string            `json:"region"`
			AccessKeyID           string            `json:"accessKeyID"`
			SecretAccessKey       string            `json:"secretAccessKey"`
			RepositoryPathPattern string            `json:"repositoryPathPattern"`
			GitCredentials        map[string]string `json:"gitCredentials"`
		}{
			Region:                "us-west-1",
			AccessKeyID:           *awsAccessKeyID,
			SecretAccessKey:       *awsSecretAccessKey,
			RepositoryPathPattern: "aws/{name}",
			GitCredentials: map[string]string{
				"username": *awsCodeCommitUsername,
				"password": *awsCodeCommitPassword,
			},
		}),
	})
	// The repo-updater might not be up yet but it will eventually catch up for the external
	// service we just added, thus it is OK to ignore this transient error.
	if err != nil && !strings.Contains(err.Error(), "/sync-external-service") {
		t.Fatal(err)
	}
	defer func() {
		err := client.DeleteExternalService(esID)
		if err != nil {
			t.Fatal(err)
		}
	}()

	const repoName = "aws/test"
	err = client.WaitForReposToBeCloned(repoName)
	if err != nil {
		t.Fatal(err)
	}

	blob, err := client.GitBlob(repoName, "master", "README")
	if err != nil {
		t.Fatal(err)
	}

	wantBlob := "README\n\nchange"
	if diff := cmp.Diff(wantBlob, blob); diff != "" {
		t.Fatalf("Blob mismatch (-want +got):\n%s", diff)
	}
}

func TestExternalService_BitbucketServer(t *testing.T) {
	if len(*bbsURL) == 0 || len(*bbsToken) == 0 || len(*bbsUsername) == 0 {
		t.Skip("Environment variable BITBUCKET_SERVER_URL, BITBUCKET_SERVER_TOKEN, or BITBUCKET_SERVER_USERNAME is not set")
	}

	// Set up external service
	esID, err := client.AddExternalService(gqltestutil.AddExternalServiceInput{
		Kind:        extsvc.KindBitbucketServer,
		DisplayName: "gqltest-bitbucket-server",
		Config: mustMarshalJSONString(struct {
			URL                   string   `json:"url"`
			Token                 string   `json:"token"`
			Username              string   `json:"username"`
			Repos                 []string `json:"repos"`
			RepositoryPathPattern string   `json:"repositoryPathPattern"`
		}{
			URL:                   *bbsURL,
			Token:                 *bbsToken,
			Username:              *bbsUsername,
			Repos:                 []string{"SOURCEGRAPH/jsonrpc2"},
			RepositoryPathPattern: "bbs/{projectKey}/{repositorySlug}",
		}),
	})
	// The repo-updater might not be up yet but it will eventually catch up for the external
	// service we just added, thus it is OK to ignore this transient error.
	if err != nil && !strings.Contains(err.Error(), "/sync-external-service") {
		t.Fatal(err)
	}
	defer func() {
		err := client.DeleteExternalService(esID)
		if err != nil {
			t.Fatal(err)
		}
	}()

	const repoName = "bbs/SOURCEGRAPH/jsonrpc2"
	err = client.WaitForReposToBeCloned(repoName)
	if err != nil {
		t.Fatal(err)
	}

	blob, err := client.GitBlob(repoName, "master", ".travis.yml")
	if err != nil {
		t.Fatal(err)
	}

	wantBlob := "language: go\ngo: \n - 1.x\n\nscript:\n - go test -race -v ./...\n"
	if diff := cmp.Diff(wantBlob, blob); diff != "" {
		t.Fatalf("Blob mismatch (-want +got):\n%s", diff)
	}
}
