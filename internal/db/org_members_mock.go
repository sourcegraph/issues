package db

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
)

type MockOrgMembers struct {
	GetByOrgIDAndUserID func(ctx context.Context, orgID, userID int32) (*types.OrgMembership, error)
}

func (s *MockOrgMembers) MockGetByOrgIDAndUserID_Return(t *testing.T, returns *types.OrgMembership, returnsErr error) (called *bool) {
	called = new(bool)
	s.GetByOrgIDAndUserID = func(ctx context.Context, orgID, userID int32) (*types.OrgMembership, error) {
		*called = true
		return returns, returnsErr
	}
	return
}
