package discovery

import (
	"context"

	"github.com/sourcegraph/sourcegraph/internal/insights"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/schema"
)

// SettingStore is a subset of the API exposed by the database.Settings() store.
type SettingStore interface {
	GetLatest(context.Context, api.SettingsSubject) (*api.Settings, error)
	GetLastestSchemaSettings(context.Context, api.SettingsSubject) (*schema.Settings, error)
}

// InsightFilterArgs contains arguments that will filter out insights when discovered if matched.
type InsightFilterArgs struct {
	Ids []string
}

// Discover uses the given settings store to look for insights in the global user settings.
//
// TODO(slimsag): future: include user/org settings and consider security implications of doing so.
// In the future, this will be expanded to also include insights from users/orgs.
func Discover(ctx context.Context, settingStore SettingStore, loader insights.Loader, args InsightFilterArgs) ([]insights.SearchInsight, error) {
	// Get latest Global user settings.
	subject := api.SettingsSubject{Site: true}
	globalSettingsRaw, err := settingStore.GetLatest(ctx, subject)
	if err != nil {
		return nil, err
	}
	globalSettings, err := parseUserSettings(globalSettingsRaw)
	if err != nil {
		return nil, err
	}
	results := convertFromBackendInsight(globalSettings.Insights)

	// load any insights that are integrated from the extensions version
	integrated, err := loader.LoadAll(ctx)
	if err != nil {
		return []insights.SearchInsight{}, err
	}
	results = append(results, integrated...)

	return applyFilters(results, args), nil
}

// convertFromBackendInsight is an adapter method that will transform the 'backend' insight schema to the schema that is
// used by the extensions on the frontend, and will be used in the future. As soon as the backend and frontend are fully integrated these
// 'backend' insights will be deprecated.
func convertFromBackendInsight(backendInsights []*schema.Insight) []insights.SearchInsight {
	converted := make([]insights.SearchInsight, 0)
	for _, backendInsight := range backendInsights {
		var temp insights.SearchInsight
		temp.Title = backendInsight.Title
		temp.Description = backendInsight.Description
		for _, series := range backendInsight.Series {
			temp.Series = append(temp.Series, insights.TimeSeries{
				Name:  series.Label,
				Query: series.Search,
			})
		}
		temp.ID = backendInsight.Id
		converted = append(converted, temp)
	}

	return converted
}

func parseUserSettings(settings *api.Settings) (*schema.Settings, error) {
	if settings == nil {
		// Settings have never been saved for this subject; equivalent to `{}`.
		return &schema.Settings{}, nil
	}
	var v schema.Settings
	if err := jsonc.Unmarshal(settings.Contents, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// applyFilters will apply any filters defined as arguments serially and return the intersection.
func applyFilters(total []insights.SearchInsight, args InsightFilterArgs) []insights.SearchInsight {
	filtered := total

	if len(args.Ids) > 0 {
		filtered = filterByIds(args.Ids, total)
	}

	return filtered
}

func filterByIds(ids []string, insight []insights.SearchInsight) []insights.SearchInsight {
	filtered := make([]insights.SearchInsight, 0)
	keys := make(map[string]bool)
	for _, id := range ids {
		keys[id] = true
	}

	for _, searchInsight := range insight {
		if _, ok := keys[searchInsight.ID]; ok {
			filtered = append(filtered, searchInsight)
		}
	}
	return filtered
}
