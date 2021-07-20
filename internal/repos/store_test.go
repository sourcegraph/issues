package repos_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/awscodecommit"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitolite"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/timeutil"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

func testStoreUpsertRepos(store *repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		clock := timeutil.NewFakeClock(time.Now(), 0)
		now := clock.Now()

		kinds := []string{
			extsvc.KindGitHub,
			extsvc.KindGitLab,
			extsvc.KindBitbucketServer,
			extsvc.KindAWSCodeCommit,
			extsvc.KindOther,
			extsvc.KindGitolite,
		}

		servicesPerKind := createExternalServices(t, store)

		github := types.Repo{
			Name:        "github.com/foo/bar",
			URI:         "github.com/foo/bar",
			Description: "The description",
			Stars:       100,
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "AAAAA==",
				ServiceType: "github",
				ServiceID:   "http://github.com",
			},
			Sources: map[string]*types.SourceInfo{
				servicesPerKind[extsvc.KindGitHub].URN(): {
					ID:       servicesPerKind[extsvc.KindGitHub].URN(),
					CloneURL: "git@github.com:foo/bar.git",
				},
			},
			Metadata: new(github.Repository),
		}

		gitlab := types.Repo{
			Name:        "gitlab.com/foo/bar",
			URI:         "gitlab.com/foo/bar",
			Description: "The description",
			Stars:       100,
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "1234",
				ServiceType: extsvc.TypeGitLab,
				ServiceID:   "http://gitlab.com",
			},
			Sources: map[string]*types.SourceInfo{
				servicesPerKind[extsvc.KindGitLab].URN(): {
					ID:       servicesPerKind[extsvc.KindGitLab].URN(),
					CloneURL: "git@gitlab.com:foo/bar.git",
				},
			},
			Metadata: new(gitlab.Project),
		}

		bitbucketServer := types.Repo{
			Name:        "bitbucketserver.mycorp.com/foo/bar",
			URI:         "bitbucketserver.mycorp.com/foo/bar",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "1234",
				ServiceType: "bitbucketServer",
				ServiceID:   "http://bitbucketserver.mycorp.com",
			},
			Sources: map[string]*types.SourceInfo{
				servicesPerKind[extsvc.KindBitbucketServer].URN(): {
					ID:       servicesPerKind[extsvc.KindBitbucketServer].URN(),
					CloneURL: "git@bitbucketserver.mycorp.com:foo/bar.git",
				},
			},
			Metadata: new(bitbucketserver.Repo),
		}

		awsCodeCommit := types.Repo{
			Name:        "git-codecommit.us-west-1.amazonaws.com/stripe-go",
			URI:         "git-codecommit.us-west-1.amazonaws.com/stripe-go",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "f001337a-3450-46fd-b7d2-650c0EXAMPLE",
				ServiceType: extsvc.TypeAWSCodeCommit,
				ServiceID:   "arn:aws:codecommit:us-west-1:999999999999:",
			},
			Sources: map[string]*types.SourceInfo{
				servicesPerKind[extsvc.KindAWSCodeCommit].URN(): {
					ID:       servicesPerKind[extsvc.KindAWSCodeCommit].URN(),
					CloneURL: "git@git-codecommit.us-west-1.amazonaws.com/v1/repos/stripe-go",
				},
			},
			Metadata: new(awscodecommit.Repository),
		}

		otherRepo := types.Repo{
			Name: "git-host.com/org/foo",
			URI:  "git-host.com/org/foo",
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "git-host.com/org/foo",
				ServiceID:   "https://git-host.com/",
				ServiceType: extsvc.TypeOther,
			},
			Metadata: &extsvc.OtherRepoMetadata{},
			Sources: map[string]*types.SourceInfo{
				servicesPerKind[extsvc.KindOther].URN(): {
					ID:       servicesPerKind[extsvc.KindOther].URN(),
					CloneURL: "https://git-host.com/org/foo",
				},
			},
		}

		gitoliteRepo := types.Repo{
			Name:      "gitolite.mycorp.com/bar",
			URI:       "gitolite.mycorp.com/bar",
			CreatedAt: now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "bar",
				ServiceType: extsvc.TypeGitolite,
				ServiceID:   "git@gitolite.mycorp.com",
			},
			Sources: map[string]*types.SourceInfo{
				servicesPerKind[extsvc.KindGitolite].URN(): {
					ID:       servicesPerKind[extsvc.KindGitolite].URN(),
					CloneURL: "git@gitolite.mycorp.com:bar.git",
				},
			},
			Metadata: new(gitolite.Repo),
		}

		repositories := types.Repos{
			&github,
			&gitlab,
			&bitbucketServer,
			&awsCodeCommit,
			&otherRepo,
			&gitoliteRepo,
		}

		ctx := context.Background()

		t.Run("no repos", func(t *testing.T) {
			if err := store.UpsertRepos(ctx); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}
		})

		t.Run("many repos", transact(ctx, store, func(t testing.TB, tx *repos.Store) {
			want := mkRepos(7, repositories...)

			if err := tx.UpsertRepos(ctx, want...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}
			if err := tx.UpsertSources(ctx, want.Sources(), nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			sort.Sort(want)

			if noID := want.Filter(hasNoID); len(noID) > 0 {
				t.Fatalf("UpsertRepos didn't assign an ID to all repos: %v", noID.Names())
			}

			have, err := tx.RepoStore.List(ctx, database.ReposListOptions{
				ServiceTypes: kinds,
			})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff(have, []*types.Repo(want), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}

			suffix := "-updated"
			now := clock.Now()
			for _, r := range want {
				r.Name += api.RepoName(suffix)
				r.URI += suffix
				r.Description += suffix
				r.UpdatedAt = now
				r.CreatedAt = now
				r.Stars++
				r.Archived = !r.Archived
				r.Fork = !r.Fork
			}

			if err = tx.UpsertRepos(ctx, want.Clone()...); err != nil {
				t.Errorf("UpsertRepos error: %s", err)
			} else if err = tx.UpsertSources(ctx, want.Clone().Sources(), nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			} else if have, err = tx.RepoStore.List(ctx, database.ReposListOptions{}); err != nil {
				t.Errorf("ListRepos error: %s", err)
			} else if diff := cmp.Diff(have, []*types.Repo(want), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ListRepos:\n%s", diff)
			}

			deleted := want.Clone().With(types.Opt.RepoDeletedAt(now))
			args := database.ReposListOptions{}

			if err = tx.UpsertRepos(ctx, deleted...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			} else if have, err = tx.RepoStore.List(ctx, args); err != nil {
				t.Errorf("ListRepos error: %s", err)
			} else if diff := cmp.Diff(have, []*types.Repo(nil), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ListRepos:\n%s", diff)
			}

			// Insert previously soft-deleted repos. Ensure we get back the same ID.
			if err = tx.UpsertRepos(ctx, want.Clone().With(types.Opt.RepoID(0))...); err != nil {
				t.Errorf("UpsertRepos error: %s", err)
			} else if err = tx.UpsertSources(ctx, want.Clone().Sources(), nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			} else if have, err = tx.RepoStore.List(ctx, database.ReposListOptions{}); err != nil {
				t.Errorf("ListRepos error: %s", err)
			} else if diff := cmp.Diff(have, []*types.Repo(want), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ListRepos:\n%s", diff)
			}

			// Delete all again, then try insert repos with different external
			// IDs but same name. Check we get new IDs.
			for _, r := range want {
				r.ID = 0
				r.ExternalRepo.ID += "-different"
			}
			if err = tx.UpsertRepos(ctx, deleted...); err != nil {
				t.Fatalf("UpsertRepos deleted error: %s", err)
			} else if err = tx.UpsertRepos(ctx, want...); err != nil {
				t.Fatalf("UpsertRepos want error: %s", err)
			} else if err = tx.UpsertSources(ctx, want.Sources(), nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			} else if have, err = tx.RepoStore.List(ctx, database.ReposListOptions{}); err != nil {
				t.Errorf("ListRepos error: %s", err)
			} else if diff := cmp.Diff(have, []*types.Repo(want), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ListRepos:\n%s", diff)
			} else if sameIDs := want.Filter(hasID(deleted.IDs()...)); len(sameIDs) > 0 {
				t.Errorf("ListRepos returned IDs of soft deleted repos: %v", sameIDs.Names())
			}
		}))

		t.Run("many repos soft-deleted and single repo reinserted", transact(ctx, store, func(t testing.TB, tx *repos.Store) {
			all := mkRepos(7, repositories...)

			if err := tx.UpsertRepos(ctx, all...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}
			if err := tx.UpsertSources(ctx, all.Sources(), nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			sort.Sort(all)

			if noID := all.Filter(hasNoID); len(noID) > 0 {
				t.Fatalf("UpsertRepos didn't assign an ID to all repos: %v", noID.Names())
			}

			have, err := tx.RepoStore.List(ctx, database.ReposListOptions{
				ServiceTypes: kinds,
			})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff(have, []*types.Repo(all), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}

			allDeleted := all.Clone().With(types.Opt.RepoDeletedAt(now))
			args := database.ReposListOptions{}

			if err = tx.UpsertRepos(ctx, allDeleted...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			} else if have, err = tx.RepoStore.List(ctx, args); err != nil {
				t.Errorf("ListRepos error: %s", err)
			} else if diff := cmp.Diff(have, []*types.Repo(nil), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ListRepos:\n%s", diff)
			}

			// Insert one of the previously soft-deleted repos. Ensure ID on upserted repo is set and we get back the same ID.
			want := types.Repos{all[0]}
			upsert := want.Clone().With(types.Opt.RepoID(0))
			if err = tx.UpsertRepos(ctx, upsert...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}
			if upsert[0].ID == 0 {
				t.Fatalf("Repo ID is zero")
			}
			if err := tx.UpsertSources(ctx, upsert.Sources(), nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			if have, err = tx.RepoStore.List(ctx, database.ReposListOptions{}); err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}
			if diff := cmp.Diff(have, []*types.Repo(want), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}
		}))
	}
}

func testStoreUpsertSources(store *repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		clock := timeutil.NewFakeClock(time.Now(), 0)
		now := clock.Now()

		servicesPerKind := createExternalServices(t, store)

		github := types.Repo{
			Name:        "github.com/foo/bar",
			URI:         "github.com/foo/bar",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "AAAAA==",
				ServiceType: "github",
				ServiceID:   "http://github.com",
			},
			Sources: map[string]*types.SourceInfo{
				servicesPerKind[extsvc.KindGitHub].URN(): {
					ID:       servicesPerKind[extsvc.KindGitHub].URN(),
					CloneURL: "git@github.com:foo/bar.git",
				},
			},
			Metadata: new(github.Repository),
		}

		gitlab := types.Repo{
			Name:        "gitlab.com/foo/bar",
			URI:         "gitlab.com/foo/bar",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "1234",
				ServiceType: extsvc.TypeGitLab,
				ServiceID:   "http://gitlab.com",
			},
			Sources: map[string]*types.SourceInfo{
				servicesPerKind[extsvc.KindGitLab].URN(): {
					ID:       servicesPerKind[extsvc.KindGitLab].URN(),
					CloneURL: "git@gitlab.com:foo/bar.git",
				},
			},
			Metadata: new(gitlab.Project),
		}

		repositories := types.Repos{
			&github,
			&gitlab,
		}

		ctx := context.Background()

		t.Run("no sources", func(t *testing.T) {
			if err := store.UpsertSources(ctx, nil, nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}
		})

		t.Run("delete repo", transact(ctx, store, func(t testing.TB, tx *repos.Store) {
			want := mkRepos(7, repositories...)

			if err := tx.UpsertRepos(ctx, want...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			sources := want.Sources()

			if err := tx.UpsertSources(ctx, sources, nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			// delete a repository
			want[0].DeletedAt = now
			if err := tx.RepoStore.Delete(ctx, want[0].ID); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			// un delete it
			want[0].DeletedAt = time.Time{}
			if err := tx.UpsertRepos(ctx, want...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			// it should not contain any source
			want[0].Sources = nil

			got, err := tx.RepoStore.List(ctx, database.ReposListOptions{})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff([]*types.Repo(want), got, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}
		}))

		t.Run("delete external service", transact(ctx, store, func(t testing.TB, tx *repos.Store) {
			origRepos := mkRepos(7, repositories...)

			if err := tx.UpsertRepos(ctx, origRepos...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			sources := origRepos.Sources()

			if err := tx.UpsertSources(ctx, sources, nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			// delete an external service
			svc := servicesPerKind[extsvc.KindGitHub]
			if err := tx.ExternalServiceStore.Delete(ctx, svc.ID); err != nil {
				t.Fatalf("Upsert externalServices error: %s", err)
			}

			// un delete it
			svc.DeletedAt = time.Time{}
			if err := tx.ExternalServiceStore.Upsert(ctx, svc); err != nil {
				t.Fatalf("Upsert externalServices error: %s", err)
			}

			// All GitHub sources should be deleted and all orphan repositories should be excluded
			want := make([]*types.Repo, 0, len(origRepos))
			origRepos.Apply(func(r *types.Repo) {
				for urn := range r.Sources {
					if strings.Contains(urn, "github") {
						delete(r.Sources, urn)
					}
				}
				if len(r.Sources) > 0 {
					want = append(want, r)
				}
			})

			got, err := tx.RepoStore.List(ctx, database.ReposListOptions{})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}
		}))

		t.Run("inserts updates and deletes", transact(ctx, store, func(t testing.TB, tx *repos.Store) {
			want := mkRepos(7, repositories...)

			if err := tx.UpsertRepos(ctx, want...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			sources := want.Sources()

			if err := tx.UpsertSources(ctx, sources, nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			have, err := tx.RepoStore.List(ctx, database.ReposListOptions{})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff([]*types.Repo(want), have, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}

			updates := make(map[api.RepoID][]types.SourceInfo)
			deletes := make(map[api.RepoID][]types.SourceInfo)

			updates[want[0].ID] = sources[want[0].ID]
			updates[want[0].ID][0].CloneURL = "something-else"
			deletes[want[1].ID] = sources[want[1].ID]

			if err := tx.UpsertSources(ctx, nil, updates, deletes); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			want[0].Sources[servicesPerKind[extsvc.KindGitHub].URN()] = &types.SourceInfo{
				CloneURL: "something-else",
				ID:       servicesPerKind[extsvc.KindGitHub].URN(),
			}

			// Remove the second element from want because it should be deleted automatically
			// by the time it become orphaned.
			want = append(want[:1], want[2:]...)

			have, err = tx.RepoStore.List(ctx, database.ReposListOptions{})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff([]*types.Repo(want), have, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}
		}))
	}
}

func testStoreSetClonedRepos(store *repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		servicesPerKind := createExternalServices(t, store)

		var repositories types.Repos
		for i := 0; i < 3; i++ {
			repositories = append(repositories, &types.Repo{
				Name: api.RepoName(fmt.Sprintf("github.com/%d/%d", i, i)),
				URI:  fmt.Sprintf("github.com/%d/%d", i, i),
				ExternalRepo: api.ExternalRepoSpec{
					ID:          strconv.Itoa(i),
					ServiceType: extsvc.TypeGitHub,
					ServiceID:   "http://github.com",
				},
				Sources: map[string]*types.SourceInfo{
					servicesPerKind[extsvc.KindGitHub].URN(): {
						ID:       servicesPerKind[extsvc.KindGitHub].URN(),
						CloneURL: "git@github.com:foo/bar.git",
					},
				},
				Metadata: new(github.Repository),
			})
		}

		check := func(t testing.TB, ctx context.Context, tx *repos.Store, wantNames []string) {
			t.Helper()

			res, err := tx.RepoStore.List(ctx, database.ReposListOptions{OnlyCloned: true})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			cloned := types.Repos(res).Names()
			sort.Strings(cloned)

			if got, want := cloned, wantNames; !cmp.Equal(got, want) {
				t.Fatalf("got=%v, want=%v: %s", got, want, cmp.Diff(got, want))
			}
		}

		ctx := context.Background()

		t.Run("no repo name", func(t *testing.T) {
			if err := store.SetClonedRepos(ctx); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}
		})

		t.Run("many repo names", transact(ctx, store, func(t testing.TB, tx *repos.Store) {
			stored := mkRepos(9, repositories...)

			if err := tx.RepoStore.Create(ctx, stored...); err != nil {
				t.Fatalf("InsertRepos error: %s", err)
			}

			sort.Sort(stored)

			names := stored[:3].Names()
			sort.Strings(names)

			if err := tx.SetClonedRepos(ctx, names...); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}
			check(t, ctx, tx, names)

			// setClonedRepositories should be idempotent and have the same behavior
			// when called with the same repos
			if err := tx.SetClonedRepos(ctx, names...); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}
			check(t, ctx, tx, names)

			// when adding another repo to the list, the other repos must be set as well
			names = stored[:4].Names()
			sort.Strings(names)
			if err := tx.SetClonedRepos(ctx, names...); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}

			check(t, ctx, tx, names)
		}))

		t.Run("repo names in mixed case", transact(ctx, store, func(t testing.TB, tx *repos.Store) {
			stored := mkRepos(9, repositories...)
			for i := range stored {
				if i%2 == 0 {
					stored[i].Name = api.RepoName(strings.ToUpper(string(stored[i].Name)))
				}
			}

			if err := tx.RepoStore.Create(ctx, stored...); err != nil {
				t.Fatalf("InsertRepos error: %s", err)
			}

			sort.Sort(stored)

			originalNames := stored.Names()
			sort.Strings(originalNames)

			lowerCaseNames := make([]string, 0, len(originalNames))
			for _, n := range originalNames {
				lowerCaseNames = append(lowerCaseNames, strings.ToLower(n))
			}

			if err := tx.SetClonedRepos(ctx, lowerCaseNames...); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}

			check(t, ctx, tx, originalNames)
		}))
	}
}

func testStoreCountNotClonedRepos(store *repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		servicesPerKind := createExternalServices(t, store)

		var repositories types.Repos
		for i := 0; i < 3; i++ {
			repositories = append(repositories, &types.Repo{
				Name: api.RepoName(fmt.Sprintf("github.com/%d/%d", i, i)),
				URI:  fmt.Sprintf("github.com/%d/%d", i, i),
				ExternalRepo: api.ExternalRepoSpec{
					ID:          strconv.Itoa(i),
					ServiceType: extsvc.TypeGitHub,
					ServiceID:   "http://github.com",
				},
				Sources: map[string]*types.SourceInfo{
					servicesPerKind[extsvc.KindGitHub].URN(): {
						ID:       servicesPerKind[extsvc.KindGitHub].URN(),
						CloneURL: "git@github.com:foo/bar.git",
					},
				},
				Metadata: new(github.Repository),
			})
		}

		ctx := context.Background()

		t.Run("no cloned repos", func(t *testing.T) {
			count, err := store.RepoStore.Count(ctx, database.ReposListOptions{NoCloned: true})
			if err != nil {
				t.Fatalf("CountNotClonedRepos error: %s", err)
			}
			if diff := cmp.Diff(count, 0); diff != "" {
				t.Fatalf("CountNotClonedRepos:\n%s", diff)
			}
		})

		t.Run("multiple cloned repos", transact(ctx, store, func(t testing.TB, tx *repos.Store) {
			stored := mkRepos(10, repositories...)

			if err := tx.RepoStore.Create(ctx, stored...); err != nil {
				t.Fatalf("InsertRepos error: %s", err)
			}

			sort.Sort(stored)
			cloned := stored[:3].Names()

			if err := tx.SetClonedRepos(ctx, cloned...); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}

			sort.Strings(cloned)

			count, err := tx.RepoStore.Count(ctx, database.ReposListOptions{NoCloned: true})
			if err != nil {
				t.Fatalf("CountNotClonedRepos error: %s", err)
			}
			if diff := cmp.Diff(count, 7); diff != "" {
				t.Fatalf("CountNotClonedRepos:\n%s", diff)
			}
		}))

		t.Run("deleted non cloned repos", transact(ctx, store, func(t testing.TB, tx *repos.Store) {
			stored := mkRepos(10, repositories...)

			if err := tx.RepoStore.Create(ctx, stored...); err != nil {
				t.Fatalf("InsertRepos error: %s", err)
			}

			sort.Sort(stored)
			cloned := stored[:3].Names()

			if err := tx.SetClonedRepos(ctx, cloned...); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}

			sort.Strings(cloned)
			deletedCloned := stored[8:]

			if err := tx.RepoStore.Delete(ctx, deletedCloned.IDs()...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			count, err := tx.RepoStore.Count(ctx, database.ReposListOptions{NoCloned: true})
			if err != nil {
				t.Fatalf("CountNotClonedRepos error: %s", err)
			}
			if diff := cmp.Diff(count, 5); diff != "" {
				t.Fatalf("CountNotClonedRepos:\n%s", diff)
			}
		}))
	}
}

func hasNoID(r *types.Repo) bool {
	return r.ID == 0
}

func hasID(ids ...api.RepoID) func(r *types.Repo) bool {
	return func(r *types.Repo) bool {
		for _, id := range ids {
			if r.ID == id {
				return true
			}
		}
		return false
	}
}

func testStoreListExternalRepoSpecs(store *repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		// Insert test repositories
		err := store.Exec(ctx, sqlf.Sprintf(`
INSERT INTO repo (id, name, description, fork, external_id, external_service_type, external_service_id, deleted_at)
VALUES
(1, 'github.com/user/repo1', '', FALSE, NULL, 'github', 'https://github.com/', NULL),
(2, 'github.com/user/repo2', '', FALSE, 'MDEwOlJlcG9zaXRvcnky', NULL, 'https://github.com/', NULL),
(3, 'github.com/user/repo3', '', FALSE, 'MDEwOlJlcG9zaXRvcnkz', 'github', NULL, NULL),
(4, 'github.com/user/repo4', '', FALSE, 'MDEwOlJlcG9zaXRvcnk0', 'github', 'https://github.com/', NOW()),
(5, 'github.com/user/repo5', '', FALSE, 'MDEwOlJlcG9zaXRvcnk1', 'github', 'https://github.com/', NULL)
`))
		if err != nil {
			t.Fatal(err)
		}

		ids, err := store.ListExternalRepoSpecs(ctx)
		if err != nil {
			t.Fatal(err)
		}
		want := map[api.ExternalRepoSpec]struct{}{
			{
				ID:          "MDEwOlJlcG9zaXRvcnk1",
				ServiceType: "github",
				ServiceID:   "https://github.com/",
			}: {},
		}
		if diff := cmp.Diff(want, ids); diff != "" {
			t.Fatalf("IDs mismatch (-want +got):\n%s", diff)
		}
	}
}

func testSyncRateLimiters(store *repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		clock := timeutil.NewFakeClock(time.Now(), 0)
		now := clock.Now()
		ctx := context.Background()
		transact(ctx, store, func(t testing.TB, tx *repos.Store) {
			toCreate := 501 // Larger than default page size in order to test pagination
			services := make([]*types.ExternalService, 0, toCreate)
			for i := 0; i < toCreate; i++ {
				svc := &types.ExternalService{
					ID:          int64(i) + 1,
					Kind:        "GitHub",
					DisplayName: "GitHub",
					CreatedAt:   now,
					UpdatedAt:   now,
					DeletedAt:   time.Time{},
				}
				config := schema.GitLabConnection{
					Url: fmt.Sprintf("http://example%d.com/", i),
					RateLimit: &schema.GitLabRateLimit{
						RequestsPerHour: 3600,
						Enabled:         true,
					},
				}
				data, err := json.Marshal(config)
				if err != nil {
					t.Fatal(err)
				}
				svc.Config = string(data)
				services = append(services, svc)
			}

			if err := tx.ExternalServiceStore.Upsert(ctx, services...); err != nil {
				t.Fatalf("failed to setup store: %v", err)
			}

			registry := ratelimit.NewRegistry()
			syncer := repos.NewRateLimitSyncer(registry, tx.ExternalServiceStore)
			err := syncer.SyncRateLimiters(ctx)
			if err != nil {
				t.Fatal(err)
			}
			have := registry.Count()
			if have != toCreate {
				t.Fatalf("Want %d, got %d", toCreate, have)
			}
		})(t)
	}
}

func testStoreEnqueueSyncJobs(store *repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		clock := timeutil.NewFakeClock(time.Now(), 0)
		now := clock.Now()

		services := generateExternalServices(10, mkExternalServices(now)...)

		type testCase struct {
			name            string
			stored          types.ExternalServices
			queued          func(types.ExternalServices) []int64
			ignoreSiteAdmin bool
			err             error
		}

		var testCases []testCase

		testCases = append(testCases, testCase{
			name: "enqueue everything",
			stored: services.With(func(s *types.ExternalService) {
				s.NextSyncAt = now.Add(-10 * time.Second)
			}),
			queued: func(svcs types.ExternalServices) []int64 { return svcs.IDs() },
		})

		testCases = append(testCases, testCase{
			name: "nothing to enqueue",
			stored: services.With(func(s *types.ExternalService) {
				s.NextSyncAt = now.Add(10 * time.Second)
			}),
			queued: func(svcs types.ExternalServices) []int64 { return []int64{} },
		})

		testCases = append(testCases, testCase{
			name: "ignore siteadmin repos",
			stored: services.With(func(s *types.ExternalService) {
				s.NextSyncAt = now.Add(10 * time.Second)
			}),
			ignoreSiteAdmin: true,
			queued:          func(svcs types.ExternalServices) []int64 { return []int64{} },
		})

		{
			i := 0
			testCases = append(testCases, testCase{
				name: "some to enqueue",
				stored: services.With(func(s *types.ExternalService) {
					if i%2 == 0 {
						s.NextSyncAt = now.Add(10 * time.Second)
					} else {
						s.NextSyncAt = now.Add(-10 * time.Second)
					}
					i++
				}),
				queued: func(svcs types.ExternalServices) []int64 {
					var ids []int64
					for i := range svcs {
						if i%2 != 0 {
							ids = append(ids, svcs[i].ID)
						}
					}
					return ids
				},
			})
		}

		for _, tc := range testCases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Cleanup(func() {
					if err := store.Exec(ctx, sqlf.Sprintf("DELETE FROM external_service_sync_jobs;DELETE FROM external_services")); err != nil {
						t.Fatal(err)
					}
				})
				stored := tc.stored.Clone()

				if err := store.ExternalServiceStore.Upsert(ctx, stored...); err != nil {
					t.Fatalf("failed to setup store: %v", err)
				}

				err := store.EnqueueSyncJobs(ctx, tc.ignoreSiteAdmin)
				if have, want := fmt.Sprint(err), fmt.Sprint(tc.err); have != want {
					t.Errorf("error:\nhave: %v\nwant: %v", have, want)
				}

				jobs, err := store.ListSyncJobs(ctx)
				if err != nil {
					t.Fatal(err)
				}

				gotIDs := make([]int64, 0, len(jobs))
				for _, job := range jobs {
					gotIDs = append(gotIDs, job.ExternalServiceID)
				}

				want := tc.queued(stored)
				sort.Slice(gotIDs, func(i, j int) bool {
					return gotIDs[i] < gotIDs[j]
				})
				sort.Slice(want, func(i, j int) bool {
					return want[i] < want[j]
				})

				if diff := cmp.Diff(want, gotIDs); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	}
}

func testStoreEnqueueSingleSyncJob(store *repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		clock := timeutil.NewFakeClock(time.Now(), 0)
		now := clock.Now()

		ctx := context.Background()
		t.Cleanup(func() {
			if err := store.Exec(ctx, sqlf.Sprintf("DELETE FROM external_service_sync_jobs;DELETE FROM external_services")); err != nil {
				t.Fatal(err)
			}
		})
		service := types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test",
			Config:      `{"url": "https://github.com", "repositoryQuery": ["none"], "token": "abc"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Create a new external service
		confGet := func() *conf.Unified {
			return &conf.Unified{}
		}
		err := database.ExternalServicesWith(store).Create(ctx, confGet, &service)
		if err != nil {
			t.Fatal(err)
		}

		assertCount := func(t *testing.T, want int) {
			t.Helper()
			var count int
			if err := store.QueryRow(ctx, sqlf.Sprintf("SELECT COUNT(*) FROM external_service_sync_jobs")).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != want {
				t.Fatalf("Expected %d rows, got %d", want, count)
			}
		}
		assertCount(t, 0)

		err = store.EnqueueSingleSyncJob(ctx, service.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertCount(t, 1)

		// Doing it again should not fail or add a new row
		err = store.EnqueueSingleSyncJob(ctx, service.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertCount(t, 1)

		// If we change status to processing it should not add a new row
		if err := store.Exec(ctx, sqlf.Sprintf("UPDATE external_service_sync_jobs SET state='processing'")); err != nil {
			t.Fatal(err)
		}
		err = store.EnqueueSingleSyncJob(ctx, service.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertCount(t, 1)

		// If we change status to completed we should be able to enqueue another one
		if err = store.Exec(ctx, sqlf.Sprintf("UPDATE external_service_sync_jobs SET state='completed'")); err != nil {
			t.Fatal(err)
		}
		err = store.EnqueueSingleSyncJob(ctx, service.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertCount(t, 2)
	}
}

func mkRepos(n int, base ...*types.Repo) types.Repos {
	if len(base) == 0 {
		return nil
	}

	rs := make(types.Repos, 0, n)
	for i := 0; i < n; i++ {
		id := strconv.Itoa(i)
		r := base[i%len(base)].Clone()
		r.Name += api.RepoName(id)
		r.ExternalRepo.ID += id
		rs = append(rs, r)
	}
	return rs
}

func generateExternalServices(n int, base ...*types.ExternalService) types.ExternalServices {
	if len(base) == 0 {
		return nil
	}
	es := make(types.ExternalServices, 0, n)
	for i := 0; i < n; i++ {
		id := strconv.Itoa(i)
		r := base[i%len(base)].Clone()
		r.DisplayName += id
		es = append(es, r)
	}
	return es
}

func transact(ctx context.Context, s *repos.Store, test func(testing.TB, *repos.Store)) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		var err error
		txStore := s

		if !s.InTransaction() {
			txStore, err = s.Transact(ctx)
			if err != nil {
				t.Fatalf("failed to start transaction: %v", err)
			}
			defer txStore.Done(errRollback)
		}

		test(t, txStore)
	}
}

func createExternalServices(t *testing.T, store *repos.Store, opts ...func(*types.ExternalService)) map[string]*types.ExternalService {
	clock := timeutil.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	svcs := mkExternalServices(now)
	for _, svc := range svcs {
		for _, opt := range opts {
			opt(svc)
		}
	}

	// create a few external services
	if err := store.ExternalServiceStore.Upsert(context.Background(), svcs...); err != nil {
		t.Fatalf("failed to insert external services: %v", err)
	}

	services, err := store.ExternalServiceStore.List(context.Background(), database.ExternalServicesListOptions{})
	if err != nil {
		t.Fatal("failed to list external services")
	}

	servicesPerKind := make(map[string]*types.ExternalService)
	for _, svc := range services {
		servicesPerKind[svc.Kind] = svc
	}

	return servicesPerKind
}

func mkExternalServices(now time.Time) types.ExternalServices {
	githubSvc := types.ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "Github - Test",
		Config:      `{"url": "https://github.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	gitlabSvc := types.ExternalService{
		Kind:        extsvc.KindGitLab,
		DisplayName: "GitLab - Test",
		Config:      `{"url": "https://gitlab.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	bitbucketServerSvc := types.ExternalService{
		Kind:        extsvc.KindBitbucketServer,
		DisplayName: "Bitbucket Server - Test",
		Config:      `{"url": "https://bitbucket.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	bitbucketCloudSvc := types.ExternalService{
		Kind:        extsvc.KindBitbucketCloud,
		DisplayName: "Bitbucket Cloud - Test",
		Config:      `{"url": "https://bitbucket.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	awsSvc := types.ExternalService{
		Kind:        extsvc.KindAWSCodeCommit,
		DisplayName: "AWS Code - Test",
		Config:      `{"url": "https://aws.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	otherSvc := types.ExternalService{
		Kind:        extsvc.KindOther,
		DisplayName: "Other - Test",
		Config:      `{"url": "https://other.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	gitoliteSvc := types.ExternalService{
		Kind:        extsvc.KindGitolite,
		DisplayName: "Gitolite - Test",
		Config:      `{"url": "https://gitolite.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return []*types.ExternalService{
		&githubSvc,
		&gitlabSvc,
		&bitbucketServerSvc,
		&bitbucketCloudSvc,
		&awsSvc,
		&otherSvc,
		&gitoliteSvc,
	}
}
