package graphqlbackend

import (
	"context"
	"errors"

	"github.com/graph-gophers/graphql-go"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/internal/conf"
)

var ErrExtensionsDisabled = errors.New("extensions are disabled in site configuration (contact the site admin to enable extensions)")

func (r *schemaResolver) ExtensionRegistry(ctx context.Context) (ExtensionRegistryResolver, error) {
	if conf.Extensions() == nil {
		if !ExtensionRegistry.ImplementsLocalExtensionRegistry() {
			// The OSS build doesn't implement a local extension registry, so the reason for
			// extensions being disabled is probably that the OSS build is in use.
			return nil, errors.New("no extension registry is available (use Sourcegraph Free or Sourcegraph Enterprise to access the Sourcegraph extension registry and/or to host a private internal extension registry)")
		}

		return nil, ErrExtensionsDisabled
	}
	return ExtensionRegistry, nil
}

// ExtensionRegistry is the implementation of the GraphQL types ExtensionRegistry and
// ExtensionRegistryMutation.
var ExtensionRegistry ExtensionRegistryResolver

// ExtensionRegistryResolver is the interface for the GraphQL types ExtensionRegistry and
// ExtensionRegistryMutation.
//
// Some methods are only implemented if there is a local extension registry. For these methods, the
// implementation (if one exists) is set on the XyzFunc struct field.
type ExtensionRegistryResolver interface {
	Extensions(context.Context, *RegistryExtensionConnectionArgs) (RegistryExtensionConnection, error)
	Extension(context.Context, *ExtensionRegistryExtensionArgs) (RegistryExtension, error)
	ViewerPublishers(context.Context) ([]RegistryPublisher, error)
	Publishers(context.Context, *graphqlutil.ConnectionArgs) (RegistryPublisherConnection, error)
	CreateExtension(context.Context, *ExtensionRegistryCreateExtensionArgs) (ExtensionRegistryMutationResult, error)
	UpdateExtension(context.Context, *ExtensionRegistryUpdateExtensionArgs) (ExtensionRegistryMutationResult, error)
	PublishExtension(context.Context, *ExtensionRegistryPublishExtensionArgs) (ExtensionRegistryMutationResult, error)
	DeleteExtension(context.Context, *ExtensionRegistryDeleteExtensionArgs) (*EmptyResponse, error)
	LocalExtensionIDPrefix() *string

	ImplementsLocalExtensionRegistry() bool // not exposed via GraphQL
	// FilterRemoteExtensions enforces `allowRemoteExtensions` by returning a
	// new slice with extension IDs that were present in
	// `allowRemoteExtensions`. It returns the original extension IDs if
	// `allowRemoteExtensions` is not set.
	FilterRemoteExtensions([]string) []string // not exposed via GraphQL
}

type RegistryExtensionConnectionArgs struct {
	graphqlutil.ConnectionArgs
	Query                  *string
	Publisher              *graphql.ID
	Local                  bool
	Remote                 bool
	PrioritizeExtensionIDs *[]string
}

type ExtensionRegistryExtensionArgs struct {
	ExtensionID string
}

type ExtensionRegistryCreateExtensionArgs struct {
	Publisher graphql.ID
	Name      string
}

type ExtensionRegistryUpdateExtensionArgs struct {
	Extension graphql.ID
	Name      *string
}

type ExtensionRegistryPublishExtensionArgs struct {
	ExtensionID string
	Manifest    string
	Bundle      *string
	SourceMap   *string
	Force       bool
}

type ExtensionRegistryDeleteExtensionArgs struct {
	Extension graphql.ID
}

// ExtensionRegistryMutationResult is the interface for the GraphQL type ExtensionRegistryMutationResult.
type ExtensionRegistryMutationResult interface {
	Extension(context.Context) (RegistryExtension, error)
}

// NodeToRegistryExtension is called to convert GraphQL node values to values of type
// RegistryExtension. It is assigned at init time.
var NodeToRegistryExtension func(interface{}) (RegistryExtension, bool)

// RegistryExtensionByID is called to look up values of GraphQL type RegistryExtension. It is
// assigned at init time.
var RegistryExtensionByID func(context.Context, graphql.ID) (RegistryExtension, error)

// RegistryExtension is the interface for the GraphQL type RegistryExtension.
type RegistryExtension interface {
	ID() graphql.ID
	UUID() string
	ExtensionID() string
	ExtensionIDWithoutRegistry() string
	Publisher(ctx context.Context) (RegistryPublisher, error)
	Name() string
	Manifest(ctx context.Context) (ExtensionManifest, error)
	CreatedAt() *DateTime
	UpdatedAt() *DateTime
	PublishedAt(context.Context) (*DateTime, error)
	URL() string
	RemoteURL() *string
	RegistryName() (string, error)
	IsLocal() bool
	IsWorkInProgress() bool
	ViewerCanAdminister(ctx context.Context) (bool, error)
}

// ExtensionManifest is the interface for the GraphQL type ExtensionManifest.
type ExtensionManifest interface {
	Raw() string
	Description() (*string, error)
	BundleURL() (*string, error)
}

// RegistryPublisher is the interface for the GraphQL type RegistryPublisher.
type RegistryPublisher interface {
	ToUser() (*UserResolver, bool)
	ToOrg() (*OrgResolver, bool)

	// Helpers that are not GraphQL fields.
	RegistryExtensionConnectionURL() (*string, error)
}

// RegistryExtensionConnection is the interface for the GraphQL type RegistryExtensionConnection.
type RegistryExtensionConnection interface {
	Nodes(context.Context) ([]RegistryExtension, error)
	TotalCount(context.Context) (int32, error)
	PageInfo(context.Context) (*graphqlutil.PageInfo, error)
	URL(context.Context) (*string, error)
	Error(context.Context) *string
}

// RegistryPublisherConnection is the interface for the GraphQL type RegistryPublisherConnection.
type RegistryPublisherConnection interface {
	Nodes(context.Context) ([]RegistryPublisher, error)
	TotalCount(context.Context) (int32, error)
	PageInfo(context.Context) (*graphqlutil.PageInfo, error)
}
