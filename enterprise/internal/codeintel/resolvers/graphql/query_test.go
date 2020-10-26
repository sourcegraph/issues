package graphql

import (
	"context"
	"encoding/base64"
	"testing"

	gql "github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	resolvermocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/resolvers/mocks"
)

func TestRanges(t *testing.T) {
	mockResolver := resolvermocks.NewMockQueryResolver()
	resolver := NewQueryResolver(mockResolver, NewCachedLocationResolver())

	args := &gql.LSIFRangesArgs{StartLine: 10, EndLine: 20}
	if _, err := resolver.Ranges(context.Background(), args); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(mockResolver.RangesFunc.History()) != 1 {
		t.Fatalf("unexpected call count. want=%d have=%d", 1, len(mockResolver.RangesFunc.History()))
	}
	if val := mockResolver.RangesFunc.History()[0].Arg1; val != 10 {
		t.Fatalf("unexpected start line. want=%d have=%d", 10, val)
	}
	if val := mockResolver.RangesFunc.History()[0].Arg2; val != 20 {
		t.Fatalf("unexpected end line. want=%d have=%d", 20, val)
	}
}

func TestDefinitions(t *testing.T) {
	mockResolver := resolvermocks.NewMockQueryResolver()
	resolver := NewQueryResolver(mockResolver, NewCachedLocationResolver())

	args := &gql.LSIFQueryPositionArgs{Line: 10, Character: 15}
	if _, err := resolver.Definitions(context.Background(), args); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(mockResolver.DefinitionsFunc.History()) != 1 {
		t.Fatalf("unexpected call count. want=%d have=%d", 1, len(mockResolver.DefinitionsFunc.History()))
	}
	if val := mockResolver.DefinitionsFunc.History()[0].Arg1; val != 10 {
		t.Fatalf("unexpected line. want=%d have=%d", 10, val)
	}
	if val := mockResolver.DefinitionsFunc.History()[0].Arg2; val != 15 {
		t.Fatalf("unexpected character. want=%d have=%d", 15, val)
	}
}

func TestReferences(t *testing.T) {
	mockResolver := resolvermocks.NewMockQueryResolver()
	resolver := NewQueryResolver(mockResolver, NewCachedLocationResolver())

	offset := int32(25)
	cursor := base64.StdEncoding.EncodeToString([]byte("test-cursor"))

	args := &gql.LSIFPagedQueryPositionArgs{
		LSIFQueryPositionArgs: gql.LSIFQueryPositionArgs{
			Line:      10,
			Character: 15,
		},
		ConnectionArgs: graphqlutil.ConnectionArgs{First: &offset},
		After:          &cursor,
	}

	if _, err := resolver.References(context.Background(), args); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(mockResolver.ReferencesFunc.History()) != 1 {
		t.Fatalf("unexpected call count. want=%d have=%d", 1, len(mockResolver.ReferencesFunc.History()))
	}
	if val := mockResolver.ReferencesFunc.History()[0].Arg1; val != 10 {
		t.Fatalf("unexpected line. want=%d have=%d", 10, val)
	}
	if val := mockResolver.ReferencesFunc.History()[0].Arg2; val != 15 {
		t.Fatalf("unexpected character. want=%d have=%d", 15, val)
	}
	if val := mockResolver.ReferencesFunc.History()[0].Arg3; val != 25 {
		t.Fatalf("unexpected character. want=%d have=%d", 25, val)
	}
	if val := mockResolver.ReferencesFunc.History()[0].Arg4; val != "test-cursor" {
		t.Fatalf("unexpected character. want=%s have=%s", "test-cursor", val)
	}
}

func TestReferencesDefaultLimit(t *testing.T) {
	mockResolver := resolvermocks.NewMockQueryResolver()
	resolver := NewQueryResolver(mockResolver, NewCachedLocationResolver())

	args := &gql.LSIFPagedQueryPositionArgs{
		LSIFQueryPositionArgs: gql.LSIFQueryPositionArgs{
			Line:      10,
			Character: 15,
		},
		ConnectionArgs: graphqlutil.ConnectionArgs{},
	}

	if _, err := resolver.References(context.Background(), args); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(mockResolver.ReferencesFunc.History()) != 1 {
		t.Fatalf("unexpected call count. want=%d have=%d", 1, len(mockResolver.DiagnosticsFunc.History()))
	}
	if val := mockResolver.ReferencesFunc.History()[0].Arg3; val != DefaultReferencesPageSize {
		t.Fatalf("unexpected limit. want=%d have=%d", DefaultReferencesPageSize, val)
	}
}

func TestReferencesDefaultIllegalLimit(t *testing.T) {
	mockResolver := resolvermocks.NewMockQueryResolver()
	resolver := NewQueryResolver(mockResolver, NewCachedLocationResolver())

	offset := int32(-1)
	args := &gql.LSIFPagedQueryPositionArgs{
		LSIFQueryPositionArgs: gql.LSIFQueryPositionArgs{
			Line:      10,
			Character: 15,
		},
		ConnectionArgs: graphqlutil.ConnectionArgs{First: &offset},
	}

	if _, err := resolver.References(context.Background(), args); err != ErrIllegalLimit {
		t.Fatalf("unexpected error. want=%q have=%q", ErrIllegalLimit, err)
	}
}

func TestHover(t *testing.T) {
	mockResolver := resolvermocks.NewMockQueryResolver()
	mockResolver.HoverFunc.SetDefaultReturn("text", bundles.Range{}, true, nil)
	resolver := NewQueryResolver(mockResolver, NewCachedLocationResolver())

	args := &gql.LSIFQueryPositionArgs{Line: 10, Character: 15}
	if _, err := resolver.Hover(context.Background(), args); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(mockResolver.HoverFunc.History()) != 1 {
		t.Fatalf("unexpected call count. want=%d have=%d", 1, len(mockResolver.HoverFunc.History()))
	}
	if val := mockResolver.HoverFunc.History()[0].Arg1; val != 10 {
		t.Fatalf("unexpected line. want=%d have=%d", 10, val)
	}
	if val := mockResolver.HoverFunc.History()[0].Arg2; val != 15 {
		t.Fatalf("unexpected character. want=%d have=%d", 15, val)
	}
}

func TestDiagnostics(t *testing.T) {
	mockResolver := resolvermocks.NewMockQueryResolver()
	resolver := NewQueryResolver(mockResolver, NewCachedLocationResolver())

	offset := int32(25)
	args := &gql.LSIFDiagnosticsArgs{
		ConnectionArgs: graphqlutil.ConnectionArgs{First: &offset},
	}

	if _, err := resolver.Diagnostics(context.Background(), args); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(mockResolver.DiagnosticsFunc.History()) != 1 {
		t.Fatalf("unexpected call count. want=%d have=%d", 1, len(mockResolver.DiagnosticsFunc.History()))
	}
	if val := mockResolver.DiagnosticsFunc.History()[0].Arg1; val != 25 {
		t.Fatalf("unexpected limit. want=%d have=%d", 25, val)
	}
}

func TestDiagnosticsDefaultLimit(t *testing.T) {
	mockResolver := resolvermocks.NewMockQueryResolver()
	resolver := NewQueryResolver(mockResolver, NewCachedLocationResolver())

	args := &gql.LSIFDiagnosticsArgs{
		ConnectionArgs: graphqlutil.ConnectionArgs{},
	}

	if _, err := resolver.Diagnostics(context.Background(), args); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(mockResolver.DiagnosticsFunc.History()) != 1 {
		t.Fatalf("unexpected call count. want=%d have=%d", 1, len(mockResolver.DiagnosticsFunc.History()))
	}
	if val := mockResolver.DiagnosticsFunc.History()[0].Arg1; val != DefaultDiagnosticsPageSize {
		t.Fatalf("unexpected limit. want=%d have=%d", DefaultDiagnosticsPageSize, val)
	}
}

func TestDiagnosticsDefaultIllegalLimit(t *testing.T) {
	mockResolver := resolvermocks.NewMockQueryResolver()
	resolver := NewQueryResolver(mockResolver, NewCachedLocationResolver())

	offset := int32(-1)
	args := &gql.LSIFDiagnosticsArgs{
		ConnectionArgs: graphqlutil.ConnectionArgs{First: &offset},
	}

	if _, err := resolver.Diagnostics(context.Background(), args); err != ErrIllegalLimit {
		t.Fatalf("unexpected error. want=%q have=%q", ErrIllegalLimit, err)
	}
}
