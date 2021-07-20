package gitolite

import (
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

func TestDecodeRepos(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		gitoliteInfo string
		expRepos     []*Repo
	}{
		{
			name: "with SCP host format",
			host: "git@gitolite.example.com",
			gitoliteInfo: `hello admin, this is git@gitolite-799486b5db-ghrxg running gitolite3 v3.6.6-0-g908f8c6 on git 2.7.4

		 R W    gitolite-admin
		 R W    repowith@sign
		 R W    testing
		`,
			expRepos: []*Repo{
				{Name: "gitolite-admin", URL: "git@gitolite.example.com:gitolite-admin"},
				{Name: "repowith@sign", URL: "git@gitolite.example.com:repowith@sign"},
				{Name: "testing", URL: "git@gitolite.example.com:testing"},
			},
		},
		{
			name: "with URL host format",
			host: "ssh://git@gitolite.example.com:2222/",
			gitoliteInfo: `hello admin, this is git@gitolite-799486b5db-ghrxg running gitolite3 v3.6.6-0-g908f8c6 on git 2.7.4

		 R W    gitolite-admin
		 R W    repowith@sign
		 R W    testing
		`,
			expRepos: []*Repo{
				{Name: "gitolite-admin", URL: "ssh://git@gitolite.example.com:2222/gitolite-admin"},
				{Name: "repowith@sign", URL: "ssh://git@gitolite.example.com:2222/repowith@sign"},
				{Name: "testing", URL: "ssh://git@gitolite.example.com:2222/testing"},
			},
		},
		{
			name:         "handles empty response",
			host:         "git@gitolite.example.com",
			gitoliteInfo: "",
			expRepos:     nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repos := decodeRepos(test.host, test.gitoliteInfo)
			if diff := cmp.Diff(repos, test.expRepos); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestMaybeUnauthorized(t *testing.T) {
	err := errors.New("random")
	if errcode.IsUnauthorized(maybeUnauthorized(err)) {
		t.Errorf("Should not be unauthorized")
	}

	err = errors.New("permission denied (public key)")
	if !errcode.IsUnauthorized(maybeUnauthorized(err)) {
		t.Errorf("Should be unauthorized")
	}
}
