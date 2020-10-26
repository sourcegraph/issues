package backend

import (
	"context"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/schema"
)

// Configuration backend.
var Configuration = &configuration{}

type configuration struct{}

// GetForSubject gets the latest settings for a single settings subject, without performing any
// cascading (merging settings from multiple subjects).
func (configuration) GetForSubject(ctx context.Context, subject api.SettingsSubject) (*schema.Settings, error) {
	settings, err := db.Settings.GetLatest(ctx, subject)
	if err != nil {
		return nil, err
	}

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
