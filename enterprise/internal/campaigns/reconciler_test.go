package campaigns

import (
	"context"
	"strings"
	"time"

	"testing"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/testing"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

func TestReconcilerProcess(t *testing.T) {
	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)

	now := time.Now().UTC().Truncate(time.Microsecond)
	clock := func() time.Time {
		return now.UTC().Truncate(time.Microsecond)
	}
	store := NewStoreWithClock(dbconn.Global, clock)

	admin := createTestUser(ctx, t)
	if !admin.SiteAdmin {
		t.Fatalf("admin is not site admin")
	}

	rs, extSvc := createTestRepos(t, ctx, dbconn.Global, 1)

	state := ct.MockChangesetSyncState(&protocol.RepoInfo{
		Name: api.RepoName(rs[0].Name),
		VCS:  protocol.VCSInfo{URL: rs[0].URI},
	})
	defer state.Unmock()

	internalClient = &mockInternalClient{externalURL: "https://sourcegraph.test"}
	defer func() { internalClient = api.InternalClient }()

	githubPR := buildGithubPR(clock(), campaigns.ChangesetExternalStateOpen)
	draftGithubPR := buildGithubPR(clock(), campaigns.ChangesetExternalStateDraft)
	closedGitHubPR := buildGithubPR(clock(), campaigns.ChangesetExternalStateClosed)

	notFoundErr := repos.ChangesetNotFoundError{
		Changeset: &repos.Changeset{
			Changeset: &campaigns.Changeset{ExternalID: "100000"},
		},
	}
	notFoundErrMsg := notFoundErr.Error()

	campaignSpec := createCampaignSpec(t, ctx, store, "reconciler-test-campaign", admin.ID)
	campaign := createCampaign(t, ctx, store, "reconciler-test-campaign", admin.ID, campaignSpec.ID)

	type testCase struct {
		changeset    testChangesetOpts
		currentSpec  *testSpecOpts
		previousSpec *testSpecOpts

		sourcerMetadata interface{}
		sourcerErr      error
		// Whether or not the source responds to CreateChangeset with "already exists"
		alreadyExists bool

		wantCreateOnCodeHost      bool
		wantCreateDraftOnCodeHost bool
		wantUndraftOnCodeHost     bool
		wantUpdateOnCodeHost      bool
		wantCloseOnCodeHost       bool
		wantLoadFromCodeHost      bool
		wantReopenOnCodeHost      bool

		wantGitserverCommit bool

		wantChangeset changesetAssertions
	}

	tests := map[string]testCase{
		"published unsynced changeset without changesetSpec": {
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       githubPR.ID,
				unsynced:         true,
			},
			sourcerMetadata: githubPR,

			wantLoadFromCodeHost: true,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				externalState:    campaigns.ChangesetExternalStateOpen,
				unsynced:         false,
				title:            githubPR.Title,
				body:             githubPR.Body,
				diffStat:         state.DiffStat,
			},
		},
		"unpublished changeset stay unpublished": {
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/repo-1-branch-1",
				published: false,
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStateUnpublished,
			},
			sourcerMetadata: githubPR,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStateUnpublished,
				externalState:    "",
				externalID:       "",
				externalBranch:   "",
			},
		},
		"publish changeset": {
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStateUnpublished,
				ownedByCampaign:  campaign.ID,
			},
			sourcerMetadata: githubPR,

			wantCreateOnCodeHost: true,
			wantGitserverCommit:  true,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				externalState:    campaigns.ChangesetExternalStateOpen,
				title:            githubPR.Title,
				body:             githubPR.Body,
				diffStat:         state.DiffStat,
				ownedByCampaign:  campaign.ID,
			},
		},
		"retry publish changeset": {
			// This test case makes sure that everything works when the code host says
			// that the changeset already exists.
			alreadyExists: true,
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,
			},
			changeset: testChangesetOpts{
				failureMessage:   "publication failed",
				publicationState: campaigns.ChangesetPublicationStateUnpublished,
				ownedByCampaign:  campaign.ID,
			},
			sourcerMetadata: githubPR,

			// We first do a create and since that fails with "already exists"
			// we update.
			wantCreateOnCodeHost: true,
			wantUpdateOnCodeHost: true,
			wantGitserverCommit:  true,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				externalState:    campaigns.ChangesetExternalStateOpen,
				title:            githubPR.Title,
				body:             githubPR.Body,
				diffStat:         state.DiffStat,
				ownedByCampaign:  campaign.ID,
			},
		},
		"update published changeset metadata": {
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				title: "new title",
				body:  "new body",
			},
			previousSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				title: "old title",
				body:  "old body",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       "12345",
				externalBranch:   "head-ref-on-github",
				ownedByCampaign:  campaign.ID,
			},
			sourcerMetadata: githubPR,

			// We don't want a new commit, only an update on the code host.
			wantUpdateOnCodeHost: true,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				externalState:    campaigns.ChangesetExternalStateOpen,
				diffStat:         state.DiffStat,
				ownedByCampaign:  campaign.ID,
				// We update the title/body but want the title/body returned by the code host.
				title: githubPR.Title,
				body:  githubPR.Body,
			},
		},
		"retry update published changeset metadata": {
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				title: "new title",
				body:  "new body",
			},
			previousSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				title: "old title",
				body:  "old body",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				externalState:    campaigns.ChangesetExternalStateOpen,
				ownedByCampaign:  campaign.ID,
				// Previous update failed:
				failureMessage: "failed to update changeset metadata",
			},
			sourcerMetadata: githubPR,

			wantUpdateOnCodeHost: true,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				externalState:    campaigns.ChangesetExternalStateOpen,
				title:            githubPR.Title,
				body:             githubPR.Body,
				diffStat:         state.DiffStat,
				ownedByCampaign:  campaign.ID,
				// failureMessage should be nil
			},
		},
		"update published changeset commit": {
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				// Title and body the same, but commit changed
				commitDiff:    "new diff",
				commitMessage: "new message",
			},
			previousSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				commitDiff:    "old diff",
				commitMessage: "old message",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       "12345",
				externalBranch:   "head-ref-on-github",
				externalState:    campaigns.ChangesetExternalStateOpen,
				ownedByCampaign:  campaign.ID,
			},
			sourcerMetadata: githubPR,

			// We don't want an update on the code host, only a new commit pushed.
			wantGitserverCommit: true,
			// And we want the changeset to be synced after pushing the commit.
			wantLoadFromCodeHost: true,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalState:    campaigns.ChangesetExternalStateOpen,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				diffStat:         state.DiffStat,
				ownedByCampaign:  campaign.ID,
			},
		},
		"retry update published changeset commit": {
			currentSpec: &testSpecOpts{
				headRef:       "refs/heads/head-ref-on-github",
				published:     true,
				commitDiff:    "new diff",
				commitMessage: "new message",
			},
			previousSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				commitDiff:    "old diff",
				commitMessage: "old message",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       "12345",
				externalBranch:   "head-ref-on-github",
				externalState:    campaigns.ChangesetExternalStateOpen,
				ownedByCampaign:  campaign.ID,

				// Previous update failed:
				failureMessage: "failed to update changeset commit",
			},
			sourcerMetadata: githubPR,

			wantGitserverCommit:  true,
			wantLoadFromCodeHost: true,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalState:    campaigns.ChangesetExternalStateOpen,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				diffStat:         state.DiffStat,
				ownedByCampaign:  campaign.ID,
				// failureMessage should be nil
			},
		},
		"update published changeset commit author": {
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				// Everything the same, except author changed
				commitAuthorName:  "Fernando the fish",
				commitAuthorEmail: "fernando@deep.sea",
			},
			previousSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				// Old author data
				commitAuthorName:  "Larry the Llama",
				commitAuthorEmail: "larry@winamp.com",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       "12345",
				externalBranch:   "head-ref-on-github",
				externalState:    campaigns.ChangesetExternalStateOpen,
				ownedByCampaign:  campaign.ID,
			},
			sourcerMetadata: githubPR,

			// We don't want an update on the code host, only a new commit pushed.
			wantGitserverCommit: true,
			// And we want the changeset to be synced after pushing the commit.
			wantLoadFromCodeHost: true,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalState:    campaigns.ChangesetExternalStateOpen,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				diffStat:         state.DiffStat,
				ownedByCampaign:  campaign.ID,
			},
		},
		"reprocess published changeset without changes": {
			// ChangesetSpec is already published and has no previous spec.
			// Simply a reprocessing of the same changeset.
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				title: "title",
				body:  "body",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				externalState:    campaigns.ChangesetExternalStateOpen,
			},
			sourcerMetadata: githubPR,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				externalState:    campaigns.ChangesetExternalStateOpen,
			},
		},
		"closing published open changeset": {
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				title: "title",
				body:  "body",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				externalState:    campaigns.ChangesetExternalStateOpen,
				closing:          true,
				ownedByCampaign:  campaign.ID,
			},
			// We return a closed GitHub PR here
			sourcerMetadata: closedGitHubPR,

			wantCloseOnCodeHost: true,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				closing:          false,

				externalID:     closedGitHubPR.ID,
				externalBranch: closedGitHubPR.HeadRefName,
				externalState:  campaigns.ChangesetExternalStateClosed,

				title:    closedGitHubPR.Title,
				body:     closedGitHubPR.Body,
				diffStat: state.DiffStat,

				ownedByCampaign: campaign.ID,
			},
		},
		"closing non-open changeset": {
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				title: "title",
				body:  "body",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				externalState:    campaigns.ChangesetExternalStateClosed,
				closing:          true,
				ownedByCampaign:  campaign.ID,
			},
			// We return a closed GitHub PR here, but since it's a noop, we
			// don't sync and thus don't set its attributes on the changeset.
			sourcerMetadata: closedGitHubPR,

			// Should be a noop
			wantCloseOnCodeHost: false,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				closing:          false,

				externalID:     closedGitHubPR.ID,
				externalBranch: closedGitHubPR.HeadRefName,
				externalState:  campaigns.ChangesetExternalStateClosed,
			},
		},
		"reopening closed changeset without updates": {
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				title: "title",
				body:  "body",
			},
			previousSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				title: "title",
				body:  "body",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				externalState:    campaigns.ChangesetExternalStateClosed,
				ownedByCampaign:  campaign.ID,
				closing:          false,
			},
			// We return the open GitHub PR here
			sourcerMetadata: githubPR,

			wantReopenOnCodeHost: true,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,

				externalID:     githubPR.ID,
				externalBranch: githubPR.HeadRefName,
				externalState:  campaigns.ChangesetExternalStateOpen,

				title:    githubPR.Title,
				body:     githubPR.Body,
				diffStat: state.DiffStat,
			},
		},

		"reopening closed changeset with updates": {
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				title: "title",
				body:  "body",
			},
			previousSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,

				title: "old title",
				body:  "old body",

				commitDiff:    "old diff",
				commitMessage: "old message",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       githubPR.ID,
				externalBranch:   githubPR.HeadRefName,
				externalState:    campaigns.ChangesetExternalStateClosed,
				ownedByCampaign:  campaign.ID,
				closing:          false,
			},
			sourcerMetadata: githubPR,

			// Reopen it
			wantReopenOnCodeHost: true,
			// Update the metadata
			wantUpdateOnCodeHost: true,
			// Update the commit
			wantGitserverCommit: true,

			wantLoadFromCodeHost: false,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,

				externalID:     githubPR.ID,
				externalBranch: githubPR.HeadRefName,
				externalState:  campaigns.ChangesetExternalStateOpen,

				title:    githubPR.Title,
				body:     githubPR.Body,
				diffStat: state.DiffStat,
			},
		},

		"publish as draft mode for supported codehost": {
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: "draft",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStateUnpublished,
				ownedByCampaign:  campaign.ID,
			},
			sourcerMetadata: draftGithubPR,

			// Update the commit
			wantGitserverCommit:       true,
			wantCreateDraftOnCodeHost: true,

			wantLoadFromCodeHost: false,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,

				externalID:     draftGithubPR.ID,
				externalBranch: draftGithubPR.HeadRefName,
				externalState:  campaigns.ChangesetExternalStateDraft,

				title:    draftGithubPR.Title,
				body:     draftGithubPR.Body,
				diffStat: state.DiffStat,
			},
		},

		"published false to published draft": {
			previousSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: false,
			},
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: "draft",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStateUnpublished,
				ownedByCampaign:  campaign.ID,
			},
			sourcerMetadata: draftGithubPR,

			// Update the commit
			wantGitserverCommit:       true,
			wantCreateDraftOnCodeHost: true,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,

				externalID:     draftGithubPR.ID,
				externalBranch: draftGithubPR.HeadRefName,
				externalState:  campaigns.ChangesetExternalStateDraft,

				title:    draftGithubPR.Title,
				body:     draftGithubPR.Body,
				diffStat: state.DiffStat,
			},
		},

		"undraft a changeset": {
			currentSpec: &testSpecOpts{
				headRef:   "refs/heads/head-ref-on-github",
				published: true,
			},
			previousSpec: &testSpecOpts{
				published: "draft",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalState:    campaigns.ChangesetExternalStateDraft,
				ownedByCampaign:  campaign.ID,
			},
			sourcerMetadata: githubPR,

			wantUndraftOnCodeHost: true,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,

				externalID:     githubPR.ID,
				externalBranch: githubPR.HeadRefName,
				externalState:  campaigns.ChangesetExternalStateOpen,

				title:    githubPR.Title,
				body:     githubPR.Body,
				diffStat: state.DiffStat,
			},
		},
		"syncing not found changeset": {
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				externalID:       "100000",
				unsynced:         true,
			},
			sourcerErr: notFoundErr,

			wantLoadFromCodeHost: true,

			wantChangeset: changesetAssertions{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				failureMessage:   &notFoundErrMsg,
				externalID:       "100000",
				reconcilerState:  campaigns.ReconcilerStateErrored,
				numFailures:      reconcilerMaxNumRetries + 999,
				unsynced:         true,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Clean up database.
			truncateTables(t, dbconn.Global, "changeset_events", "changesets", "campaigns", "campaign_specs", "changeset_specs")

			// Create necessary associations.
			campaignSpec := createCampaignSpec(t, ctx, store, "reconciler-test-campaign", admin.ID)
			campaign := createCampaign(t, ctx, store, "reconciler-test-campaign", admin.ID, campaignSpec.ID)

			// Create the changesetSpec with associations wired up correctly.
			var changesetSpec *campaigns.ChangesetSpec
			if tc.currentSpec != nil {
				specOpts := *tc.currentSpec
				specOpts.user = admin.ID
				specOpts.repo = rs[0].ID
				specOpts.campaignSpec = campaignSpec.ID
				changesetSpec = createChangesetSpec(t, ctx, store, specOpts)
			}

			// If we need a previous spec, we need to set that up too.
			var previousSpec *campaigns.ChangesetSpec
			if tc.previousSpec != nil {
				previousCampaignSpec := createCampaignSpec(t, ctx, store, "previous-campaign-spec", admin.ID)
				specOpts := *tc.previousSpec
				specOpts.user = admin.ID
				specOpts.repo = rs[0].ID
				specOpts.campaignSpec = previousCampaignSpec.ID
				previousSpec = createChangesetSpec(t, ctx, store, specOpts)
			}

			// Create the changeset with correct associations.
			changesetOpts := tc.changeset
			changesetOpts.repo = rs[0].ID
			changesetOpts.campaign = campaign.ID
			if changesetSpec != nil {
				changesetOpts.currentSpec = changesetSpec.ID
			}
			if previousSpec != nil {
				changesetOpts.previousSpec = previousSpec.ID
			}
			changeset := createChangeset(t, ctx, store, changesetOpts)

			// Setup gitserver dependency.
			gitClient := &ct.FakeGitserverClient{ResponseErr: nil}
			if changesetSpec != nil {
				gitClient.Response = changesetSpec.Spec.HeadRef
			}

			// Setup the sourcer that's used to create a Source with which
			// to create/update a changeset.
			fakeSource := &ct.FakeChangesetSource{
				Svc:             extSvc,
				Err:             tc.sourcerErr,
				ChangesetExists: tc.alreadyExists,
				FakeMetadata:    tc.sourcerMetadata,
			}
			if changesetSpec != nil {
				fakeSource.WantHeadRef = changesetSpec.Spec.HeadRef
				fakeSource.WantBaseRef = changesetSpec.Spec.BaseRef
			}

			sourcer := repos.NewFakeSourcer(nil, fakeSource)

			// Run the reconciler
			rec := reconciler{
				noSleepBeforeSync: true,
				gitserverClient:   gitClient,
				sourcer:           sourcer,
				store:             store,
			}
			if err := rec.process(ctx, store, changeset); err != nil {
				t.Fatalf("reconciler process failed: %s", err)
			}

			// Assert that all the calls happened
			if have, want := gitClient.CreateCommitFromPatchCalled, tc.wantGitserverCommit; have != want {
				t.Fatalf("wrong CreateCommitFromPatch call. wantCalled=%t, wasCalled=%t", want, have)
			}

			if have, want := fakeSource.CreateDraftChangesetCalled, tc.wantCreateDraftOnCodeHost; have != want {
				t.Fatalf("wrong CreateDraftChangeset call. wantCalled=%t, wasCalled=%t", want, have)
			}

			if have, want := fakeSource.UndraftedChangesetsCalled, tc.wantUndraftOnCodeHost; have != want {
				t.Fatalf("wrong UndraftChangeset call. wantCalled=%t, wasCalled=%t", want, have)
			}

			if have, want := fakeSource.CreateChangesetCalled, tc.wantCreateOnCodeHost; have != want {
				t.Fatalf("wrong CreateChangeset call. wantCalled=%t, wasCalled=%t", want, have)
			}

			if have, want := fakeSource.UpdateChangesetCalled, tc.wantUpdateOnCodeHost; have != want {
				t.Fatalf("wrong UpdateChangeset call. wantCalled=%t, wasCalled=%t", want, have)
			}

			if have, want := fakeSource.ReopenChangesetCalled, tc.wantReopenOnCodeHost; have != want {
				t.Fatalf("wrong ReopenChangeset call. wantCalled=%t, wasCalled=%t", want, have)
			}

			if have, want := fakeSource.LoadChangesetCalled, tc.wantLoadFromCodeHost; have != want {
				t.Fatalf("wrong LoadChangeset call. wantCalled=%t, wasCalled=%t", want, have)
			}

			if have, want := fakeSource.CloseChangesetCalled, tc.wantCloseOnCodeHost; have != want {
				t.Fatalf("wrong CloseChangeset call. wantCalled=%t, wasCalled=%t", want, have)
			}

			// Assert that the changeset in the database looks like we want
			assertions := tc.wantChangeset
			assertions.repo = rs[0].ID
			assertions.ownedByCampaign = changesetOpts.ownedByCampaign
			if changesetSpec != nil {
				assertions.currentSpec = changesetSpec.ID
			}
			if previousSpec != nil {
				assertions.previousSpec = previousSpec.ID
			}
			reloadAndAssertChangeset(t, ctx, store, changeset, assertions)

			// Assert that the body included a backlink if needed. We'll do
			// more detailed unit tests of decorateChangesetBody elsewhere;
			// we're just looking for a basic marker here that _something_
			// happened.
			var rcs *repos.Changeset
			if tc.wantCreateOnCodeHost && fakeSource.CreateChangesetCalled {
				rcs = fakeSource.CreatedChangesets[0]
			} else if tc.wantUpdateOnCodeHost && fakeSource.UpdateChangesetCalled {
				rcs = fakeSource.UpdatedChangesets[0]
			}

			if rcs != nil {
				if !strings.Contains(rcs.Body, "Created by Sourcegraph campaign") {
					t.Errorf("did not find backlink in body: %q", rcs.Body)
				}
			}
		})
	}
}

func TestDeterminePlan(t *testing.T) {
	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)

	store := NewStore(dbconn.Global)

	rs, _ := createTestRepos(t, ctx, dbconn.Global, 1)
	githubRepo := rs[0]

	rs, _ = createBbsTestRepos(t, ctx, dbconn.Global, 1)
	bbsRepo := rs[0]

	admin := createTestUser(ctx, t)
	if !admin.SiteAdmin {
		t.Fatalf("admin is not site admin")
	}

	campaignSpec := createCampaignSpec(t, ctx, store, "test-plan", admin.ID)
	createCampaign(t, ctx, store, "test-plan", admin.ID, campaignSpec.ID)

	tcs := []struct {
		name           string
		previousSpec   testSpecOpts
		currentSpec    testSpecOpts
		changeset      testChangesetOpts
		wantOperations operations
	}{
		{
			name: "GitHub publish",
			currentSpec: testSpecOpts{
				published: true,
				repo:      githubRepo.ID,
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStateUnpublished,
				repo:             githubRepo.ID,
			},
			wantOperations: operations{operationPush, operationPublish},
		},
		{
			name: "GitHub publish as draft",
			currentSpec: testSpecOpts{
				published: "draft",
				repo:      githubRepo.ID,
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStateUnpublished,
				repo:             githubRepo.ID,
			},
			wantOperations: operations{operationPush, operationPublishDraft},
		},
		{
			name: "GitHub publish false",
			currentSpec: testSpecOpts{
				published: false,
				repo:      githubRepo.ID,
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStateUnpublished,
				repo:             githubRepo.ID,
			},
			wantOperations: operations{},
		},
		{
			name: "set to draft but unsupported",
			currentSpec: testSpecOpts{
				published: "draft",
				repo:      bbsRepo.ID,
			},
			changeset: testChangesetOpts{
				externalServiceType: extsvc.TypeBitbucketServer,
				publicationState:    campaigns.ChangesetPublicationStateUnpublished,
				repo:                bbsRepo.ID,
			},
			wantOperations: operations{},
		},
		{
			name: "set from draft to publish true",
			previousSpec: testSpecOpts{
				published: "draft",
				repo:      githubRepo.ID,
			},
			currentSpec: testSpecOpts{
				published: true,
				repo:      githubRepo.ID,
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				repo:             githubRepo.ID,
			},
			wantOperations: operations{operationUndraft},
		},
		{
			name: "set from draft to publish true on unpublished",
			previousSpec: testSpecOpts{
				published: "draft",
				repo:      githubRepo.ID,
			},
			currentSpec: testSpecOpts{
				published: true,
				repo:      githubRepo.ID,
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStateUnpublished,
				repo:             githubRepo.ID,
			},
			wantOperations: operations{operationPush, operationPublish},
		},
		{
			name: "changeset spec changed attribute, needs update",
			previousSpec: testSpecOpts{
				published: true,
				repo:      githubRepo.ID,
				title:     "Before",
			},
			currentSpec: testSpecOpts{
				published: true,
				repo:      githubRepo.ID,
				title:     "After",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				repo:             githubRepo.ID,
			},
			wantOperations: operations{operationUpdate},
		},
		{
			name: "changeset spec changed, needs new commit but no update",
			previousSpec: testSpecOpts{
				published:  true,
				repo:       githubRepo.ID,
				commitDiff: "testDiff",
			},
			currentSpec: testSpecOpts{
				published:  true,
				repo:       githubRepo.ID,
				commitDiff: "newTestDiff",
			},
			changeset: testChangesetOpts{
				publicationState: campaigns.ChangesetPublicationStatePublished,
				repo:             githubRepo.ID,
			},
			wantOperations: operations{operationPush, operationSleep, operationSync},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := store.Transact(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Done(errors.New("fail tx purposefully"))
			tc.currentSpec.campaignSpec = campaignSpec.ID
			var previousSpec *campaigns.ChangesetSpec
			if tc.previousSpec != (testSpecOpts{}) {
				previousSpec = createChangesetSpec(t, ctx, tx, tc.previousSpec)
				tc.changeset.previousSpec = previousSpec.ID
			}
			currentSpec := createChangesetSpec(t, ctx, tx, tc.currentSpec)
			tc.changeset.currentSpec = currentSpec.ID
			cs := createChangeset(t, ctx, tx, tc.changeset)
			plan, err := determinePlan(previousSpec, currentSpec, cs)
			if err != nil {
				t.Fatal(err)
			}
			if have, want := plan.ops, tc.wantOperations; !have.Equal(want) {
				t.Fatalf("incorrect plan determined, want=%v have=%v", want, have)
			}
		})
	}
}

func TestReconcilerProcess_PublishedChangesetDuplicateBranch(t *testing.T) {
	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)

	store := NewStore(dbconn.Global)

	admin := createTestUser(ctx, t)
	if !admin.SiteAdmin {
		t.Fatalf("admin is not site admin")
	}

	rs, _ := createTestRepos(t, ctx, dbconn.Global, 1)

	state := ct.MockChangesetSyncState(&protocol.RepoInfo{
		Name: api.RepoName(rs[0].Name),
		VCS:  protocol.VCSInfo{URL: rs[0].URI},
	})
	defer state.Unmock()

	commonHeadRef := "refs/heads/collision"

	// Create a published changeset.
	campaignSpec := createCampaignSpec(t, ctx, store, "reconciler-test-campaign", admin.ID)
	campaign := createCampaign(t, ctx, store, "reconciler-test-campaign", admin.ID, campaignSpec.ID)
	changesetSpec := createChangesetSpec(t, ctx, store, testSpecOpts{
		user:         admin.ID,
		repo:         rs[0].ID,
		campaignSpec: campaignSpec.ID,
		headRef:      commonHeadRef,
	})
	createChangeset(t, ctx, store, testChangesetOpts{
		repo:             rs[0].ID,
		publicationState: campaigns.ChangesetPublicationStatePublished,
		campaign:         campaign.ID,
		ownedByCampaign:  campaign.ID,
		currentSpec:      changesetSpec.ID,
		externalBranch:   git.AbbreviateRef(commonHeadRef),
		externalID:       "123",
	})

	// Try to publish a changeset on the same HeadRef/ExternalBranch.
	otherCampaignSpec := createCampaignSpec(t, ctx, store, "other-test-campaign", admin.ID)
	otherCampaign := createCampaign(t, ctx, store, "other-test-campaign", admin.ID, otherCampaignSpec.ID)
	otherChangesetSpec := createChangesetSpec(t, ctx, store, testSpecOpts{
		user:         admin.ID,
		repo:         rs[0].ID,
		campaignSpec: otherCampaignSpec.ID,
		headRef:      commonHeadRef,
		published:    true,
	})
	otherChangeset := createChangeset(t, ctx, store, testChangesetOpts{
		repo:             rs[0].ID,
		publicationState: campaigns.ChangesetPublicationStateUnpublished,
		campaign:         otherCampaign.ID,
		ownedByCampaign:  otherCampaign.ID,
		currentSpec:      otherChangesetSpec.ID,
	})

	// Run the reconciler
	rec := reconciler{
		noSleepBeforeSync: true,
		sourcer:           repos.NewFakeSourcer(nil, &ct.FakeChangesetSource{}),
		store:             store,
	}
	haveErr := rec.process(ctx, store, otherChangeset)
	if !errors.Is(haveErr, ErrPublishSameBranch) {
		t.Fatalf("reconciler process failed with wrong error: %s", haveErr)
	}
}

func buildGithubPR(now time.Time, externalState campaigns.ChangesetExternalState) *github.PullRequest {
	state := string(externalState)

	pr := &github.PullRequest{
		ID:          "12345",
		Number:      12345,
		Title:       state + " GitHub PR",
		Body:        state + " GitHub PR",
		State:       state,
		HeadRefName: git.AbbreviateRef("head-ref-on-github"),
		TimelineItems: []github.TimelineItem{
			{Type: "PullRequestCommit", Item: &github.PullRequestCommit{
				Commit: github.Commit{
					OID:           "new-f00bar",
					PushedDate:    now,
					CommittedDate: now,
				},
			}},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if externalState == campaigns.ChangesetExternalStateDraft {
		pr.State = "OPEN"
		pr.IsDraft = true
	}

	if externalState == campaigns.ChangesetExternalStateClosed {
		// We add a "ClosedEvent" so that the SyncChangesets call that happens after closing
		// the PR has the "correct" state to set the ExternalState
		pr.TimelineItems = append(pr.TimelineItems, github.TimelineItem{
			Type: "ClosedEvent",
			Item: &github.ClosedEvent{CreatedAt: now.Add(1 * time.Hour)},
		})
		pr.UpdatedAt = now.Add(1 * time.Hour)
	}

	return pr
}

type testChangesetOpts struct {
	repo         api.RepoID
	campaign     int64
	currentSpec  int64
	previousSpec int64

	externalServiceType string
	externalID          string
	externalBranch      string
	externalState       campaigns.ChangesetExternalState

	publicationState campaigns.ChangesetPublicationState

	reconcilerState campaigns.ReconcilerState
	failureMessage  string
	numFailures     int64

	ownedByCampaign int64

	unsynced bool
	closing  bool
}

func createChangeset(
	t *testing.T,
	ctx context.Context,
	store *Store,
	opts testChangesetOpts,
) *campaigns.Changeset {
	t.Helper()

	if opts.externalServiceType == "" {
		opts.externalServiceType = extsvc.TypeGitHub
	}

	changeset := &campaigns.Changeset{
		RepoID:         opts.repo,
		CurrentSpecID:  opts.currentSpec,
		PreviousSpecID: opts.previousSpec,

		ExternalServiceType: opts.externalServiceType,
		ExternalID:          opts.externalID,
		ExternalBranch:      opts.externalBranch,
		ExternalState:       opts.externalState,

		PublicationState: opts.publicationState,

		OwnedByCampaignID: opts.ownedByCampaign,

		Unsynced: opts.unsynced,
		Closing:  opts.closing,

		ReconcilerState: opts.reconcilerState,
		NumFailures:     opts.numFailures,
	}

	if opts.failureMessage != "" {
		changeset.FailureMessage = &opts.failureMessage
	}

	if opts.campaign != 0 {
		changeset.CampaignIDs = []int64{opts.campaign}
	}

	if err := store.CreateChangeset(ctx, changeset); err != nil {
		t.Fatalf("creating changeset failed: %s", err)
	}

	return changeset
}

func TestDecorateChangesetBody(t *testing.T) {
	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)

	now := time.Now().UTC().Truncate(time.Microsecond)
	clock := func() time.Time {
		return now.UTC().Truncate(time.Microsecond)
	}
	store := NewStoreWithClock(dbconn.Global, clock)

	admin := createTestUser(ctx, t)
	if !admin.SiteAdmin {
		t.Fatal("admin is not site admin")
	}

	rs, _ := createTestRepos(t, ctx, dbconn.Global, 1)

	state := ct.MockChangesetSyncState(&protocol.RepoInfo{
		Name: api.RepoName(rs[0].Name),
		VCS:  protocol.VCSInfo{URL: rs[0].URI},
	})
	defer state.Unmock()

	internalClient = &mockInternalClient{externalURL: "https://sourcegraph.test"}
	defer func() { internalClient = api.InternalClient }()

	// Create a changeset.
	campaignSpec := createCampaignSpec(t, ctx, store, "reconciler-test-campaign", admin.ID)
	campaign := createCampaign(t, ctx, store, "reconciler-test-campaign", admin.ID, campaignSpec.ID)
	cs := createChangeset(t, ctx, store, testChangesetOpts{
		repo:            rs[0].ID,
		ownedByCampaign: campaign.ID,
	})

	body := "body"
	rcs := &repos.Changeset{Body: body, Changeset: cs, Repo: rs[0]}
	if err := decorateChangesetBody(ctx, store, rcs); err != nil {
		t.Errorf("unexpected non-nil error: %v", err)
	}
	if want := body + "\n\n[_Created by Sourcegraph campaign `" + admin.Username + "/reconciler-test-campaign`._](https://sourcegraph.test/users/" + admin.Username + "/campaigns/reconciler-test-campaign)"; rcs.Body != want {
		t.Errorf("repos.Changeset body unexpectedly changed: have=%q want=%q", rcs.Body, want)
	}
}

func TestCampaignURL(t *testing.T) {
	ctx := context.Background()

	t.Run("errors", func(t *testing.T) {
		for name, tc := range map[string]*mockInternalClient{
			"ExternalURL error": {err: errors.New("foo")},
			"invalid URL":       {externalURL: "foo://:bar"},
		} {
			t.Run(name, func(t *testing.T) {
				internalClient = tc
				defer func() { internalClient = api.InternalClient }()

				if _, err := campaignURL(ctx, nil, nil); err == nil {
					t.Error("unexpected nil error")
				}
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		internalClient = &mockInternalClient{externalURL: "https://sourcegraph.test"}
		defer func() { internalClient = api.InternalClient }()

		url, err := campaignURL(
			ctx,
			&db.Namespace{Name: "foo", Organization: 123},
			&campaigns.Campaign{Name: "bar"},
		)
		if err != nil {
			t.Errorf("unexpected non-nil error: %v", err)
		}
		if want := "https://sourcegraph.test/organizations/foo/campaigns/bar"; url != want {
			t.Errorf("unexpected URL: have=%q want=%q", url, want)
		}
	})
}

func TestNamespaceURL(t *testing.T) {
	for name, tc := range map[string]struct {
		ns   *db.Namespace
		want string
	}{
		"user": {
			ns:   &db.Namespace{User: 123, Name: "user"},
			want: "/users/user",
		},
		"org": {
			ns:   &db.Namespace{Organization: 123, Name: "org"},
			want: "/organizations/org",
		},
		"neither": {
			ns:   &db.Namespace{Name: "user"},
			want: "/users/user",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if have := namespaceURL(tc.ns); have != tc.want {
				t.Errorf("unexpected URL: have=%q want=%q", have, tc.want)
			}
		})
	}
}

type mockInternalClient struct {
	externalURL string
	err         error
}

func (c *mockInternalClient) ExternalURL(ctx context.Context) (string, error) {
	return c.externalURL, c.err
}
