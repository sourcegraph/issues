package insights

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/inconshreveable/log15"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
)

func init() {
	dbtesting.DBNameSuffix = "insights"
}

func TestGetSearchInsights(t *testing.T) {
	ctx := context.Background()

	db := dbtesting.GetDB(t)
	_, err := db.Exec(`INSERT INTO orgs(id, name) VALUES (1, 'first-org'), (2, 'second-org');`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`

			INSERT INTO settings (id, org_id, contents, created_at, user_id, author_user_id)
			VALUES  (1, 1, $1, CURRENT_TIMESTAMP, NULL, NULL)`, insightSettingSimple)
	if err != nil {
		t.Fatal(err)
	}

	got, err := GetSearchInsights(ctx, db, All)
	if err != nil {
		t.Fatal(err)
	}

	weeks := 2

	want := []SearchInsight{{
		ID:           "searchInsights.insight.global.simple",
		Title:        "my insight",
		Repositories: []string{"github.com/sourcegraph/sourcegraph"},
		Series: []TimeSeries{{
			Name:   "Redis",
			Stroke: "var(--oc-red-7)",
			Query:  "redis",
		}},
		Step: Interval{
			Weeks: &weeks,
		},
	}}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatched search insight want/got: %v", diff)
	}
}

func TestGetIntegrationInsights(t *testing.T) {
	ctx := context.Background()

	db := dbtesting.GetDB(t)
	_, err := db.Exec(`INSERT INTO orgs(id, name) VALUES (1, 'first-org'), (2, 'second-org');`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`

			INSERT INTO settings (id, org_id, contents, created_at, user_id, author_user_id)
			VALUES  (1, 1, $1, CURRENT_TIMESTAMP, NULL, NULL)`, integratedInsightSimple)
	if err != nil {
		t.Fatal(err)
	}

	got, err := GetIntegratedInsights(ctx, db)
	if err != nil {
		t.Fatal(err)
	}

	// sorting for deterministic test
	sort.Slice(got, func(i, j int) bool {
		return strings.Compare(got[i].ID, got[j].ID) < 0
	})

	weeks := 2

	want := []SearchInsight{
		{
			ID:           "unique-id1",
			Title:        "my insight",
			Repositories: []string{"github.com/sourcegraph/sourcegraph"},
			Series: []TimeSeries{{
				Name:   "Redis",
				Stroke: "var(--oc-red-7)",
				Query:  "redis",
			}},
			Step: Interval{
				Weeks: &weeks,
			},
		},
		{
			ID:           "unique-id2",
			Title:        "my insight2",
			Repositories: []string{"github.com/sourcegraph/sourcegraph"},
			Series: []TimeSeries{{
				Name:   "Redis",
				Stroke: "var(--oc-red-7)",
				Query:  "redis2",
			}},
			Step: Interval{
				Weeks: &weeks,
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected integration insights want/got: %v", diff)
	}

	log15.Info("msg", "got", got)
}

const integratedInsightSimple = `{
	"insights.allrepos":{
		"unique-id1": {
			"title": "my insight",
			"repositories": ["github.com/sourcegraph/sourcegraph"],
			"series": [
			  {
				"name": "Redis",
				"query": "redis",
				"stroke": "var(--oc-red-7)"
			  }
			],
			"step": {
			  "weeks": 2
			}
		},
		"unique-id2": {
			"title": "my insight2",
			"repositories": ["github.com/sourcegraph/sourcegraph"],
			"series": [
			  {
				"name": "Redis",
				"query": "redis2",
				"stroke": "var(--oc-red-7)"
			  }
			],
			"step": {
			  "weeks": 2
			}
		},
		"invalid-schema": {
			"title": {}
		}
  	},
	"random-setting": {}
}`

const insightSettingSimple = `{"searchInsights.insight.global.simple": {
    "title": "my insight",
    "repositories": ["github.com/sourcegraph/sourcegraph"],
    "series": [
      {
        "name": "Redis",
        "query": "redis",
        "stroke": "var(--oc-red-7)"
      }
    ],
    "step": {
      "weeks": 2
    }
  }}`
