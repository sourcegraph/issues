package graphqlbackend

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/rcache"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

type externalServiceResolver struct {
	db              dbutil.DB
	externalService *types.ExternalService
	warning         string

	webhookURLOnce sync.Once
	webhookURL     string
	webhookErr     error
}

const externalServiceIDKind = "ExternalService"

func externalServiceByID(ctx context.Context, db dbutil.DB, gqlID graphql.ID) (*externalServiceResolver, error) {
	id, err := unmarshalExternalServiceID(gqlID)
	if err != nil {
		return nil, err
	}

	es, err := database.ExternalServices(db).GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := checkExternalServiceAccess(ctx, db, es.NamespaceUserID); err != nil {
		return nil, err
	}
	return &externalServiceResolver{db: db, externalService: es}, nil
}

func marshalExternalServiceID(id int64) graphql.ID {
	return relay.MarshalID(externalServiceIDKind, id)
}

func unmarshalExternalServiceID(id graphql.ID) (externalServiceID int64, err error) {
	if kind := relay.UnmarshalKind(id); kind != externalServiceIDKind {
		err = errors.Errorf("expected graphql ID to have kind %q; got %q", externalServiceIDKind, kind)
		return
	}
	err = relay.UnmarshalSpec(id, &externalServiceID)
	return
}

func (r *externalServiceResolver) ID() graphql.ID {
	return marshalExternalServiceID(r.externalService.ID)
}

func (r *externalServiceResolver) Kind() string {
	return r.externalService.Kind
}

func (r *externalServiceResolver) DisplayName() string {
	return r.externalService.DisplayName
}

func (r *externalServiceResolver) Config() (JSONCString, error) {
	err := r.externalService.RedactConfigSecrets()
	if err != nil {
		return "", err
	}
	return JSONCString(r.externalService.Config), nil
}

func (r *externalServiceResolver) CreatedAt() DateTime {
	return DateTime{Time: r.externalService.CreatedAt}
}

func (r *externalServiceResolver) UpdatedAt() DateTime {
	return DateTime{Time: r.externalService.UpdatedAt}
}

func (r *externalServiceResolver) Namespace(ctx context.Context) (*NamespaceResolver, error) {
	if r.externalService.NamespaceUserID == 0 {
		return nil, nil
	}
	userID := MarshalUserID(r.externalService.NamespaceUserID)
	n, err := NamespaceByID(ctx, r.db, userID)
	if err != nil {
		return nil, err
	}
	return &NamespaceResolver{n}, nil
}

func (r *externalServiceResolver) WebhookURL() (*string, error) {
	r.webhookURLOnce.Do(func() {
		parsed, err := extsvc.ParseConfig(r.externalService.Kind, r.externalService.Config)
		if err != nil {
			r.webhookErr = errors.Wrap(err, "parsing external service config")
			return
		}
		u := extsvc.WebhookURL(r.externalService.Kind, r.externalService.ID, conf.ExternalURL())
		switch c := parsed.(type) {
		case *schema.BitbucketServerConnection:
			if c.Webhooks != nil {
				r.webhookURL = u
			}
			if c.Plugin != nil && c.Plugin.Webhooks != nil {
				r.webhookURL = u
			}
		case *schema.GitHubConnection:
			if len(c.Webhooks) > 0 {
				r.webhookURL = u
			}
		case *schema.GitLabConnection:
			if len(c.Webhooks) > 0 {
				r.webhookURL = u
			}
		}
	})
	if r.webhookURL == "" {
		return nil, r.webhookErr
	}
	return &r.webhookURL, r.webhookErr
}

func (r *externalServiceResolver) Warning() *string {
	if r.warning == "" {
		return nil
	}
	return &r.warning
}

func (r *externalServiceResolver) LastSyncError(ctx context.Context) (*string, error) {
	latestError, err := database.ExternalServices(r.db).GetLastSyncError(ctx, r.externalService.ID)
	if err != nil {
		return nil, err
	}
	if latestError == "" {
		return nil, nil
	}
	return &latestError, nil
}

func (r *externalServiceResolver) RepoCount(ctx context.Context) (int32, error) {
	return database.ExternalServices(r.db).RepoCount(ctx, r.externalService.ID)
}

func (r *externalServiceResolver) LastSyncAt() *DateTime {
	if r.externalService.LastSyncAt.IsZero() {
		return nil
	}
	return &DateTime{Time: r.externalService.LastSyncAt}
}

func (r *externalServiceResolver) NextSyncAt() *DateTime {
	if r.externalService.NextSyncAt.IsZero() {
		return nil
	}
	return &DateTime{Time: r.externalService.NextSyncAt}
}

var scopeCache = rcache.New("extsvc_token_scope")

func (r *externalServiceResolver) GrantedScopes(ctx context.Context) (*[]string, error) {
	scopes, err := repos.GrantedScopes(ctx, scopeCache, r.externalService)
	if err != nil {
		// It's possible that we fail to fetch scope from the code host, in this case we
		// don't want the entire resolver to fail.
		log15.Error("Getting service scope", "id", r.externalService.ID, "error", err)
		return nil, nil
	}
	if scopes == nil {
		return nil, nil
	}
	return &scopes, nil
}
