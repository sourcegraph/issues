package enterprise

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/webhooks"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
)

// Services is a bag of HTTP handlers and factory functions that are registered by the
// enterprise frontend setup hook.
type Services struct {
	GitHubWebhook             webhooks.Registerer
	GitLabWebhook             http.Handler
	BitbucketServerWebhook    http.Handler
	NewCodeIntelUploadHandler NewCodeIntelUploadHandler
	NewExecutorProxyHandler   NewExecutorProxyHandler
	AuthzResolver             graphqlbackend.AuthzResolver
	BatchChangesResolver      graphqlbackend.BatchChangesResolver
	CodeIntelResolver         graphqlbackend.CodeIntelResolver
	InsightsResolver          graphqlbackend.InsightsResolver
	CodeMonitorsResolver      graphqlbackend.CodeMonitorsResolver
	LicenseResolver           graphqlbackend.LicenseResolver
	DotcomResolver            graphqlbackend.DotcomRootResolver
}

// NewCodeIntelUploadHandler creates a new handler for the LSIF upload endpoint. The
// resulting handler skips auth checks when the internal flag is true.
type NewCodeIntelUploadHandler func(internal bool) http.Handler

// NewExecutorProxyHandler creates a new proxy handler for routes accessible to the
// executor services deployed separately from the k8s cluster. This handler is protected
// via a shared username and password.
type NewExecutorProxyHandler func() http.Handler

// DefaultServices creates a new Services value that has default implementations for all services.
func DefaultServices() Services {
	return Services{
		GitHubWebhook:             registerFunc(func(webhook *webhooks.GitHubWebhook) {}),
		GitLabWebhook:             makeNotFoundHandler("gitlab webhook"),
		BitbucketServerWebhook:    makeNotFoundHandler("bitbucket server webhook"),
		NewCodeIntelUploadHandler: func(_ bool) http.Handler { return makeNotFoundHandler("code intel upload") },
		NewExecutorProxyHandler:   func() http.Handler { return makeNotFoundHandler("executor proxy") },
	}
}

// makeNotFoundHandler returns an HTTP handler that respond 404 for all requests.
func makeNotFoundHandler(handlerName string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(fmt.Sprintf("%s is only available in enterprise", handlerName)))
	})
}

type registerFunc func(webhook *webhooks.GitHubWebhook)

func (fn registerFunc) Register(w *webhooks.GitHubWebhook) {
	fn(w)
}

type ErrBatchChangesDisabledDotcom struct{}

func (e ErrBatchChangesDisabledDotcom) Error() string {
	return "access to batch changes on Sourcegraph.com is currently not available"
}

type ErrBatchChangesDisabled struct{}

func (e ErrBatchChangesDisabled) Error() string {
	return "batch changes are disabled. Ask a site admin to set 'batchChanges.enabled' in the site configuration to enable the feature."
}

type ErrBatchChangesDisabledForUser struct{}

func (e ErrBatchChangesDisabledForUser) Error() string {
	return "batch changes are disabled for non-site-admin users. Ask a site admin to unset 'batchChanges.restrictToAdmins' in the site configuration to enable the feature for all users."
}

// Checks if Batch Changes are enabled at the site-level and returns `nil` if they are, or
// else an error indicating why they're disabled
func BatchChangesEnabledForSite() error {
	if !conf.BatchChangesEnabled() {
		return ErrBatchChangesDisabled{}
	}

	// Batch Changes are disabled on sourcegraph.com
	if envvar.SourcegraphDotComMode() {
		return ErrBatchChangesDisabledDotcom{}
	}

	return nil
}

// Checks if Batch Changes are enabled for the current user and returns `nil` if they are,
// or else an error indicating why they're disabled
func BatchChangesEnabledForUser(ctx context.Context, db dbutil.DB) error {
	if err := BatchChangesEnabledForSite(); err != nil {
		return err
	}

	if conf.BatchChangesRestrictedToAdmins() && backend.CheckCurrentUserIsSiteAdmin(ctx, db) != nil {
		return ErrBatchChangesDisabledForUser{}
	}
	return nil
}
