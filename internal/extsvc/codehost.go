package extsvc

import (
	"net/url"
	"strings"

	"github.com/sourcegraph/sourcegraph/internal/api"
)

type CodeHost struct {
	ServiceID   string
	ServiceType string
	BaseURL     *url.URL
}

// Known public code hosts and their URLs
var (
	GitHubDotComURL = mustParseURL("https://github.com")
	GitHubDotCom    = NewCodeHost(GitHubDotComURL, TypeGitHub)

	GitLabDotComURL = mustParseURL("https://gitlab.com")
	GitLabDotCom    = NewCodeHost(GitLabDotComURL, TypeGitLab)

	MavenURL = &url.URL{Host: "maven"}
	Maven    = NewCodeHost(MavenURL, TypeMaven)

	PublicCodeHosts = []*CodeHost{
		GitHubDotCom,
		GitLabDotCom,
		Maven,
	}
)

func NewCodeHost(baseURL *url.URL, serviceType string) *CodeHost {
	return &CodeHost{
		ServiceID:   NormalizeBaseURL(baseURL).String(),
		ServiceType: serviceType,
		BaseURL:     baseURL,
	}
}

// IsHostOfRepo returns true if the repository belongs to given code host.
func IsHostOfRepo(c *CodeHost, repo *api.ExternalRepoSpec) bool {
	return c.ServiceID == repo.ServiceID && c.ServiceType == repo.ServiceType
}

// IsHostOfAccount returns true if the account belongs to given code host.
func IsHostOfAccount(c *CodeHost, account *Account) bool {
	return c.ServiceID == account.ServiceID && c.ServiceType == account.ServiceType
}

// NormalizeBaseURL modifies the input and returns a normalized form of the a base URL with insignificant
// differences (such as in presence of a trailing slash, or hostname case) eliminated. Its return value should be
// used for the (ExternalRepoSpec).ServiceID field (and passed to XyzExternalRepoSpec) instead of a non-normalized
// base URL.
func NormalizeBaseURL(baseURL *url.URL) *url.URL {
	baseURL.Host = strings.ToLower(baseURL.Host)
	if !strings.HasSuffix(baseURL.Path, "/") {
		baseURL.Path += "/"
	}
	return baseURL
}

// CodeHostOf returns the CodeHost of the given repo, if any, as
// determined by a common prefix between the repo name and the
// code hosts' URL hostname component.
func CodeHostOf(name api.RepoName, codehosts ...*CodeHost) *CodeHost {
	for _, c := range codehosts {
		if strings.HasPrefix(strings.ToLower(string(name)), c.BaseURL.Hostname()) {
			return c
		}
	}
	return nil
}

func mustParseURL(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return u
}
