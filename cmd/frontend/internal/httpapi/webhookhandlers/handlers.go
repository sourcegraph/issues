package webhookhandlers

import (
	"github.com/sourcegraph/sourcegraph/cmd/frontend/webhooks"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

func Init(db dbutil.DB, w *webhooks.GitHubWebhook) {
	w.Register(handleGitHubRepoAuthzEvent(db), "public")
	w.Register(handleGitHubRepoAuthzEvent(db), "repository")
	w.Register(handleGitHubRepoAuthzEvent(db), "member") // member has both users and repos
	w.Register(handleGitHubRepoAuthzEvent(db), "team_add")

	w.Register(handleGitHubUserAuthzEvent(db), "organisation")
	w.Register(handleGitHubUserAuthzEvent(db), "member") // member has both users and repos
	w.Register(handleGitHubUserAuthzEvent(db), "membership")
}
