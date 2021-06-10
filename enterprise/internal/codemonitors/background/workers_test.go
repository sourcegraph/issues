package background

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/graph-gophers/graphql-go/relay"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codemonitors"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codemonitors/email"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codemonitors/resolvers"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codemonitors/storetest"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
)

func TestActionRunner(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	externalURL := "https://www.sourcegraph.com"
	testQuery := "test patternType:literal"

	// Mocks.
	got := email.TemplateDataNewSearchResults{}
	email.MockSendEmailForNewSearchResult = func(ctx context.Context, userID int32, data *email.TemplateDataNewSearchResults) error {
		got = *data
		return nil
	}
	email.MockExternalURL = func() *url.URL {
		externalURL, _ := url.Parse("https://www.sourcegraph.com")
		return externalURL
	}

	tests := []struct {
		name               string
		numResults         int
		wantNumResultsText string
	}{
		{
			name:               "5 results",
			numResults:         5,
			wantNumResultsText: "There were 5 new search results for your query",
		},
		{
			name:               "1 result",
			numResults:         1,
			wantNumResultsText: "There was 1 new search result for your query",
		},
	}

	want := email.TemplateDataNewSearchResults{
		Priority:       "New",
		SearchURL:      externalURL + "/search?q=test+patternType%3Aliteral&utm_source=code-monitoring-email",
		Description:    "test description",
		CodeMonitorURL: externalURL + "/code-monitoring/" + string(relay.MarshalID(resolvers.MonitorKind, 1)) + "?utm_source=code-monitoring-email",
	}

	var (
		queryID      int64 = 1
		triggerEvent       = 1
		record       *codemonitors.ActionJob
	)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a TestStore.
			var err error
			db := dbtest.NewDB(t, "")
			now := time.Now()
			clock := func() time.Time { return now }
			s := codemonitors.NewStoreWithClock(db, clock)
			ctx, ts := storetest.NewTestStoreWithStore(t, s)

			_, _, _, userCtx := storetest.NewTestUser(ctx, t, db)

			// Run a complete pipeline from creation of a code monitor to sending of an email.
			_, err = ts.InsertTestMonitor(userCtx, t)
			if err != nil {
				t.Fatal(err)
			}
			err = ts.EnqueueTriggerQueries(ctx)
			if err != nil {
				t.Fatal(err)
			}
			err = ts.LogSearch(ctx, testQuery, tt.numResults, triggerEvent)
			if err != nil {
				t.Fatal(err)
			}
			err = ts.EnqueueActionEmailsForQueryIDInt64(ctx, queryID, triggerEvent)
			if err != nil {
				t.Fatal(err)
			}
			record, err = ts.ActionJobForIDInt(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}

			a := actionRunner{s}
			err = a.Handle(ctx, createDBWorkerStoreForActionJobs(s), record)
			if err != nil {
				t.Fatal(err)
			}

			want.NumberOfResultsWithDetail = tt.wantNumResultsText
			if diff := cmp.Diff(got, want); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
		})
	}
}
