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
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
)

func TestChangesetResolver(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)

	userID := insertTestUser(t, dbconn.Global, "campaign-resolver", true)

	now := time.Now().UTC().Truncate(time.Microsecond)
	clock := func() time.Time { return now.UTC().Truncate(time.Microsecond) }
	store := ee.NewStoreWithClock(dbconn.Global, clock)
	rstore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	repo := newGitHubTestRepo("github.com/sourcegraph/sourcegraph", newGitHubExternalService(t, rstore))
	if err := rstore.InsertRepos(ctx, repo); err != nil {
		t.Fatal(err)
	}

	// We use the same mocks for both changesets, even though the unpublished
	// changesets doesn't have a HeadRev (since no commit has been made). The
	// PreviewRepositoryComparison uses a subset of the mocks, though.
	baseRev := "53339e93a17b7934abf3bc4aae3565c15a0631a9"
	headRev := "fa9e174e4847e5f551b31629542377395d6fc95a"
	// These are needed for preview repository comparisons.
	mockBackendCommits(t, api.CommitID(baseRev))
	mockRepoComparison(t, baseRev, headRev, testDiff)

	unpublishedSpec := createChangesetSpec(t, ctx, store, testSpecOpts{
		user:          userID,
		repo:          repo.ID,
		headRef:       "refs/heads/my-new-branch",
		published:     false,
		title:         "ChangesetSpec Title",
		body:          "ChangesetSpec Body",
		commitMessage: "The commit message",
		commitDiff:    testDiff,
		baseRev:       baseRev,
		baseRef:       "refs/heads/master",
	})
	unpublishedChangeset := createChangeset(t, ctx, store, testChangesetOpts{
		repo:                repo.ID,
		currentSpec:         unpublishedSpec.ID,
		externalServiceType: "github",
		publicationState:    campaigns.ChangesetPublicationStateUnpublished,
		reconcilerState:     campaigns.ReconcilerStateCompleted,
	})
	erroredSpec := createChangesetSpec(t, ctx, store, testSpecOpts{
		user:          userID,
		repo:          repo.ID,
		headRef:       "refs/heads/my-failing-branch",
		published:     true,
		title:         "ChangesetSpec Title",
		body:          "ChangesetSpec Body",
		commitMessage: "The commit message",
		commitDiff:    testDiff,
		baseRev:       baseRev,
		baseRef:       "refs/heads/master",
	})
	erroredChangeset := createChangeset(t, ctx, store, testChangesetOpts{
		repo:                repo.ID,
		currentSpec:         erroredSpec.ID,
		externalServiceType: "github",
		publicationState:    campaigns.ChangesetPublicationStateUnpublished,
		reconcilerState:     campaigns.ReconcilerStateErrored,
		failureMessage:      "very bad error",
	})

	labelEventDescriptionText := "the best label in town"

	syncedGitHubChangeset := createChangeset(t, ctx, store, testChangesetOpts{
		repo:                repo.ID,
		externalServiceType: "github",
		externalID:          "12345",
		externalBranch:      "open-pr",
		externalState:       campaigns.ChangesetExternalStateOpen,
		externalCheckState:  campaigns.ChangesetCheckStatePending,
		externalReviewState: campaigns.ChangesetReviewStateChangesRequested,
		publicationState:    campaigns.ChangesetPublicationStatePublished,
		reconcilerState:     campaigns.ReconcilerStateCompleted,
		metadata: &github.PullRequest{
			ID:          "12345",
			Title:       "GitHub PR Title",
			Body:        "GitHub PR Body",
			Number:      12345,
			State:       "OPEN",
			URL:         "https://github.com/sourcegraph/sourcegraph/pull/12345",
			HeadRefName: "open-pr",
			HeadRefOid:  headRev,
			BaseRefOid:  baseRev,
			BaseRefName: "master",
			TimelineItems: []github.TimelineItem{
				{Type: "PullRequestCommit", Item: &github.PullRequestCommit{
					Commit: github.Commit{
						OID:           "d34db33f",
						PushedDate:    now,
						CommittedDate: now,
					},
				}},
				{Type: "LabeledEvent", Item: &github.LabelEvent{
					CreatedAt: now.Add(5 * time.Second),
					Label: github.Label{
						ID:          "label-event",
						Name:        "cool-label",
						Color:       "blue",
						Description: labelEventDescriptionText,
					},
				}},
			},
			Labels: struct{ Nodes []github.Label }{
				Nodes: []github.Label{
					{ID: "label-no-description", Name: "no-description", Color: "121212"},
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	})
	events := syncedGitHubChangeset.Events()
	if err := store.UpsertChangesetEvents(ctx, events...); err != nil {
		t.Fatal(err)
	}

	unsyncedChangeset := createChangeset(t, ctx, store, testChangesetOpts{
		repo:                repo.ID,
		externalServiceType: "github",
		externalID:          "9876",
		unsynced:            true,
		publicationState:    campaigns.ChangesetPublicationStatePublished,
		reconcilerState:     campaigns.ReconcilerStateQueued,
	})

	spec := &campaigns.CampaignSpec{
		UserID:          userID,
		NamespaceUserID: userID,
	}
	if err := store.CreateCampaignSpec(ctx, spec); err != nil {
		t.Fatal(err)
	}

	campaign := &campaigns.Campaign{
		Name:             "my-unique-name",
		NamespaceUserID:  userID,
		InitialApplierID: userID,
		CampaignSpecID:   spec.ID,
		LastApplierID:    userID,
		LastAppliedAt:    time.Now(),
	}
	if err := store.CreateCampaign(ctx, campaign); err != nil {
		t.Fatal(err)
	}
	// Associate the changeset with a campaign, so it's considered in syncer logic.
	addChangeset(t, ctx, store, campaign, syncedGitHubChangeset.ID)

	s, err := graphqlbackend.NewSchema(&Resolver{store: store}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		changeset *campaigns.Changeset
		want      apitest.Changeset
	}{
		{
			name:      "unpublished changeset",
			changeset: unpublishedChangeset,
			want: apitest.Changeset{
				Typename:   "ExternalChangeset",
				Title:      unpublishedSpec.Spec.Title,
				Body:       unpublishedSpec.Spec.Body,
				Repository: apitest.Repository{Name: repo.Name},
				// Not scheduled for sync, because it's not published.
				NextSyncAt: "",
				Labels:     []apitest.Label{},
				Diff: apitest.Comparison{
					Typename:  "PreviewRepositoryComparison",
					FileDiffs: testDiffGraphQL,
				},
				PublicationState: string(campaigns.ChangesetPublicationStateUnpublished),
				ReconcilerState:  string(campaigns.ReconcilerStateCompleted),
				CurrentSpec:      apitest.ChangesetSpec{ID: string(marshalChangesetSpecRandID(unpublishedSpec.RandID))},
			},
		},
		{
			name:      "errored changeset",
			changeset: erroredChangeset,
			want: apitest.Changeset{
				Typename:   "ExternalChangeset",
				Title:      erroredSpec.Spec.Title,
				Body:       erroredSpec.Spec.Body,
				Repository: apitest.Repository{Name: repo.Name},
				// Not scheduled for sync, because it's not published.
				NextSyncAt: "",
				Labels:     []apitest.Label{},
				Diff: apitest.Comparison{
					Typename:  "PreviewRepositoryComparison",
					FileDiffs: testDiffGraphQL,
				},
				PublicationState: string(campaigns.ChangesetPublicationStateUnpublished),
				ReconcilerState:  string(campaigns.ReconcilerStateErrored),
				Error:            "very bad error",
				CurrentSpec:      apitest.ChangesetSpec{ID: string(marshalChangesetSpecRandID(erroredSpec.RandID))},
			},
		},
		{
			name:      "synced github changeset",
			changeset: syncedGitHubChangeset,
			want: apitest.Changeset{
				Typename:      "ExternalChangeset",
				Title:         "GitHub PR Title",
				Body:          "GitHub PR Body",
				ExternalState: "OPEN",
				ExternalID:    "12345",
				CheckState:    "PENDING",
				ReviewState:   "CHANGES_REQUESTED",
				NextSyncAt:    marshalDateTime(t, now.Add(8*time.Hour)),
				Repository:    apitest.Repository{Name: repo.Name},
				ExternalURL: apitest.ExternalURL{
					URL:         "https://github.com/sourcegraph/sourcegraph/pull/12345",
					ServiceType: "github",
				},
				PublicationState: string(campaigns.ChangesetPublicationStatePublished),
				ReconcilerState:  string(campaigns.ReconcilerStateCompleted),
				Events: apitest.ChangesetEventConnection{
					TotalCount: 2,
				},
				Labels: []apitest.Label{
					{Text: "cool-label", Color: "blue", Description: &labelEventDescriptionText},
					{Text: "no-description", Color: "121212", Description: nil},
				},
				Diff: apitest.Comparison{
					Typename:  "RepositoryComparison",
					FileDiffs: testDiffGraphQL,
				},
			},
		},
		{
			name:      "unsynced changeset",
			changeset: unsyncedChangeset,
			want: apitest.Changeset{
				Typename:         "ExternalChangeset",
				ExternalID:       "9876",
				Repository:       apitest.Repository{Name: repo.Name},
				Labels:           []apitest.Label{},
				PublicationState: string(campaigns.ChangesetPublicationStatePublished),
				ReconcilerState:  string(campaigns.ReconcilerStateQueued),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			apiID := marshalChangesetID(tc.changeset.ID)
			input := map[string]interface{}{"changeset": apiID}

			var response struct{ Node apitest.Changeset }
			apitest.MustExec(ctx, t, s, input, &response, queryChangeset)

			tc.want.ID = string(apiID)
			if diff := cmp.Diff(tc.want, response.Node); diff != "" {
				t.Fatalf("wrong campaign response (-want +got):\n%s", diff)
			}
		})
	}
}

const queryChangeset = `
fragment fileDiffNode on FileDiff {
    oldPath
    newPath
    oldFile { name }
    hunks {
      body
      oldRange { startLine, lines }
      newRange { startLine, lines }
    }
    stat { added, changed, deleted }
}

query($changeset: ID!) {
  node(id: $changeset) {
    __typename

    ... on ExternalChangeset {
      id

      title
      body

      externalID
      externalState
      reviewState
      checkState
      externalURL { url, serviceType }
      nextSyncAt
      publicationState
      reconcilerState
      error

      repository { name }

      events(first: 100) { totalCount }
      labels { text, color, description }

      currentSpec { id }

      diff {
        __typename

        ... on RepositoryComparison {
          fileDiffs {
             totalCount
             rawDiff
             diffStat { added, changed, deleted }
             nodes {
               ... fileDiffNode
             }
          }
        }

        ... on PreviewRepositoryComparison {
          fileDiffs {
             totalCount
             rawDiff
             diffStat { added, changed, deleted }
             nodes {
               ... fileDiffNode
             }
          }
        }
      }
    }
  }
}
`
