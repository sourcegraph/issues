package resolvers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/batches/resolvers/apitest"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/testing"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	internalapitest "github.com/sourcegraph/sourcegraph/internal/apitest"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestChangesetApplyPreviewConnectionResolver(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := backend.WithAuthzBypass(context.Background())
	db := dbtesting.GetDB(t)

	userID := ct.CreateTestUser(t, db, false).ID

	cstore := store.New(db, nil)

	batchSpec := &btypes.BatchSpec{
		UserID:          userID,
		NamespaceUserID: userID,
	}
	if err := cstore.CreateBatchSpec(ctx, batchSpec); err != nil {
		t.Fatal(err)
	}

	esStore := database.ExternalServicesWith(cstore)
	repoStore := database.ReposWith(cstore)

	rs := make([]*types.Repo, 0, 3)
	for i := 0; i < cap(rs); i++ {
		name := fmt.Sprintf("github.com/sourcegraph/test-changeset-apply-preview-connection-repo-%d", i)
		r := newGitHubTestRepo(name, newGitHubExternalService(t, esStore))
		if err := repoStore.Create(ctx, r); err != nil {
			t.Fatal(err)
		}
		rs = append(rs, r)
	}

	changesetSpecs := make([]*btypes.ChangesetSpec, 0, len(rs))
	for i, r := range rs {
		repoID := graphqlbackend.MarshalRepositoryID(r.ID)
		s, err := btypes.NewChangesetSpecFromRaw(ct.NewRawChangesetSpecGitBranch(repoID, fmt.Sprintf("d34db33f-%d", i)))
		if err != nil {
			t.Fatal(err)
		}
		s.BatchSpecID = batchSpec.ID
		s.UserID = userID
		s.RepoID = r.ID

		if err := cstore.CreateChangesetSpec(ctx, s); err != nil {
			t.Fatal(err)
		}

		changesetSpecs = append(changesetSpecs, s)
	}

	s, err := graphqlbackend.NewSchema(db, &Resolver{store: cstore}, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	apiID := string(marshalBatchSpecRandID(batchSpec.RandID))

	tests := []struct {
		first int

		wantTotalCount  int
		wantHasNextPage bool
	}{
		{first: 1, wantTotalCount: 3, wantHasNextPage: true},
		{first: 2, wantTotalCount: 3, wantHasNextPage: true},
		{first: 3, wantTotalCount: 3, wantHasNextPage: false},
	}

	for _, tc := range tests {
		input := map[string]interface{}{"batchSpec": apiID, "first": tc.first}
		var response struct{ Node apitest.BatchSpec }
		internalapitest.MustExec(ctx, t, s, input, &response, queryChangesetApplyPreviewConnection)

		specs := response.Node.ApplyPreview
		if diff := cmp.Diff(tc.wantTotalCount, specs.TotalCount); diff != "" {
			t.Fatalf("first=%d, unexpected total count (-want +got):\n%s", tc.first, diff)
		}

		if diff := cmp.Diff(tc.wantHasNextPage, specs.PageInfo.HasNextPage); diff != "" {
			t.Fatalf("first=%d, unexpected hasNextPage (-want +got):\n%s", tc.first, diff)
		}
	}

	var endCursor *string
	for i := range changesetSpecs {
		input := map[string]interface{}{"batchSpec": apiID, "first": 1}
		if endCursor != nil {
			input["after"] = *endCursor
		}
		wantHasNextPage := i != len(changesetSpecs)-1

		var response struct{ Node apitest.BatchSpec }
		internalapitest.MustExec(ctx, t, s, input, &response, queryChangesetApplyPreviewConnection)

		specs := response.Node.ApplyPreview
		if diff := cmp.Diff(1, len(specs.Nodes)); diff != "" {
			t.Fatalf("unexpected number of nodes (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(len(changesetSpecs), specs.TotalCount); diff != "" {
			t.Fatalf("unexpected total count (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(wantHasNextPage, specs.PageInfo.HasNextPage); diff != "" {
			t.Fatalf("unexpected hasNextPage (-want +got):\n%s", diff)
		}

		endCursor = specs.PageInfo.EndCursor
		if want, have := wantHasNextPage, endCursor != nil; have != want {
			t.Fatalf("unexpected endCursor existence. want=%t, have=%t", want, have)
		}
	}
}

const queryChangesetApplyPreviewConnection = `
query($batchSpec: ID!, $first: Int!, $after: String) {
  node(id: $batchSpec) {
    __typename

    ... on BatchSpec {
      id

      applyPreview(first: $first, after: $after) {
        totalCount
        pageInfo { hasNextPage, endCursor }
        nodes {
          __typename
        }
        stats {
          push
          update
          undraft
          publish
          publishDraft
          sync
          import
          close
          reopen
          sleep
          detach

          added
          modified
          removed
        }
      }
    }
  }
}
`

func TestRewirerMappings(t *testing.T) {
	addResolverFixture := func(rw *rewirerMappingsFacade, mapping *btypes.RewirerMapping, resolver graphqlbackend.ChangesetApplyPreviewResolver) {
		rw.resolversMu.Lock()
		defer rw.resolversMu.Unlock()

		rw.resolvers[mapping] = resolver
	}
	opPtr := func(op btypes.ReconcilerOperation) *btypes.ReconcilerOperation {
		return &op
	}
	ctx := context.Background()

	t.Run("Page", func(t *testing.T) {
		// Set up a scenario that allows for some filtering.
		var (
			detach   = &btypes.RewirerMapping{ChangesetSpecID: 1}
			hidden   = &btypes.RewirerMapping{ChangesetSpecID: 2}
			noAction = &btypes.RewirerMapping{ChangesetSpecID: 3}
			publishA = &btypes.RewirerMapping{ChangesetSpecID: 4}
			publishB = &btypes.RewirerMapping{ChangesetSpecID: 5}
		)
		rmf := newRewirerMappingsFacade(nil, 0)
		rmf.All = btypes.RewirerMappings{detach, hidden, noAction, publishA, publishB}
		addResolverFixture(rmf, detach, &mockChangesetApplyPreviewResolver{
			visible: &mockVisibleChangesetApplyPreviewResolver{
				operations: []btypes.ReconcilerOperation{btypes.ReconcilerOperationDetach},
			},
		})
		addResolverFixture(rmf, hidden, &mockChangesetApplyPreviewResolver{
			hidden: &mockHiddenChangesetApplyPreviewResolver{},
		})
		addResolverFixture(rmf, noAction, &mockChangesetApplyPreviewResolver{
			visible: &mockVisibleChangesetApplyPreviewResolver{
				operations: []btypes.ReconcilerOperation{},
			},
		})
		addResolverFixture(rmf, publishA, &mockChangesetApplyPreviewResolver{
			visible: &mockVisibleChangesetApplyPreviewResolver{
				operations: []btypes.ReconcilerOperation{btypes.ReconcilerOperationPublish},
			},
		})
		addResolverFixture(rmf, publishB, &mockChangesetApplyPreviewResolver{
			visible: &mockVisibleChangesetApplyPreviewResolver{
				operations: []btypes.ReconcilerOperation{btypes.ReconcilerOperationPublish},
			},
		})

		// Scenario done! Let's run some tests where we expect success. Note
		// that the existence of hidden is important: any time we're filtering
		// by operation, it should never appear in the result.
		for name, tc := range map[string]struct {
			opts rewirerMappingPageOpts
			want rewirerMappingPage
		}{
			"no ops or limit": {
				opts: rewirerMappingPageOpts{},
				want: rewirerMappingPage{
					Mappings:   rmf.All,
					TotalCount: len(rmf.All),
				},
			},
			"no ops, first 3": {
				opts: rewirerMappingPageOpts{
					LimitOffset: &database.LimitOffset{Limit: 3},
				},
				want: rewirerMappingPage{
					Mappings:   rmf.All[0:3],
					TotalCount: len(rmf.All),
				},
			},
			"no ops, last 2": {
				opts: rewirerMappingPageOpts{
					LimitOffset: &database.LimitOffset{Limit: 3, Offset: 3},
				},
				want: rewirerMappingPage{
					Mappings:   rmf.All[3:],
					TotalCount: len(rmf.All),
				},
			},
			"no ops, last 2 without limit": {
				opts: rewirerMappingPageOpts{
					LimitOffset: &database.LimitOffset{Offset: 3},
				},
				want: rewirerMappingPage{
					Mappings:   rmf.All[3:],
					TotalCount: len(rmf.All),
				},
			},
			"no ops, negative limit": {
				opts: rewirerMappingPageOpts{
					LimitOffset: &database.LimitOffset{Limit: -1},
				},
				want: rewirerMappingPage{
					Mappings:   btypes.RewirerMappings{},
					TotalCount: len(rmf.All),
				},
			},
			"no ops, negative offset": {
				opts: rewirerMappingPageOpts{
					LimitOffset: &database.LimitOffset{Offset: -1},
				},
				want: rewirerMappingPage{
					Mappings:   btypes.RewirerMappings{},
					TotalCount: len(rmf.All),
				},
			},
			"no ops, out of bounds offset": {
				opts: rewirerMappingPageOpts{
					LimitOffset: &database.LimitOffset{Offset: 5},
				},
				want: rewirerMappingPage{
					Mappings:   btypes.RewirerMappings{},
					TotalCount: len(rmf.All),
				},
			},
			"non-existent op": {
				opts: rewirerMappingPageOpts{
					Op: opPtr(btypes.ReconcilerOperationClose),
				},
				want: rewirerMappingPage{
					Mappings:   btypes.RewirerMappings{},
					TotalCount: 0,
				},
			},
			"extant op, no limit": {
				opts: rewirerMappingPageOpts{
					Op: opPtr(btypes.ReconcilerOperationPublish),
				},
				want: rewirerMappingPage{
					Mappings:   btypes.RewirerMappings{publishA, publishB},
					TotalCount: 2,
				},
			},
			"extant op, high limit": {
				opts: rewirerMappingPageOpts{
					LimitOffset: &database.LimitOffset{Limit: 5},
					Op:          opPtr(btypes.ReconcilerOperationPublish),
				},
				want: rewirerMappingPage{
					Mappings:   btypes.RewirerMappings{publishA, publishB},
					TotalCount: 2,
				},
			},
			"extant op, low limit": {
				opts: rewirerMappingPageOpts{
					LimitOffset: &database.LimitOffset{Limit: 1},
					Op:          opPtr(btypes.ReconcilerOperationPublish),
				},
				want: rewirerMappingPage{
					Mappings:   btypes.RewirerMappings{publishA},
					TotalCount: 2,
				},
			},
			"extant op, low limit and offset": {
				opts: rewirerMappingPageOpts{
					LimitOffset: &database.LimitOffset{Limit: 1, Offset: 1},
					Op:          opPtr(btypes.ReconcilerOperationPublish),
				},
				want: rewirerMappingPage{
					Mappings:   btypes.RewirerMappings{publishB},
					TotalCount: 2,
				},
			},
		} {
			t.Run(name, func(t *testing.T) {
				// We'll run the test twice to ensure we hit the memoisation.
				test := func(t *testing.T) {
					have, err := rmf.Page(ctx, tc.opts)
					if err != nil {
						t.Errorf("unexpected error: %+v", err)
					}
					if diff := cmp.Diff(have, &tc.want); diff != "" {
						t.Errorf("unexpected page (-have +want):\n%s", diff)
					}
				}

				t.Run("cold cache", test)
				t.Run("cache check", func(t *testing.T) {
					if _, ok := rmf.pages[tc.opts]; !ok {
						t.Error("unexpected cache miss")
					}
				})
				t.Run("warm cache", test)
			})
		}

		// And now, let's make sure we handle our one failure case gracefully by
		// replacing the detach resolver with one that errors.
		addResolverFixture(rmf, detach, &mockChangesetApplyPreviewResolver{
			visible: &mockVisibleChangesetApplyPreviewResolver{
				operationsErr: errors.New("just as reliable as the Canucks"),
			},
		})

		if _, err := rmf.Page(ctx, rewirerMappingPageOpts{
			Op: opPtr(btypes.ReconcilerOperationClose),
		}); err == nil {
			t.Error("unexpected nil error")
		}
	})

	t.Run("Resolver", func(t *testing.T) {
		compareResolvers := func(t *testing.T, have, want *changesetApplyPreviewResolver) {
			t.Helper()

			if have.store != want.store {
				t.Errorf("unexpected store: have=%p want=%p", have.store, want.store)
			}
			if have.mapping != want.mapping {
				t.Errorf("unexpected mapping: have=%p want=%p", have.mapping, want.mapping)
			}
			if have.preloadedBatchChange != want.preloadedBatchChange {
				t.Errorf("unexpected batch change: have=%p want=%p", have.preloadedBatchChange, want.preloadedBatchChange)
			}
			if !have.preloadedNextSync.Equal(want.preloadedNextSync) {
				t.Errorf("unexpected next sync: have=%s want=%s", have.preloadedNextSync, want.preloadedNextSync)
			}
			if have.batchSpecID != want.batchSpecID {
				t.Errorf("unexpected spec ID: have=%d want=%d", have.batchSpecID, want.batchSpecID)
			}
		}

		s := &store.Store{}
		rmf := newRewirerMappingsFacade(s, 1)
		rmf.batchChange = &btypes.BatchChange{}

		mapping := &btypes.RewirerMapping{}

		have := rmf.Resolver(mapping).(*changesetApplyPreviewResolver)
		want := &changesetApplyPreviewResolver{
			store:                s,
			mapping:              mapping,
			preloadedBatchChange: rmf.batchChange,
			batchSpecID:          1,
		}
		compareResolvers(t, have, want)

		// Ensure we get the same resolver the second time.
		if cached := rmf.Resolver(mapping).(*changesetApplyPreviewResolver); cached != have {
			t.Errorf("unexpected resolver from warm cache: have=%v want=%v", cached, have)
		}

		// Ensure we get a resolver with the correct next sync time if given.
		nextSync := time.Now()
		have = rmf.ResolverWithNextSync(mapping, nextSync).(*changesetApplyPreviewResolver)
		want.preloadedNextSync = nextSync
		compareResolvers(t, have, want)
	})
}

type mockChangesetApplyPreviewResolver struct {
	hidden  graphqlbackend.HiddenChangesetApplyPreviewResolver
	visible graphqlbackend.VisibleChangesetApplyPreviewResolver
}

func (r *mockChangesetApplyPreviewResolver) ToHiddenChangesetApplyPreview() (graphqlbackend.HiddenChangesetApplyPreviewResolver, bool) {
	return r.hidden, r.hidden != nil
}
func (r *mockChangesetApplyPreviewResolver) ToVisibleChangesetApplyPreview() (graphqlbackend.VisibleChangesetApplyPreviewResolver, bool) {
	return r.visible, r.visible != nil
}

var _ graphqlbackend.ChangesetApplyPreviewResolver = &mockChangesetApplyPreviewResolver{}

type mockHiddenChangesetApplyPreviewResolver struct{}

func (*mockHiddenChangesetApplyPreviewResolver) Operations(context.Context) ([]string, error) {
	return nil, errors.New("hidden changeset")
}
func (*mockHiddenChangesetApplyPreviewResolver) Delta(context.Context) (graphqlbackend.ChangesetSpecDeltaResolver, error) {
	return nil, errors.New("hidden changeset")
}
func (*mockHiddenChangesetApplyPreviewResolver) Targets() graphqlbackend.HiddenApplyPreviewTargetsResolver {
	return nil
}

var _ graphqlbackend.HiddenChangesetApplyPreviewResolver = &mockHiddenChangesetApplyPreviewResolver{}

type mockVisibleChangesetApplyPreviewResolver struct {
	operations    []btypes.ReconcilerOperation
	operationsErr error
	delta         graphqlbackend.ChangesetSpecDeltaResolver
	deltaErr      error
	targets       graphqlbackend.VisibleApplyPreviewTargetsResolver
}

func (r *mockVisibleChangesetApplyPreviewResolver) Operations(context.Context) ([]string, error) {
	strOps := make([]string, 0, len(r.operations))
	for _, op := range r.operations {
		strOps = append(strOps, string(op))
	}
	return strOps, r.operationsErr
}
func (r *mockVisibleChangesetApplyPreviewResolver) Delta(context.Context) (graphqlbackend.ChangesetSpecDeltaResolver, error) {
	return r.delta, r.deltaErr
}
func (r *mockVisibleChangesetApplyPreviewResolver) Targets() graphqlbackend.VisibleApplyPreviewTargetsResolver {
	return r.targets
}

var _ graphqlbackend.VisibleChangesetApplyPreviewResolver = &mockVisibleChangesetApplyPreviewResolver{}
