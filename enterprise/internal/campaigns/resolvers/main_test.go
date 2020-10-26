package resolvers

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	ee "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/resolvers/apitest"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

func init() {
	dbtesting.DBNameSuffix = "campaignsresolversdb"
}

var update = flag.Bool("update", false, "update testdata")

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Verbose() {
		log15.Root().SetHandler(log15.DiscardHandler())
	}
	os.Exit(m.Run())
}

const testDiff = `diff README.md README.md
index 671e50a..851b23a 100644
--- README.md
+++ README.md
@@ -1,2 +1,2 @@
 # README
-This file is hosted at example.com and is a test file.
+This file is hosted at sourcegraph.com and is a test file.
diff --git urls.txt urls.txt
index 6f8b5d9..17400bc 100644
--- urls.txt
+++ urls.txt
@@ -1,3 +1,3 @@
 another-url.com
-example.com
+sourcegraph.com
 never-touch-the-mouse.com
`

// testDiffGraphQL is the parsed representation of testDiff.
var testDiffGraphQL = apitest.FileDiffs{
	TotalCount: 2,
	RawDiff:    testDiff,
	DiffStat:   apitest.DiffStat{Changed: 2},
	PageInfo:   apitest.PageInfo{},
	Nodes: []apitest.FileDiff{
		{
			OldPath: "README.md",
			NewPath: "README.md",
			OldFile: apitest.File{Name: "README.md"},
			Hunks: []apitest.FileDiffHunk{
				{
					Body:     " # README\n-This file is hosted at example.com and is a test file.\n+This file is hosted at sourcegraph.com and is a test file.\n",
					OldRange: apitest.DiffRange{StartLine: 1, Lines: 2},
					NewRange: apitest.DiffRange{StartLine: 1, Lines: 2},
				},
			},
			Stat: apitest.DiffStat{Changed: 1},
		},
		{
			OldPath: "urls.txt",
			NewPath: "urls.txt",
			OldFile: apitest.File{Name: "urls.txt"},
			Hunks: []apitest.FileDiffHunk{
				{
					Body:     " another-url.com\n-example.com\n+sourcegraph.com\n never-touch-the-mouse.com\n",
					OldRange: apitest.DiffRange{StartLine: 1, Lines: 3},
					NewRange: apitest.DiffRange{StartLine: 1, Lines: 3},
				},
			},
			Stat: apitest.DiffStat{Changed: 1},
		},
	},
}

func marshalJSON(t testing.TB, v interface{}) string {
	t.Helper()

	bs, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	return string(bs)
}

func marshalDateTime(t testing.TB, ts time.Time) string {
	t.Helper()

	dt := graphqlbackend.DateTime{Time: ts}

	bs, err := dt.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// Unquote the date time.
	return strings.ReplaceAll(string(bs), "\"", "")
}

func parseJSONTime(t testing.TB, ts string) time.Time {
	t.Helper()

	timestamp, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t.Fatal(err)
	}

	return timestamp
}

func insertTestUser(t *testing.T, db *sql.DB, name string, isAdmin bool) (userID int32) {
	t.Helper()

	q := sqlf.Sprintf("INSERT INTO users (username, site_admin) VALUES (%s, %t) RETURNING id", name, isAdmin)

	err := db.QueryRow(q.Query(sqlf.PostgresBindVar), q.Args()...).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}

	return userID
}

func newGitHubExternalService(t *testing.T, store repos.Store) *repos.ExternalService {
	t.Helper()

	clock := repos.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	svc := repos.ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "Github - Test",
		Config:      `{"url": "https://github.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// create a few external services
	if err := store.UpsertExternalServices(context.Background(), &svc); err != nil {
		t.Fatalf("failed to insert external services: %v", err)
	}

	return &svc
}

func newGitHubTestRepo(name string, externalService *repos.ExternalService) *repos.Repo {
	return &repos.Repo{
		Name: name,
		ExternalRepo: api.ExternalRepoSpec{
			ID:          fmt.Sprintf("external-id-%d", externalService.ID),
			ServiceType: "github",
			ServiceID:   "https://github.com/",
		},
		Sources: map[string]*repos.SourceInfo{
			externalService.URN(): {
				ID:       externalService.URN(),
				CloneURL: fmt.Sprintf("https://secrettoken@%s", name),
			},
		},
	}
}

func mockBackendCommits(t *testing.T, revs ...api.CommitID) {
	t.Helper()

	byRev := map[api.CommitID]struct{}{}
	for _, r := range revs {
		byRev[r] = struct{}{}
	}

	backend.Mocks.Repos.ResolveRev = func(_ context.Context, _ *types.Repo, rev string) (api.CommitID, error) {
		if _, ok := byRev[api.CommitID(rev)]; !ok {
			t.Fatalf("ResolveRev received unexpected rev: %q", rev)
		}
		return api.CommitID(rev), nil
	}
	t.Cleanup(func() { backend.Mocks.Repos.ResolveRev = nil })

	backend.Mocks.Repos.GetCommit = func(_ context.Context, _ *types.Repo, id api.CommitID) (*git.Commit, error) {
		if _, ok := byRev[id]; !ok {
			t.Fatalf("GetCommit received unexpected ID: %s", id)
		}
		return &git.Commit{ID: id}, nil
	}
	t.Cleanup(func() { backend.Mocks.Repos.GetCommit = nil })
}

func mockRepoComparison(t *testing.T, baseRev, headRev, diff string) {
	t.Helper()

	spec := fmt.Sprintf("%s...%s", baseRev, headRev)

	git.Mocks.GetCommit = func(id api.CommitID) (*git.Commit, error) {
		if string(id) != baseRev && string(id) != headRev {
			t.Fatalf("git.Mocks.GetCommit received unknown commit id: %s", id)
		}
		return &git.Commit{ID: id}, nil
	}
	t.Cleanup(func() { git.Mocks.GetCommit = nil })

	git.Mocks.ExecReader = func(args []string) (io.ReadCloser, error) {
		if len(args) < 1 && args[0] != "diff" {
			t.Fatalf("gitserver.ExecReader received wrong args: %v", args)
		}

		if have, want := args[len(args)-2], spec; have != want {
			t.Fatalf("gitserver.ExecReader received wrong spec: %q, want %q", have, want)
		}
		return ioutil.NopCloser(strings.NewReader(diff)), nil
	}
	t.Cleanup(func() { git.Mocks.ExecReader = nil })

	git.Mocks.MergeBase = func(repo gitserver.Repo, a, b api.CommitID) (api.CommitID, error) {
		if string(a) != baseRev && string(b) != headRev {
			t.Fatalf("git.Mocks.MergeBase received unknown commit ids: %s %s", a, b)
		}
		return a, nil
	}
	t.Cleanup(func() { git.Mocks.MergeBase = nil })
}

func addChangeset(t *testing.T, ctx context.Context, s *ee.Store, c *campaigns.Campaign, changeset int64) {
	t.Helper()

	c.ChangesetIDs = append(c.ChangesetIDs, changeset)
	if err := s.UpdateCampaign(ctx, c); err != nil {
		t.Fatal(err)
	}
}

// This is duplicated from campaigns/service_test.go, we need to find a place
// to put these helpers.
type testChangesetOpts struct {
	repo         api.RepoID
	campaign     int64
	currentSpec  int64
	previousSpec int64

	externalServiceType string
	externalID          string
	externalBranch      string
	externalState       campaigns.ChangesetExternalState
	externalReviewState campaigns.ChangesetReviewState
	externalCheckState  campaigns.ChangesetCheckState

	publicationState campaigns.ChangesetPublicationState
	reconcilerState  campaigns.ReconcilerState
	failureMessage   string
	unsynced         bool

	ownedByCampaign int64

	metadata interface{}
}

func createChangeset(
	t *testing.T,
	ctx context.Context,
	store *ee.Store,
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
		ExternalReviewState: opts.externalReviewState,
		ExternalCheckState:  opts.externalCheckState,

		PublicationState: opts.publicationState,
		ReconcilerState:  opts.reconcilerState,
		Unsynced:         opts.unsynced,

		OwnedByCampaignID: opts.ownedByCampaign,

		Metadata: opts.metadata,
	}

	if opts.failureMessage != "" {
		changeset.FailureMessage = &opts.failureMessage
	}

	if string(opts.externalState) != "" {
		changeset.ExternalState = opts.externalState
	}

	if opts.campaign != 0 {
		changeset.CampaignIDs = []int64{opts.campaign}
	}

	if err := store.CreateChangeset(ctx, changeset); err != nil {
		t.Fatalf("creating changeset failed: %s", err)
	}

	if err := store.UpsertChangesetEvents(ctx, changeset.Events()...); err != nil {
		t.Fatalf("creating changeset events failed: %s", err)
	}

	return changeset
}

type testSpecOpts struct {
	user         int32
	repo         api.RepoID
	campaignSpec int64

	// If this is non-blank, the changesetSpec will be an import/track spec for
	// the changeset with the given externalID in the given repo.
	externalID string

	// If this is set, the changesetSpec will be a "create commit on this
	// branch" changeset spec.
	headRef string

	// If this is set along with headRef, the changesetSpec will have published
	// set.
	published interface{}

	title         string
	body          string
	commitMessage string
	commitDiff    string

	baseRev string
	baseRef string
}

func createChangesetSpec(
	t *testing.T,
	ctx context.Context,
	store *ee.Store,
	opts testSpecOpts,
) *campaigns.ChangesetSpec {
	t.Helper()

	published := campaigns.PublishedValue{Val: opts.published}
	if opts.published == nil {
		// Set false as the default.
		published.Val = false
	}
	if !published.Valid() {
		t.Fatalf("invalid value for published passed, got %v (%T)", opts.published, opts.published)
	}

	spec := &campaigns.ChangesetSpec{
		UserID:         opts.user,
		RepoID:         opts.repo,
		CampaignSpecID: opts.campaignSpec,
		Spec: &campaigns.ChangesetSpecDescription{
			BaseRepository: graphqlbackend.MarshalRepositoryID(opts.repo),

			BaseRev: opts.baseRev,
			BaseRef: opts.baseRef,

			ExternalID: opts.externalID,
			HeadRef:    opts.headRef,
			Published:  published,

			Title: opts.title,
			Body:  opts.body,

			Commits: []campaigns.GitCommitDescription{
				{
					Message: opts.commitMessage,
					Diff:    opts.commitDiff,
				},
			},
		},
	}

	if err := store.CreateChangesetSpec(ctx, spec); err != nil {
		t.Fatal(err)
	}

	return spec
}
