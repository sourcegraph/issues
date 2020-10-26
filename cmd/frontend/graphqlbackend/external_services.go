package graphqlbackend

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
)

var extsvcConfigAllowEdits, _ = strconv.ParseBool(env.Get("EXTSVC_CONFIG_ALLOW_EDITS", "false", "When EXTSVC_CONFIG_FILE is in use, allow edits in the application to be made which will be overwritten on next process restart"))

type addExternalServiceArgs struct {
	Input addExternalServiceInput
}

type addExternalServiceInput struct {
	Kind        string
	DisplayName string
	Config      string
	Namespace   *graphql.ID
}

func (r *schemaResolver) AddExternalService(ctx context.Context, args *addExternalServiceArgs) (*externalServiceResolver, error) {
	if os.Getenv("EXTSVC_CONFIG_FILE") != "" && !extsvcConfigAllowEdits {
		return nil, errors.New("adding external service not allowed when using EXTSVC_CONFIG_FILE")
	}

	// 🚨 SECURITY: Only site admins may add external services if user mode is disabled.
	namespaceUserID := int32(0)
	isSiteAdmin := backend.CheckCurrentUserIsSiteAdmin(ctx) == nil
	allowUserExternalServices := conf.ExternalServiceUserMode()
	if !allowUserExternalServices {
		// The user may have a tag that opts them in
		err := backend.CheckActorHasTag(ctx, backend.TagAllowUserExternalServicePublic)
		allowUserExternalServices = err == nil
	}
	if args.Input.Namespace != nil {
		if !allowUserExternalServices {
			return nil, errors.New("allow users to add external services is not enabled")
		}

		var err error
		switch relay.UnmarshalKind(*args.Input.Namespace) {
		case "User":
			err = relay.UnmarshalSpec(*args.Input.Namespace, &namespaceUserID)
		default:
			err = errors.Errorf("invalid namespace %q", *args.Input.Namespace)
		}

		if err != nil {
			return nil, err
		}

		if namespaceUserID != actor.FromContext(ctx).UID {
			return nil, errors.New("the namespace is not same as the authenticated user")
		}

	} else if !isSiteAdmin {
		return nil, backend.ErrMustBeSiteAdmin
	}

	externalService := &types.ExternalService{
		Kind:        args.Input.Kind,
		DisplayName: args.Input.DisplayName,
		Config:      args.Input.Config,
	}
	if namespaceUserID > 0 {
		externalService.NamespaceUserID = &namespaceUserID
	}

	if err := db.ExternalServices.Create(ctx, conf.Get, externalService); err != nil {
		return nil, err
	}

	res := &externalServiceResolver{externalService: externalService}
	if err := syncExternalService(ctx, externalService); err != nil {
		res.warning = fmt.Sprintf("External service created, but we encountered a problem while validating the external service: %s", err)
	}

	return res, nil
}

type updateExternalServiceArgs struct {
	Input updateExternalServiceInput
}

type updateExternalServiceInput struct {
	ID          graphql.ID
	DisplayName *string
	Config      *string
}

func (*schemaResolver) UpdateExternalService(ctx context.Context, args *updateExternalServiceArgs) (*externalServiceResolver, error) {
	if os.Getenv("EXTSVC_CONFIG_FILE") != "" && !extsvcConfigAllowEdits {
		return nil, errors.New("updating external service not allowed when using EXTSVC_CONFIG_FILE")
	}

	id, err := unmarshalExternalServiceID(args.Input.ID)
	if err != nil {
		return nil, err
	}

	es, err := db.ExternalServices.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 🚨 SECURITY: Only site admins may update all or a user's external services.
	// Otherwise, the authenticated user can only update external services under the same namespace.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		if es.NamespaceUserID == nil {
			return nil, err
		} else if actor.FromContext(ctx).UID != *es.NamespaceUserID {
			return nil, errors.New("the authenticated user does not have access to this external service")
		}
	}

	if args.Input.Config != nil && strings.TrimSpace(*args.Input.Config) == "" {
		return nil, errors.New("blank external service configuration is invalid (must be valid JSONC)")
	}

	ps := conf.Get().AuthProviders
	update := &db.ExternalServiceUpdate{
		DisplayName: args.Input.DisplayName,
		Config:      args.Input.Config,
	}
	if err := db.ExternalServices.Update(ctx, ps, id, update); err != nil {
		return nil, err
	}

	// Fetch from database again to get all fields with updated values.
	es, err = db.ExternalServices.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	res := &externalServiceResolver{externalService: es}
	if err = syncExternalService(ctx, es); err != nil {
		res.warning = fmt.Sprintf("External service updated, but we encountered a problem while validating the external service: %s", err)
	}

	return res, nil
}

// Eagerly trigger a repo-updater sync.
func syncExternalService(ctx context.Context, svc *types.ExternalService) error {
	// Only give 5s to validate external service sync. Usually if there is a
	// problem it fails sooner.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := repoupdater.DefaultClient.SyncExternalService(ctx, api.ExternalService{
		ID:              svc.ID,
		Kind:            svc.Kind,
		DisplayName:     svc.DisplayName,
		Config:          svc.Config,
		CreatedAt:       svc.CreatedAt,
		UpdatedAt:       svc.UpdatedAt,
		DeletedAt:       svc.DeletedAt,
		LastSyncAt:      svc.LastSyncAt,
		NextSyncAt:      svc.NextSyncAt,
		NamespaceUserID: svc.NamespaceUserID,
	})
	if err != nil && ctx.Err() == nil {
		return err
	}

	return nil
}

type deleteExternalServiceArgs struct {
	ExternalService graphql.ID
}

func (*schemaResolver) DeleteExternalService(ctx context.Context, args *deleteExternalServiceArgs) (*EmptyResponse, error) {
	if os.Getenv("EXTSVC_CONFIG_FILE") != "" && !extsvcConfigAllowEdits {
		return nil, errors.New("deleting external service not allowed when using EXTSVC_CONFIG_FILE")
	}

	id, err := unmarshalExternalServiceID(args.ExternalService)
	if err != nil {
		return nil, err
	}

	es, err := db.ExternalServices.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 🚨 SECURITY: Only site admins may delete all or a user's external services.
	// Otherwise, the authenticated user can only delete external services under the same namespace.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		if es.NamespaceUserID == nil {
			return nil, err
		} else if actor.FromContext(ctx).UID != *es.NamespaceUserID {
			return nil, errors.New("the authenticated user does not have access to this external service")
		}
	}

	if err := db.ExternalServices.Delete(ctx, id); err != nil {
		return nil, err
	}
	now := time.Now()
	es.DeletedAt = &now

	// The user doesn't care if triggering syncing failed when deleting a
	// service, so kick off in the background.
	go func() {
		_ = syncExternalService(context.Background(), es)
	}()

	return &EmptyResponse{}, nil
}

type ExternalServicesArgs struct {
	Namespace *graphql.ID
	graphqlutil.ConnectionArgs
	After *string
}

var errMustBeSiteAdminOrSameUser = errors.New("must be site admin or the namespace is same as the authenticated user")

func (r *schemaResolver) ExternalServices(ctx context.Context, args *ExternalServicesArgs) (*externalServiceConnectionResolver, error) {
	var namespaceUserID int32
	if args.Namespace != nil {
		var err error
		switch relay.UnmarshalKind(*args.Namespace) {
		case "User":
			err = relay.UnmarshalSpec(*args.Namespace, &namespaceUserID)
		default:
			err = errors.Errorf("invalid namespace %q", *args.Namespace)
		}

		if err != nil {
			return nil, err
		}
	}

	// 🚨 SECURITY: Only site admins may read all or a user's external services.
	// Otherwise, the authenticated user can only read external services under the same namespace.
	if backend.CheckSiteAdminOrSameUser(ctx, namespaceUserID) != nil {
		// NOTE: We do not directly return the err here because it contains the desired username,
		// which then allows attacker to brute force over our database ID and get corresponding
		// username.
		return nil, errMustBeSiteAdminOrSameUser
	}

	var afterID int64
	if args.After != nil {
		var err error
		afterID, err = unmarshalExternalServiceID(graphql.ID(*args.After))
		if err != nil {
			return nil, err
		}
	}

	opt := db.ExternalServicesListOptions{
		NamespaceUserID: namespaceUserID,
		AfterID:         afterID,
	}
	args.ConnectionArgs.Set(&opt.LimitOffset)
	return &externalServiceConnectionResolver{opt: opt}, nil
}

type externalServiceConnectionResolver struct {
	opt db.ExternalServicesListOptions

	// cache results because they are used by multiple fields
	once             sync.Once
	externalServices []*types.ExternalService
	err              error
}

func (r *externalServiceConnectionResolver) compute(ctx context.Context) ([]*types.ExternalService, error) {
	r.once.Do(func() {
		r.externalServices, r.err = db.ExternalServices.List(ctx, r.opt)
	})
	return r.externalServices, r.err
}

func (r *externalServiceConnectionResolver) Nodes(ctx context.Context) ([]*externalServiceResolver, error) {
	externalServices, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}
	resolvers := make([]*externalServiceResolver, 0, len(externalServices))
	for _, externalService := range externalServices {
		resolvers = append(resolvers, &externalServiceResolver{externalService: externalService})
	}
	return resolvers, nil
}

func (r *externalServiceConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	// Reset pagination cursor to get correct total count
	opt := r.opt
	opt.AfterID = 0
	count, err := db.ExternalServices.Count(ctx, opt)
	return int32(count), err
}

func (r *externalServiceConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	externalServices, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}

	// We would have had all results when no limit set
	if r.opt.LimitOffset == nil {
		return graphqlutil.HasNextPage(false), nil
	}

	// We got less results than limit, means we've had all results
	if len(externalServices) < r.opt.Limit {
		return graphqlutil.HasNextPage(false), nil
	}

	// In case the number of results happens to be the same as the limit,
	// we need another query to get accurate total count with same cursor
	// to determine if there are more results than the limit we set.
	count, err := db.ExternalServices.Count(ctx, r.opt)
	if err != nil {
		return nil, err
	}

	if count > len(externalServices) {
		endCursorID := externalServices[len(externalServices)-1].ID
		return graphqlutil.NextPageCursor(string(marshalExternalServiceID(endCursorID))), nil
	}
	return graphqlutil.HasNextPage(false), nil
}

type computedExternalServiceConnectionResolver struct {
	args             graphqlutil.ConnectionArgs
	externalServices []*types.ExternalService
}

func (r *computedExternalServiceConnectionResolver) Nodes(ctx context.Context) []*externalServiceResolver {
	svcs := r.externalServices
	if r.args.First != nil && int(*r.args.First) < len(svcs) {
		svcs = svcs[:*r.args.First]
	}
	resolvers := make([]*externalServiceResolver, 0, len(svcs))
	for _, svc := range svcs {
		resolvers = append(resolvers, &externalServiceResolver{externalService: svc})
	}
	return resolvers
}

func (r *computedExternalServiceConnectionResolver) TotalCount(ctx context.Context) int32 {
	return int32(len(r.externalServices))
}

func (r *computedExternalServiceConnectionResolver) PageInfo(ctx context.Context) *graphqlutil.PageInfo {
	return graphqlutil.HasNextPage(r.args.First != nil && len(r.externalServices) >= int(*r.args.First))
}
