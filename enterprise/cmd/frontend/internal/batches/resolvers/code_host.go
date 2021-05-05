package resolvers

import (
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

type batchChangesCodeHostResolver struct {
	codeHost   *btypes.CodeHost
	credential graphqlbackend.BatchChangesCredentialResolver
}

var _ graphqlbackend.BatchChangesCodeHostResolver = &batchChangesCodeHostResolver{}

func (c *batchChangesCodeHostResolver) ExternalServiceKind() string {
	return extsvc.TypeToKind(c.codeHost.ExternalServiceType)
}

func (c *batchChangesCodeHostResolver) ExternalServiceURL() string {
	return c.codeHost.ExternalServiceID
}

func (c *batchChangesCodeHostResolver) Credential() graphqlbackend.BatchChangesCredentialResolver {
	return c.credential
}

func (c *batchChangesCodeHostResolver) RequiresSSH() bool {
	return c.codeHost.RequiresSSH
}
