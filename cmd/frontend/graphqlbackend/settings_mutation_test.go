package graphqlbackend

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go/gqltesting"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

func TestSettingsMutation_EditSettings(t *testing.T) {
	resetMocks()
	db.Mocks.Users.GetByID = func(context.Context, int32) (*types.User, error) {
		return &types.User{ID: 1}, nil
	}
	db.Mocks.Settings.GetLatest = func(context.Context, api.SettingsSubject) (*api.Settings, error) {
		return &api.Settings{ID: 1, Contents: "{}"}, nil
	}
	db.Mocks.Settings.CreateIfUpToDate = func(ctx context.Context, subject api.SettingsSubject, lastID, authorUserID *int32, contents string) (*api.Settings, error) {
		if want := `{
  "p": {
    "x": 123
  }
}`; contents != want {
			t.Errorf("got %q, want %q", contents, want)
		}
		return &api.Settings{ID: 2, Contents: contents}, nil
	}

	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Context: actor.WithActor(context.Background(), &actor.Actor{UID: 1}),
			Schema:  mustParseGraphQLSchema(t),
			Query: `
				mutation($value: JSONValue) {
					settingsMutation(input: {subject: "VXNlcjox", lastID: 1}) {
						editSettings(edit: {keyPath: [{property: "p"}], value: $value}) {
							empty {
								alwaysNil
							}
						}
					}
				}
			`,
			Variables: map[string]interface{}{"value": map[string]int{"x": 123}},
			ExpectedResult: `
				{
					"settingsMutation": {
						"editSettings": {
							"empty": null
						}
					}
				}
			`,
		},
	})
}

func TestSettingsMutation_OverwriteSettings(t *testing.T) {
	resetMocks()
	db.Mocks.Users.GetByID = func(context.Context, int32) (*types.User, error) {
		return &types.User{ID: 1}, nil
	}
	db.Mocks.Settings.GetLatest = func(context.Context, api.SettingsSubject) (*api.Settings, error) {
		return &api.Settings{ID: 1, Contents: "{}"}, nil
	}
	db.Mocks.Settings.CreateIfUpToDate = func(ctx context.Context, subject api.SettingsSubject, lastID, authorUserID *int32, contents string) (*api.Settings, error) {
		if want := `x`; contents != want {
			t.Errorf("got %q, want %q", contents, want)
		}
		return &api.Settings{ID: 2, Contents: contents}, nil
	}

	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Context: actor.WithActor(context.Background(), &actor.Actor{UID: 1}),
			Schema:  mustParseGraphQLSchema(t),
			Query: `
				mutation($contents: String!) {
					settingsMutation(input: {subject: "VXNlcjox", lastID: 1}) {
						overwriteSettings(contents: $contents) {
							empty {
								alwaysNil
							}
						}
					}
				}
			`,
			Variables: map[string]interface{}{"contents": "x"},
			ExpectedResult: `
				{
					"settingsMutation": {
						"overwriteSettings": {
							"empty": null
						}
					}
				}
			`,
		},
	})
}
