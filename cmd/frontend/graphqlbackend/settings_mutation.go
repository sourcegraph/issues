package graphqlbackend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/graph-gophers/graphql-go"
	"github.com/sourcegraph/jsonx"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

// Deprecated: The GraphQL type Configuration is deprecated.
type configurationResolver struct {
	contents string
	messages []string // error and warning messages
}

func (r *configurationResolver) Contents() JSONCString {
	return JSONCString(r.contents)
}

func (r *configurationResolver) Messages() []string {
	if r.messages == nil {
		return []string{}
	}
	return r.messages
}

type settingsMutationGroupInput struct {
	Subject graphql.ID
	LastID  *int32
}

type settingsMutation struct {
	input   *settingsMutationGroupInput
	subject *settingsSubject
}

// SettingsMutation defines the Mutation.settingsMutation field.
func (r *schemaResolver) SettingsMutation(ctx context.Context, args *struct {
	Input *settingsMutationGroupInput
}) (*settingsMutation, error) {
	n, err := r.nodeByID(ctx, args.Input.Subject)
	if err != nil {
		return nil, err
	}

	subject, err := settingsSubjectForNode(ctx, n)
	if err != nil {
		return nil, err
	}

	// TODO(sqs): support multiple mutations running in a single query that all
	// increment the settings.

	// 🚨 SECURITY: Check whether the viewer can administer this subject (which is equivalent to
	// being able to mutate its settings).
	if canAdmin, err := subject.ViewerCanAdminister(ctx); err != nil {
		return nil, err
	} else if !canAdmin {
		return nil, errors.New("viewer is not allowed to edit these settings")
	}

	return &settingsMutation{
		input:   args.Input,
		subject: subject,
	}, nil
}

// Deprecated: in the GraphQL API
func (r *schemaResolver) ConfigurationMutation(ctx context.Context, args *struct {
	Input *settingsMutationGroupInput
}) (*settingsMutation, error) {
	return r.SettingsMutation(ctx, args)
}

type updateSettingsPayload struct{}

func (updateSettingsPayload) Empty() *EmptyResponse { return nil }

type settingsEdit struct {
	KeyPath                   []*keyPathSegment
	Value                     *JSONValue
	ValueIsJSONCEncodedString bool
}

type keyPathSegment struct {
	Property *string
	Index    *int32
}

func toKeyPath(gqlKeyPath []*keyPathSegment) (jsonx.Path, error) {
	keyPath := make(jsonx.Path, len(gqlKeyPath))
	for i, s := range gqlKeyPath {
		if (s.Property == nil) == (s.Index == nil) {
			return nil, fmt.Errorf("invalid key path segment at index %d: exactly 1 of property and index must be non-null", i)
		}

		var segment jsonx.Segment
		if s.Property != nil {
			segment.IsProperty = true
			segment.Property = *s.Property
		} else {
			segment.Index = int(*s.Index)
		}
		keyPath[i] = segment
	}
	return keyPath, nil
}

func (r *settingsMutation) EditSettings(ctx context.Context, args *struct {
	Edit *settingsEdit
}) (*updateSettingsPayload, error) {
	keyPath, err := toKeyPath(args.Edit.KeyPath)
	if err != nil {
		return nil, err
	}

	remove := args.Edit.Value == nil
	var value interface{}
	if args.Edit.Value != nil {
		value = args.Edit.Value.Value
	}
	if args.Edit.ValueIsJSONCEncodedString {
		s, ok := value.(string)
		if !ok {
			return nil, errors.New("value must be a string for valueIsJSONCEncodedString")
		}
		value = json.RawMessage(s)
	}

	return r.editSettings(ctx, keyPath, value, remove)
}

func (r *settingsMutation) EditConfiguration(ctx context.Context, args *struct {
	Edit *settingsEdit
}) (*updateSettingsPayload, error) {
	return r.EditSettings(ctx, args)
}

func (r *settingsMutation) editSettings(ctx context.Context, keyPath jsonx.Path, value interface{}, remove bool) (*updateSettingsPayload, error) {
	_, err := r.doUpdateSettings(ctx, func(oldSettings string) (edits []jsonx.Edit, err error) {
		if remove {
			edits, _, err = jsonx.ComputePropertyRemoval(oldSettings, keyPath, conf.FormatOptions)
		} else {
			edits, _, err = jsonx.ComputePropertyEdit(oldSettings, keyPath, value, nil, conf.FormatOptions)
		}
		return edits, err
	})
	if err != nil {
		return nil, err
	}
	return &updateSettingsPayload{}, nil
}

func (r *settingsMutation) OverwriteSettings(ctx context.Context, args *struct {
	Contents string
}) (*updateSettingsPayload, error) {
	_, err := settingsCreateIfUpToDate(ctx, r.subject, r.input.LastID, actor.FromContext(ctx).UID, args.Contents)
	if err != nil {
		return nil, err
	}
	return &updateSettingsPayload{}, nil
}

// doUpdateSettings is a helper for updating settings.
func (r *settingsMutation) doUpdateSettings(ctx context.Context, computeEdits func(oldSettings string) ([]jsonx.Edit, error)) (idAfterUpdate int32, err error) {
	currentSettings, err := r.getCurrentSettings(ctx)
	if err != nil {
		return 0, err
	}

	edits, err := computeEdits(currentSettings)
	if err != nil {
		return 0, err
	}
	newSettings, err := jsonx.ApplyEdits(currentSettings, edits...)
	if err != nil {
		return 0, err
	}

	// Write mutated settings.
	updatedSettings, err := settingsCreateIfUpToDate(ctx, r.subject, r.input.LastID, actor.FromContext(ctx).UID, newSettings)
	if err != nil {
		return 0, err
	}
	return updatedSettings.ID, nil
}

func (r *settingsMutation) getCurrentSettings(ctx context.Context) (string, error) {
	// Get the settings file whose contents to mutate.
	settings, err := db.Settings.GetLatest(ctx, r.subject.toSubject())
	if err != nil {
		return "", err
	}
	var data string
	if settings != nil && r.input.LastID != nil && settings.ID == *r.input.LastID {
		data = settings.Contents
	} else if settings == nil && r.input.LastID == nil {
		// noop
	} else {
		intOrNull := func(v *int32) string {
			if v == nil {
				return "null"
			}
			return strconv.FormatInt(int64(*v), 10)
		}
		var lastID *int32
		if settings != nil {
			lastID = &settings.ID
		}
		return "", fmt.Errorf("update settings version mismatch: last ID is %s (mutation wanted %s)", intOrNull(lastID), intOrNull(r.input.LastID))
	}

	return data, nil
}
