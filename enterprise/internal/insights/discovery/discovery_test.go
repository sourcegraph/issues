package discovery

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/insights"

	"github.com/hexops/autogold"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/schema"
)

var settingsExample = &api.Settings{ID: 1, Contents: `{
	"insights": [
		{
		  "title": "fmt usage",
		  "description": "errors.Errorf/fmt.Printf usage",
		  "id": "1",
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
			"id": "5",
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

func TestDiscover(t *testing.T) {
	settingStore := NewMockSettingStore()
	settingStore.GetLatestFunc.SetDefaultHook(func(ctx context.Context, subject api.SettingsSubject) (*api.Settings, error) {
		if !subject.Site { // TODO: future: site is an extremely poor name for "global settings", we should change this.
			t.Fatal("expected only to request settings from global user settings")
		}
		return settingsExample, nil
	})
	ctx := context.Background()

	loader := insights.NewMockLoader()

	t.Run("test_with_no_id_filter", func(t *testing.T) {
		discovered, err := Discover(ctx, settingStore, loader, InsightFilterArgs{})
		if err != nil {
			t.Fatal(err)
		}
		autogold.Want("discovered", []insights.SearchInsight{
			{
				ID:          "1",
				Title:       "fmt usage",
				Description: "errors.Errorf/fmt.Printf usage",
				Series: []insights.TimeSeries{
					{
						Name:  "errors.Errorf",
						Query: "errorf",
					},
					{
						Name:  "printf",
						Query: "fmt.Printf",
					},
				},
			},
			{
				ID:          "5",
				Title:       "gitserver usage",
				Description: "gitserver exec & close usage",
				Series: []insights.TimeSeries{
					{
						Name:  "exec",
						Query: "gitserver.Exec",
					},
					{
						Name:  "close",
						Query: "gitserver.Close",
					},
				},
			},
		}).Equal(t, discovered)
	})

	t.Run("test_with_id_filter", func(t *testing.T) {
		discovered, err := Discover(ctx, settingStore, loader, InsightFilterArgs{Ids: []string{"1"}})
		if err != nil {
			t.Fatal(err)
		}
		autogold.Want("discovered_id_filter", []insights.SearchInsight{
			{
				ID:          "1",
				Title:       "fmt usage",
				Description: "errors.Errorf/fmt.Printf usage",
				Series: []insights.TimeSeries{
					{
						Name:  "errors.Errorf",
						Query: "errorf",
					},
					{
						Name:  "printf",
						Query: "fmt.Printf",
					},
				},
			},
		}).Equal(t, discovered)
	})

	t.Run("test_with_loader", func(t *testing.T) {
		integrated := []insights.SearchInsight{{
			ID:          "1234",
			Title:       "my insight",
			Description: "woooo!!!!",
		}}

		loader.LoadAllFunc.SetDefaultReturn(integrated, nil)
		discovered, err := Discover(ctx, settingStore, loader, InsightFilterArgs{Ids: []string{"1"}})
		if err != nil {
			t.Fatal(err)
		}

		autogold.Want("discovered_with_loader", []insights.SearchInsight{{
			ID:          "1",
			Title:       "fmt usage",
			Description: "errors.Errorf/fmt.Printf usage",
			Series: []insights.TimeSeries{
				{
					Name:  "errors.Errorf",
					Query: "errorf",
				},
				{
					Name:  "printf",
					Query: "fmt.Printf",
				},
			},
		}}).Equal(t, discovered)
	})
}

func Test_parseUserSettings(t *testing.T) {
	tests := []struct {
		name  string
		input *api.Settings
		want  autogold.Value
	}{
		{
			name:  "nil",
			input: nil,
			want:  autogold.Want("nil", [2]interface{}{&schema.Settings{}, nil}),
		},
		{
			name: "empty",
			input: &api.Settings{
				Contents: "{}",
			},
			want: autogold.Want("empty", [2]interface{}{&schema.Settings{}, nil}),
		},
		{
			name:  "real",
			input: settingsExample,
			want: autogold.Want("real", [2]interface{}{
				&schema.Settings{Insights: []*schema.Insight{
					{
						Description: "errors.Errorf/fmt.Printf usage",
						Id:          "1",
						Series: []*schema.InsightSeries{
							{
								Label:  "errors.Errorf",
								Search: "errorf",
							},
							{
								Label:  "printf",
								Search: "fmt.Printf",
							},
						},
						Title: "fmt usage",
					},
					{
						Description: "gitserver exec & close usage",
						Id:          "5",
						Series: []*schema.InsightSeries{
							{
								Label:  "exec",
								Search: "gitserver.Exec",
							},
							{
								Label:  "close",
								Search: "gitserver.Close",
							},
						},
						Title: "gitserver usage",
					},
				}},
				nil,
			}),
		},
	}
	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			got, err := parseUserSettings(tst.input)
			tst.want.Equal(t, [2]interface{}{got, err})
		})
	}

}
