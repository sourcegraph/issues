package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
	"github.com/sourcegraph/sourcegraph/schema"
)

// handleGetGitolitePhabricatorMetadata serves the Gitolite
// Phabricator metadata endpoint, which returns the Phabricator
// metadata for a given repository by running a user-provided command.
func (s *Server) handleGetGitolitePhabricatorMetadata(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := func(name string) bool { _, ok := q[name]; return ok }
	switch {
	case query("gitolite"):
		gitoliteHost := q.Get("gitolite")
		repoName := q.Get("repo")

		config, err := conf.GitoliteConfigs(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Iterate through Gitolite hosts, searching for one that will return the Phabricator mapping
		for _, gconf := range config {
			if gconf.Host != gitoliteHost {
				continue
			}
			if gconf.Phabricator == nil {
				continue
			}
			callsign, err := getGitolitePhabCallsign(r.Context(), gconf, repoName, gconf.Phabricator.CallsignCommand)
			if err != nil {
				log15.Warn("failed to get Phabricator callsign", "host", gconf.Host, "repo", repoName, "err", err)
				continue
			}

			// Return the first valid mapping we find
			err = json.NewEncoder(w).Encode(protocol.GitolitePhabricatorMetadataResponse{
				Callsign: callsign,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
		// No matching Phabricator host found
		if err := json.NewEncoder(w).Encode(protocol.GitolitePhabricatorMetadataResponse{}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "unrecognized URL in /get request", http.StatusNotFound)
		return
	}
}

var callSignPattern = lazyregexp.New("^[A-Z]+$")

func getGitolitePhabCallsign(ctx context.Context, gconf *schema.GitoliteConnection, repo, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Env = append(os.Environ(), "REPO="+repo)
	stdout, err := cmd.Output()
	if err != nil {
		var e *exec.ExitError
		if errors.As(err, &e) {
			return "", errors.Errorf("Command failed: %s, stderr: %s", e, string(e.Stderr))
		}
		return "", errors.Errorf("Command failed: %s", err)
	}
	callsign := strings.TrimSpace(string(stdout))
	if !callSignPattern.MatchString(callsign) {
		return "", errors.Errorf("Callsign %q is invalid (must match `[A-Z]+`)", callsign)
	}
	return callsign, nil
}
