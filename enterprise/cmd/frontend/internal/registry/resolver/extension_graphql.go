package resolver

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/registry/service"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/registry/store"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/registry/types"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

// RegistryExtensionID identifies a registry extension, either locally or on a remote
// registry. Exactly 1 field must be set.
type RegistryExtensionID struct {
	LocalID  int32                      `json:"l,omitempty"`
	RemoteID *registryExtensionRemoteID `json:"r,omitempty"`
}

func MarshalRegistryExtensionID(id RegistryExtensionID) graphql.ID {
	return relay.MarshalID("RegistryExtension", id)
}

func UnmarshalRegistryExtensionID(id graphql.ID) (registryExtensionID RegistryExtensionID, err error) {
	err = relay.UnmarshalSpec(id, &registryExtensionID)
	return
}

// RegistryExtensionByIDInt32 looks up and returns the registry extension in the database with the
// given ID. If no such extension exists, an error is returned. The func is nil when there is no
// local registry.
var RegistryExtensionByIDInt32 func(context.Context, dbutil.DB, int32) (graphqlbackend.RegistryExtension, error)

func registryExtensionByID(ctx context.Context, db dbutil.DB, id graphql.ID) (graphqlbackend.RegistryExtension, error) {
	registryExtensionID, err := UnmarshalRegistryExtensionID(id)
	if err != nil {
		return nil, err
	}
	switch {
	case registryExtensionID.LocalID != 0 && RegistryExtensionByIDInt32 != nil:
		return RegistryExtensionByIDInt32(ctx, db, registryExtensionID.LocalID)
	case registryExtensionID.RemoteID != nil:
		x, err := getRemoteRegistryExtension(ctx, "uuid", registryExtensionID.RemoteID.UUID)
		if err != nil {
			return nil, err
		}
		return &registryExtensionRemoteResolver{v: x}, nil
	default:
		return nil, errors.New("invalid registry extension ID")
	}
}

// RegistryPublisherByID looks up and returns the registry publisher by GraphQL ID. If there is no
// local registry, it is not implemented.
var RegistryPublisherByID func(context.Context, graphql.ID) (graphqlbackend.RegistryPublisher, error)

// extensionDBResolver implements the GraphQL type RegistryExtension.
type extensionDBResolver struct {
	db dbutil.DB
	v  *store.DBExtension

	// Supplied as part of list endpoints, but
	// calculated as part of single-extension endpoints
	r *store.DBRelease
}

var _ graphqlbackend.RegistryExtension = &extensionDBResolver{}

func (r *extensionDBResolver) ID() graphql.ID {
	return MarshalRegistryExtensionID(RegistryExtensionID{LocalID: r.v.ID})
}

func (r *extensionDBResolver) UUID() string        { return r.v.UUID }
func (r *extensionDBResolver) ExtensionID() string { return r.v.NonCanonicalExtensionID }
func (r *extensionDBResolver) ExtensionIDWithoutRegistry() string {
	if r.v.NonCanonicalRegistry != "" {
		return strings.TrimPrefix(r.v.NonCanonicalExtensionID, r.v.NonCanonicalRegistry+"/")
	}
	return r.v.NonCanonicalExtensionID
}

func (r *extensionDBResolver) Publisher(ctx context.Context) (graphqlbackend.RegistryPublisher, error) {
	return getRegistryPublisher(ctx, r.db, r.v.Publisher)
}

func (r *extensionDBResolver) Name() string { return r.v.Name }
func (r *extensionDBResolver) Manifest(ctx context.Context) (graphqlbackend.ExtensionManifest, error) {
	release, err := r.release(ctx)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return NewExtensionManifest(nil), nil
	}
	return NewExtensionManifest(&release.Manifest), nil
}

func (r *extensionDBResolver) CreatedAt() *graphqlbackend.DateTime {
	return &graphqlbackend.DateTime{Time: r.v.CreatedAt}
}

func (r *extensionDBResolver) UpdatedAt() *graphqlbackend.DateTime {
	return &graphqlbackend.DateTime{Time: r.v.UpdatedAt}
}

func (r *extensionDBResolver) PublishedAt(ctx context.Context) (*graphqlbackend.DateTime, error) {
	release, err := r.release(ctx)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return &graphqlbackend.DateTime{Time: time.Time{}}, nil
	}
	return &graphqlbackend.DateTime{Time: release.CreatedAt}, nil
}

func (r *extensionDBResolver) URL() string {
	return types.ExtensionURL(r.v.NonCanonicalExtensionID)
}
func (r *extensionDBResolver) RemoteURL() *string { return nil }

func (r *extensionDBResolver) RegistryName() (string, error) {
	return r.v.NonCanonicalRegistry, nil
}

func (r *extensionDBResolver) IsLocal() bool { return true }

func (r *extensionDBResolver) IsWorkInProgress() bool {
	return r.v.NonCanonicalIsWorkInProgress
}

func (r *extensionDBResolver) ViewerCanAdminister(ctx context.Context) (bool, error) {
	err := toRegistryPublisherID(r.v).viewerCanAdminister(ctx, r.db)
	if err == backend.ErrMustBeSiteAdmin || err == backend.ErrNotAnOrgMember || err == backend.ErrNotAuthenticated {
		return false, nil
	}
	if _, ok := err.(*backend.InsufficientAuthorizationError); ok {
		return false, nil
	}
	return err == nil, err
}

func (r *extensionDBResolver) release(ctx context.Context) (*store.DBRelease, error) {
	if r.r != nil {
		return r.r, nil
	}

	svc := service.New(r.db)
	var err error
	r.r, err = svc.GetLatestRelease(ctx, r.v.NonCanonicalExtensionID, r.v.ID, "release")
	return r.r, err
}
