package gitolite

import (
	"context"
	"net/url"
	"os/exec"
	"strings"

	"github.com/inconshreveable/log15"
)

// Repo is the repository metadata returned by the Gitolite API.
type Repo struct {
	// Name is the name of the repository as it is returned by `ssh git@GITOLITE_HOST info`
	Name string

	// URL is the clone URL of the repository.
	URL string
}

// Client is a client for the Gitolite API.
//
// IMPORTANT: in order to authenticate to the Gitolite API, the client must be invoked from a
// service in an environment that contains a Gitolite-authorized SSH key. As of writing, only
// gitserver meets this criterion (i.e., only invoke this client from gitserver).
//
// Impl note: To change the above, remove the invocation of the `ssh` binary and replace it
// with use of the `ssh` package, reading arguments from config.
type Client struct {
	Host string
}

func NewClient(host string) *Client {
	return &Client{Host: host}
}

func (c *Client) ListRepos(ctx context.Context) ([]*Repo, error) {
	out, err := exec.CommandContext(ctx, "ssh", c.Host, "info").Output()
	if err != nil {
		log15.Error("listing gitolite failed", "error", err, "out", string(out))
		return nil, err
	}
	return decodeRepos(c.Host, string(out)), nil
}

func decodeRepos(host, gitoliteInfo string) []*Repo {
	lines := strings.Split(gitoliteInfo, "\n")
	var repos []*Repo
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "R" {
			continue
		}
		name := fields[len(fields)-1]
		if len(fields) >= 2 && fields[0] == "R" {
			repo := &Repo{Name: name}

			u, err := url.Parse(host)
			// see https://github.com/sourcegraph/security-issues/issues/97
			if err != nil {
				continue
			}

			// We support both URL and SCP formats
			// url: ssh://git@github.com:22/tsenart/vegeta
			// scp: git@github.com:tsenart/vegeta
			if u == nil || u.Scheme == "" {
				repo.URL = host + ":" + name
			} else if u.Scheme == "ssh" {
				u.Path = name
				repo.URL = u.String()
			}

			repos = append(repos, repo)
		}
	}

	return repos
}
