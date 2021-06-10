package resolvers

import (
	"context"
	"testing"
	"time"

	"github.com/hexops/autogold"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	insightsdbtesting "github.com/sourcegraph/sourcegraph/enterprise/internal/insights/dbtesting"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/discovery"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
)

// Note: You can `go test ./resolvers -update` to update the expected `want` values in these tests.
// See https://github.com/hexops/autogold for more information.

var testRealGlobalSettings = &api.Settings{ID: 1, Contents: `{
	"insights": [
		{
		  "title": "fmt usage",
		  "description": "fmt.Errorf/fmt.Printf usage",
		  "series": [
			{
			  "label": "fmt.Errorf",
			  "search": "errorf",
			},
			{
			  "label": "printf",
			  "search": "fmt.Printf",
			}
		  ]
		},
		{
			"title": "gitserver usage",
			"description": "gitserver exec & close usage",
			"series": [
			  {
				"label": "exec",
				"search": "gitserver.Exec",
			  },
			  {
				"label": "close",
				"search": "gitserver.Close",
			  }
			]
		  }
		]
	}
`}

// TestResolver_InsightConnection tests that the InsightConnection GraphQL resolver works.
func TestResolver_InsightConnection(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()

	testSetup := func(t *testing.T) (context.Context, graphqlbackend.InsightConnectionResolver) {
		// Setup the GraphQL resolver.
		ctx := backend.WithAuthzBypass(context.Background())
		now := time.Now().UTC().Truncate(time.Microsecond)
		clock := func() time.Time { return now }
		timescale, cleanup := insightsdbtesting.TimescaleDB(t)
		defer cleanup()
		postgres := dbtest.NewDB(t, "")
		resolver := newWithClock(timescale, postgres, clock)

		// Create the insights connection resolver.
		conn, err := resolver.Insights(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Mock the setting store to return the desired settings.
		settingStore := discovery.NewMockSettingStore()
		conn.(*insightConnectionResolver).settingStore = settingStore
		settingStore.GetLatestFunc.SetDefaultReturn(testRealGlobalSettings, nil)
		return ctx, conn
	}

	t.Run("TotalCount", func(t *testing.T) {
		ctx, conn := testSetup(t)
		totalCount, err := conn.TotalCount(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if totalCount != 2 {
			t.Fatal("incorrect length")
		}
	})

	t.Run("PageInfo", func(t *testing.T) {
		// TODO: future: our pagination support is non-existent. Currently we just return all
		// insights, regardless of how many you ask for.
		ctx, conn := testSetup(t)
		gotPageInfo, err := conn.PageInfo(ctx)
		if err != nil {
			t.Fatal(err)
		}
		autogold.Want("PageInfo", &graphqlutil.PageInfo{}).Equal(t, gotPageInfo)
	})

	t.Run("Nodes", func(t *testing.T) {
		ctx, conn := testSetup(t)
		nodes, err := conn.Nodes(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(nodes) != 2 {
			t.Fatal("incorrect length")
		}
		autogold.Want("first insight", map[string]interface{}{"description": "fmt.Errorf/fmt.Printf usage", "title": "fmt usage"}).Equal(t, map[string]interface{}{
			"title":       nodes[0].Title(),
			"description": nodes[0].Description(),
		})
		// TODO(slimsag): put series length into map (autogold bug, omits the field for some reason?)
		autogold.Want("first insight: series length", int(2)).Equal(t, len(nodes[0].Series()))

		autogold.Want("second insight", map[string]interface{}{"description": "gitserver exec & close usage", "title": "gitserver usage"}).Equal(t, map[string]interface{}{
			"title":       nodes[1].Title(),
			"description": nodes[1].Description(),
		})
		autogold.Want("second insight: series length", int(2)).Equal(t, len(nodes[1].Series()))
	})
}
