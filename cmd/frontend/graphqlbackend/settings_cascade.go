package graphqlbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

// settingsCascade implements the GraphQL type SettingsCascade (and the deprecated type ConfigurationCascade).
//
// It resolves settings from multiple sources.  When there is overlap between values, they will be
// merged in the following cascading order (first is lowest precedence):
//
// - Global site settings
// - Organization settings
// - Current user settings
type settingsCascade struct {
	// At most 1 of these fields is set.
	unauthenticatedActor bool
	subject              *settingsSubject
}

var mockSettingsCascadeSubjects func() ([]*settingsSubject, error)

func (r *settingsCascade) Subjects(ctx context.Context) ([]*settingsSubject, error) {
	if mockSettingsCascadeSubjects != nil {
		return mockSettingsCascadeSubjects()
	}

	subjects := []*settingsSubject{{defaultSettings: singletonDefaultSettingsResolver}, {site: singletonSiteResolver}}

	if r.unauthenticatedActor {
		return subjects, nil
	}

	switch {
	case r.subject.site != nil:
		// Nothing more to do.

	case r.subject.org != nil:
		subjects = append(subjects, r.subject)

	case r.subject.user != nil:
		orgs, err := db.Orgs.GetByUserID(ctx, r.subject.user.user.ID)
		if err != nil {
			return nil, err
		}
		// Stable-sort the orgs so that the priority of their settings is stable.
		sort.Slice(orgs, func(i, j int) bool {
			return orgs[i].ID < orgs[j].ID
		})
		// Apply the user's orgs' settings.
		for _, org := range orgs {
			subjects = append(subjects, &settingsSubject{org: &OrgResolver{org}})
		}
		// Apply the user's own settings last (it has highest priority).
		subjects = append(subjects, r.subject)

	default:
		return nil, errUnknownSettingsSubject
	}

	return subjects, nil
}

// viewerFinalSettings returns the final (merged) settings for the viewer.
func viewerFinalSettings(ctx context.Context) (*configurationResolver, error) {
	cascade, err := (&schemaResolver{}).ViewerSettings(ctx)
	if err != nil {
		return nil, err
	}
	return cascade.Merged(ctx)
}

func (r *settingsCascade) Final(ctx context.Context) (string, error) {
	var allSettings []string
	subjects, err := r.Subjects(ctx)
	if err != nil {
		return "", err
	}
	for _, s := range subjects {
		settings, err := s.LatestSettings(ctx)
		if err != nil {
			return "", err
		}
		if settings != nil {
			allSettings = append(allSettings, settings.settings.Contents)
		}
	}
	final, err := mergeSettings(allSettings)
	return string(final), err
}

// Deprecated: in the GraphQL API
func (r *settingsCascade) Merged(ctx context.Context) (_ *configurationResolver, err error) {
	tr, ctx := trace.New(ctx, "Merged", "")
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()
	var messages []string
	s, err := r.Final(ctx)
	if err != nil {
		messages = append(messages, err.Error())
	}
	return &configurationResolver{contents: string(s), messages: messages}, nil
}

// deeplyMergedSettingsFields contains the names of top-level settings fields whose values should be
// merged if they appear in multiple cascading settings. The value is the merge depth (how many
// levels into the object should the merging occur).
//
// For example, suppose org settings is {"a":[1]} and user settings is {"a":[2]}. If "a" is NOT a
// deeply merged field, the merged settings would be {"a":[2]}. If "a" IS a deeply merged field with
// depth >= 1, then the merged settings would be {"a":[1,2].}
var deeplyMergedSettingsFields = map[string]int{
	"search.scopes":           1,
	"search.savedQueries":     1,
	"search.repositoryGroups": 1,
	"quicklinks":              1,
	"motd":                    1,
	"extensions":              1,
}

// mergeSettings merges the specified JSON settings documents together to produce a single JSON
// settings document. The deep merging behavior is described in the documentation for
// deeplyMergedSettingsFields.
func mergeSettings(jsonSettingsStrings []string) ([]byte, error) {
	var errs []error
	merged := map[string]interface{}{}
	for _, s := range jsonSettingsStrings {
		var o map[string]interface{}
		if err := jsonc.Unmarshal(s, &o); err != nil {
			errs = append(errs, err)
		}
		for name, value := range o {
			depth := deeplyMergedSettingsFields[name]
			mergeSettingsValues(merged, name, value, depth)
		}
	}
	out, err := json.Marshal(merged)
	if err != nil {
		errs = append(errs, err)
	}
	if len(errs) == 0 {
		return out, nil
	}
	return out, fmt.Errorf("errors merging settings: %q", errs)
}

func mergeSettingsValues(dst map[string]interface{}, field string, value interface{}, depth int) {
	// Try to deeply merge this field.
	if depth > 0 {
		if mv, ok := dst[field].([]interface{}); dst[field] == nil || ok {
			if cv, ok := value.([]interface{}); dst[field] != nil || (value != nil && ok) {
				dst[field] = append(mv, cv...)
				return
			}
		} else if mv, ok := dst[field].(map[string]interface{}); dst[field] == nil || ok {
			if cv, ok := value.(map[string]interface{}); dst[field] != nil || (value != nil && ok) {
				for key, value := range cv {
					mergeSettingsValues(mv, key, value, depth-1)
				}
				dst[field] = mv
				return
			}
		}
	}

	// Otherwise just clobber any existing value.
	dst[field] = value
}

func (schemaResolver) ViewerSettings(ctx context.Context) (*settingsCascade, error) {
	user, err := CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return &settingsCascade{unauthenticatedActor: true}, nil
	}
	return &settingsCascade{subject: &settingsSubject{user: user}}, nil
}

// Deprecated: in the GraphQL API
func (schemaResolver) ViewerConfiguration(ctx context.Context) (*settingsCascade, error) {
	return schemaResolver{}.ViewerSettings(ctx)
}
