package resolvers

import (
	"context"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/authz"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/actor"

	"github.com/hexops/autogold"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	insightsdbtesting "github.com/sourcegraph/sourcegraph/enterprise/internal/insights/dbtesting"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/discovery"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
)

// Note: You can `go test ./resolvers -update` to update the expected `want` values in these tests.
// See https://github.com/hexops/autogold for more information.

var testRealGlobalSettings = &api.Settings{ID: 1, Contents: `{
	"insights": [
		{
		  "title": "fmt usage",
		  "description": "errors.Errorf/fmt.Printf usage",
		  "series": [
			{
			  "label": "errors.Errorf",
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

var testOneGlobalSeries = &api.Settings{ID: 1, Contents: `{
	"insights": [
		{
		  "title": "fmt usage",
		  "description": "errors.Errorf",
		  "series": [
			{
			  "label": "errors.Errorf",
			  "search": "errorf",
			}
		  ]
		},
    ]
  }
`}

// TestResolver_InsightConnection tests that the InsightConnection GraphQL resolver works.
func TestResolver_InsightConnection(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	//t.Parallel() // TODO: dbtesting.GetDB is not parallel-safe, yuck.

	testSetup := func(t *testing.T) (context.Context, graphqlbackend.InsightConnectionResolver) {
		// Setup the GraphQL resolver.
		ctx := backend.WithAuthzBypass(context.Background())
		now := time.Now().UTC().Truncate(time.Microsecond)
		clock := func() time.Time { return now }
		timescale, cleanup := insightsdbtesting.TimescaleDB(t)
		defer cleanup()
		postgres := dbtesting.GetDB(t)
		resolver := newWithClock(timescale, postgres, clock)

		// Create the insights connection resolver.
		conn, err := resolver.Insights(ctx, nil)
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
		autogold.Want("first insight", map[string]interface{}{"description": "errors.Errorf/fmt.Printf usage", "title": "fmt usage"}).Equal(t, map[string]interface{}{
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

func TestResolver_InsightsRepoPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	//t.Parallel() // TODO: dbtesting.GetDB is not parallel-safe, yuck.
	timescale, cleanup := insightsdbtesting.TimescaleDB(t)
	defer cleanup()
	postgres := dbtesting.GetDB(t)

	now := time.Now().UTC().Truncate(time.Microsecond)
	clock := func() time.Time { return now }
	pointTime := now.Add(-time.Hour * 24)
	authz.SetProviders(false, []authz.Provider{}) // setting authz in this way will force user permissions to be enabled

	// Create three repositories -
	// 1) private repo that will be assigned to user 1
	// 2 & 3) public repos
	_, err := postgres.Exec(`
		INSERT INTO repo (id, name, description, fork, created_at, updated_at, external_id, external_service_type,
					  external_service_id, archived, uri, deleted_at, metadata, private, cloned, stars)
		VALUES
			(1, 'test-repo1', 'description', false, current_timestamp, current_timestamp, 1, 'github', 1, false, 'github.com/test-repo/test-repo1', null, '{}', true, true, 1),
			(2, 'test-repo2', 'description', false, current_timestamp, current_timestamp, 2, 'github', 1, false, 'github.com/test-repo/test-repo2', null, '{}', false, true, 1),
			(3, 'test-repo3', 'description', false, current_timestamp, current_timestamp, 3, 'github', 1, false, 'github.com/test-repo/test-repo3', null, '{}', false, true, 1);

		INSERT INTO user_permissions (user_id, permission, object_type, object_ids, updated_at, synced_at, object_ids_ints)
		VALUES
		       (1, 'read', 'repos', '', current_timestamp, current_timestamp, ARRAY[1]);

		INSERT INTO user_permissions (user_id, permission, object_type, object_ids, updated_at, synced_at)
		VALUES
		       (2, 'read', 'repos', '', current_timestamp, current_timestamp),
		       (3, 'read', 'repos', '', current_timestamp, current_timestamp);

		INSERT INTO users (id, username, display_name, avatar_url, created_at, updated_at, deleted_at, invite_quota, passwd,
						   site_admin)
		VALUES
		(1, 'user1', 'user1', null, current_timestamp, current_timestamp, null, 1, 'abc', false),
		(2, 'user2', 'user2', null, current_timestamp, current_timestamp, null, 1, 'abc', false);`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = timescale.Exec(`INSERT INTO repo_names (id, name) VALUES (1, 'ignore-me');`)
	if err != nil {
		t.Fatal(err)
	}

	// Create some timeseries data, one row in each repository
	_, err = timescale.Exec(`
		INSERT INTO series_points (series_id, "time", "value", metadata_id, repo_id, repo_name_id, original_repo_name_id)
		VALUES
			('s:087855E6A24440837303FD8A252E9893E8ABDFECA55B61AC83DA1B521906626E', $1, 5.0, null, 1, 1, 1),
			('s:087855E6A24440837303FD8A252E9893E8ABDFECA55B61AC83DA1B521906626E', $1, 6.0, null, 2, 1, 1),
			('s:087855E6A24440837303FD8A252E9893E8ABDFECA55B61AC83DA1B521906626E', $1, 7.0, null, 3, 1, 1)`, pointTime)
	if err != nil {
		t.Fatal(err)
	}

	setUpTest := func(ctx context.Context, t *testing.T) graphqlbackend.InsightConnectionResolver {

		resolver := newWithClock(timescale, postgres, clock)
		conn, err := resolver.Insights(ctx, &graphqlbackend.InsightsArgs{})
		if err != nil {
			t.Fatal(err)
		}
		settingStore := discovery.NewMockSettingStore()
		conn.(*insightConnectionResolver).settingStore = settingStore
		settingStore.GetLatestFunc.SetDefaultReturn(testOneGlobalSeries, nil)

		return conn
	}

	t.Run("user with private repo", func(t *testing.T) {
		ctx := context.Background()
		a := actor.FromUser(1)
		ctx = actor.WithActor(ctx, a)
		conn := setUpTest(ctx, t)

		want := []point{
			{
				value: 18.0,
				time:  time.Now().Truncate(time.Hour * 24),
			},
		}

		got := resolvePoints(ctx, conn, t)
		if diff := cmp.Diff(want, got, cmp.Comparer(func(x, y point) bool {
			return x.value == y.value && x.time.Equal(y.time)
		})); diff != "" {
			t.Errorf("expected results for user with private repo diff: %v", diff)
		}
	})

	t.Run("user with only public repos", func(t *testing.T) {
		ctx := context.Background()
		a := actor.FromUser(2)
		ctx = actor.WithActor(ctx, a)
		conn := setUpTest(ctx, t)

		want := []point{
			{
				value: 13.0,
				time:  time.Now().Truncate(time.Hour * 24),
			},
		}

		got := resolvePoints(ctx, conn, t)
		if diff := cmp.Diff(want, got, cmp.Comparer(func(x, y point) bool {
			return x.value == y.value && x.time.Equal(y.time)
		})); diff != "" {
			t.Errorf("expected results for user with only public repos diff: %v", diff)
		}
	})

}

type point struct {
	value float64
	time  time.Time
}

func resolvePoints(ctx context.Context, conn graphqlbackend.InsightConnectionResolver, t *testing.T) []point {
	nodes, err := conn.Nodes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Errorf("unexpected length of nodes: want: %v got: %v", 1, len(nodes))
	}

	expected := nodes[0]
	seriesResolvers := expected.Series()
	if len(seriesResolvers) != 1 {
		t.Errorf("unexpected length of series resolvers: want: %v got: %v", 1, len(seriesResolvers))
	}
	sr := seriesResolvers[0]
	data, err := sr.Points(ctx, &graphqlbackend.InsightsPointsArgs{})
	if err != nil {
		t.Fatal(err)
	}

	var results []point
	for _, d := range data {
		var temp point
		temp.value = d.Value()
		temp.time = d.DateTime().Time
		results = append(results, temp)
	}
	return results
}
