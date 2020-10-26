package resolvers

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	ee "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/resolvers/apitest"
	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/testing"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

func TestChangesetSpecResolver(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)

	userID := insertTestUser(t, dbconn.Global, "changeset-spec-by-id", false)

	store := ee.NewStore(dbconn.Global)
	reposStore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	// Creating user with matching email to the changeset spec author.
	user, err := db.Users.Create(ctx, db.NewUser{
		Username:        "mary",
		Email:           ct.ChangesetSpecAuthorEmail,
		EmailIsVerified: true,
		DisplayName:     "Mary Tester",
	})
	if err != nil {
		t.Fatal(err)
	}

	repo := newGitHubTestRepo("github.com/sourcegraph/sourcegraph", newGitHubExternalService(t, reposStore))
	if err := reposStore.InsertRepos(ctx, repo); err != nil {
		t.Fatal(err)
	}
	repoID := graphqlbackend.MarshalRepositoryID(repo.ID)

	testRev := api.CommitID("b69072d5f687b31b9f6ae3ceafdc24c259c4b9ec")
	mockBackendCommits(t, testRev)

	s, err := graphqlbackend.NewSchema(&Resolver{store: store}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		rawSpec string
		want    func(spec *campaigns.ChangesetSpec) apitest.ChangesetSpec
	}{
		{
			name:    "GitBranchChangesetDescription",
			rawSpec: ct.NewRawChangesetSpecGitBranch(repoID, string(testRev)),
			want: func(spec *campaigns.ChangesetSpec) apitest.ChangesetSpec {
				return apitest.ChangesetSpec{
					Typename: "VisibleChangesetSpec",
					ID:       string(marshalChangesetSpecRandID(spec.RandID)),
					Description: apitest.ChangesetSpecDescription{
						Typename: "GitBranchChangesetDescription",
						BaseRepository: apitest.Repository{
							ID: string(spec.Spec.BaseRepository),
						},
						ExternalID: "",
						BaseRef:    git.AbbreviateRef(spec.Spec.BaseRef),
						HeadRepository: apitest.Repository{
							ID: string(spec.Spec.HeadRepository),
						},
						HeadRef: git.AbbreviateRef(spec.Spec.HeadRef),
						Title:   spec.Spec.Title,
						Body:    spec.Spec.Body,
						Commits: []apitest.GitCommitDescription{
							{
								Author: apitest.Person{
									Email: spec.Spec.Commits[0].AuthorEmail,
									Name:  user.Username,
									User: &apitest.User{
										ID: string(graphqlbackend.MarshalUserID(user.ID)),
									},
								},
								Diff:    spec.Spec.Commits[0].Diff,
								Message: spec.Spec.Commits[0].Message,
								Subject: "git commit message",
								Body:    "and some more content in a second paragraph.",
							},
						},
						Published: campaigns.PublishedValue{Val: false},
						Diff: struct{ FileDiffs apitest.FileDiffs }{
							FileDiffs: apitest.FileDiffs{
								DiffStat: apitest.DiffStat{
									Added:   1,
									Deleted: 1,
									Changed: 2,
								},
							},
						},
						DiffStat: apitest.DiffStat{
							Added:   1,
							Deleted: 1,
							Changed: 2,
						},
					},
					ExpiresAt: &graphqlbackend.DateTime{Time: spec.ExpiresAt().Truncate(time.Second)},
				}
			},
		},
		{
			name:    "GitBranchChangesetDescription Draft",
			rawSpec: ct.NewPublishedRawChangesetSpecGitBranch(repoID, string(testRev), campaigns.PublishedValue{Val: "draft"}),
			want: func(spec *campaigns.ChangesetSpec) apitest.ChangesetSpec {
				return apitest.ChangesetSpec{
					Typename: "VisibleChangesetSpec",
					ID:       string(marshalChangesetSpecRandID(spec.RandID)),
					Description: apitest.ChangesetSpecDescription{
						Typename: "GitBranchChangesetDescription",
						BaseRepository: apitest.Repository{
							ID: string(spec.Spec.BaseRepository),
						},
						ExternalID: "",
						BaseRef:    git.AbbreviateRef(spec.Spec.BaseRef),
						HeadRepository: apitest.Repository{
							ID: string(spec.Spec.HeadRepository),
						},
						HeadRef: git.AbbreviateRef(spec.Spec.HeadRef),
						Title:   spec.Spec.Title,
						Body:    spec.Spec.Body,
						Commits: []apitest.GitCommitDescription{
							{
								Author: apitest.Person{
									Email: spec.Spec.Commits[0].AuthorEmail,
									Name:  user.Username,
									User: &apitest.User{
										ID: string(graphqlbackend.MarshalUserID(user.ID)),
									},
								},
								Diff:    spec.Spec.Commits[0].Diff,
								Message: spec.Spec.Commits[0].Message,
								Subject: "git commit message",
								Body:    "and some more content in a second paragraph.",
							},
						},
						Published: campaigns.PublishedValue{Val: "draft"},
						Diff: struct{ FileDiffs apitest.FileDiffs }{
							FileDiffs: apitest.FileDiffs{
								DiffStat: apitest.DiffStat{
									Added:   1,
									Deleted: 1,
									Changed: 2,
								},
							},
						},
						DiffStat: apitest.DiffStat{
							Added:   1,
							Deleted: 1,
							Changed: 2,
						},
					},
					ExpiresAt: &graphqlbackend.DateTime{Time: spec.ExpiresAt().Truncate(time.Second)},
				}
			},
		},
		{
			name:    "ExistingChangesetReference",
			rawSpec: ct.NewRawChangesetSpecExisting(repoID, "9999"),
			want: func(spec *campaigns.ChangesetSpec) apitest.ChangesetSpec {
				return apitest.ChangesetSpec{
					Typename: "VisibleChangesetSpec",
					ID:       string(marshalChangesetSpecRandID(spec.RandID)),
					Description: apitest.ChangesetSpecDescription{
						Typename: "ExistingChangesetReference",
						BaseRepository: apitest.Repository{
							ID: string(spec.Spec.BaseRepository),
						},
						ExternalID: spec.Spec.ExternalID,
					},
					ExpiresAt: &graphqlbackend.DateTime{Time: spec.ExpiresAt().Truncate(time.Second)},
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := campaigns.NewChangesetSpecFromRaw(tc.rawSpec)
			if err != nil {
				t.Fatal(err)
			}
			spec.UserID = userID
			spec.RepoID = repo.ID

			if err := store.CreateChangesetSpec(ctx, spec); err != nil {
				t.Fatal(err)
			}

			input := map[string]interface{}{"id": marshalChangesetSpecRandID(spec.RandID)}
			var response struct{ Node apitest.ChangesetSpec }
			apitest.MustExec(ctx, t, s, input, &response, queryChangesetSpecNode)

			want := tc.want(spec)
			if diff := cmp.Diff(want, response.Node); diff != "" {
				t.Fatalf("unexpected response (-want +got):\n%s", diff)
			}
		})
	}
}

const queryChangesetSpecNode = `
query($id: ID!) {
  node(id: $id) {
    __typename

    ... on VisibleChangesetSpec {
      id

      description {
        __typename

        ... on ExistingChangesetReference {
          baseRepository {
             id
          }
          externalID
        }

        ... on GitBranchChangesetDescription {
          baseRepository {
              id
          }
          baseRef
          baseRev

          headRepository {
              id
          }
          headRef

          title
          body

          commits {
            message
            subject
            body
            diff
            author {
              name
              email
              user {
                id
              }
            }
          }

          published

          diff {
            fileDiffs {
              diffStat { added, changed, deleted }
            }
          }
          diffStat { added, changed, deleted }
        }
      }

      expiresAt
    }
  }
}
`
