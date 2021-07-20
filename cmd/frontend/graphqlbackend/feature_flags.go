package graphqlbackend

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/featureflag"
)

type FeatureFlagResolver struct {
	db    dbutil.DB
	inner *featureflag.FeatureFlag
}

func (f *FeatureFlagResolver) ToFeatureFlagBoolean() (*FeatureFlagBooleanResolver, bool) {
	if f.inner.Bool != nil {
		return &FeatureFlagBooleanResolver{f.db, f.inner}, true
	}
	return nil, false
}

func (f *FeatureFlagResolver) ToFeatureFlagRollout() (*FeatureFlagRolloutResolver, bool) {
	if f.inner.Rollout != nil {
		return &FeatureFlagRolloutResolver{f.db, f.inner}, true
	}
	return nil, false
}

type FeatureFlagBooleanResolver struct {
	db dbutil.DB
	// Invariant: inner.Bool is non-nil
	inner *featureflag.FeatureFlag
}

func (f *FeatureFlagBooleanResolver) Name() string { return f.inner.Name }
func (f *FeatureFlagBooleanResolver) Value() bool  { return f.inner.Bool.Value }
func (f *FeatureFlagBooleanResolver) Overrides(ctx context.Context) ([]*FeatureFlagOverrideResolver, error) {
	overrides, err := database.FeatureFlags(f.db).GetOverridesForFlag(ctx, f.inner.Name)
	if err != nil {
		return nil, err
	}
	return overridesToResolvers(f.db, overrides), nil
}

type FeatureFlagRolloutResolver struct {
	db dbutil.DB
	// Invariant: inner.Rollout is non-nil
	inner *featureflag.FeatureFlag
}

func (f *FeatureFlagRolloutResolver) Name() string              { return f.inner.Name }
func (f *FeatureFlagRolloutResolver) RolloutBasisPoints() int32 { return f.inner.Rollout.Rollout }
func (f *FeatureFlagRolloutResolver) Overrides(ctx context.Context) ([]*FeatureFlagOverrideResolver, error) {
	overrides, err := database.FeatureFlags(f.db).GetOverridesForFlag(ctx, f.inner.Name)
	if err != nil {
		return nil, err
	}
	return overridesToResolvers(f.db, overrides), nil
}

func overridesToResolvers(db dbutil.DB, input []*featureflag.Override) []*FeatureFlagOverrideResolver {
	res := make([]*FeatureFlagOverrideResolver, 0, len(input))
	for _, flag := range input {
		res = append(res, &FeatureFlagOverrideResolver{db, flag})
	}
	return res
}

type FeatureFlagOverrideResolver struct {
	db    dbutil.DB
	inner *featureflag.Override
}

func (f *FeatureFlagOverrideResolver) TargetFlag(ctx context.Context) (*FeatureFlagResolver, error) {
	res, err := database.FeatureFlags(f.db).GetFeatureFlag(ctx, f.inner.FlagName)
	return &FeatureFlagResolver{f.db, res}, err
}
func (f *FeatureFlagOverrideResolver) Value() bool { return f.inner.Value }
func (f *FeatureFlagOverrideResolver) Namespace(ctx context.Context) (*NamespaceResolver, error) {
	if f.inner.UserID != nil {
		u, err := UserByIDInt32(ctx, f.db, *f.inner.UserID)
		return &NamespaceResolver{u}, err
	} else if f.inner.OrgID != nil {
		o, err := OrgByIDInt32(ctx, f.db, *f.inner.OrgID)
		return &NamespaceResolver{o}, err
	}
	return nil, errors.Errorf("one of userID or orgID must be set")
}
func (f *FeatureFlagOverrideResolver) ID() graphql.ID {
	return marshalOverrideID(overrideSpec{
		UserID:   f.inner.UserID,
		OrgID:    f.inner.OrgID,
		FlagName: f.inner.FlagName,
	})
}

type overrideSpec struct {
	UserID, OrgID *int32
	FlagName      string
}

func marshalOverrideID(spec overrideSpec) graphql.ID {
	return relay.MarshalID("FeatureFlagOverride", spec)
}

func unmarshalOverrideID(id graphql.ID) (spec overrideSpec, err error) {
	err = relay.UnmarshalSpec(id, &spec)
	return
}

type EvaluatedFeatureFlagResolver struct {
	name  string
	value bool
}

func (e *EvaluatedFeatureFlagResolver) Name() string {
	return e.name
}

func (e *EvaluatedFeatureFlagResolver) Value() bool {
	return e.value
}

func (r *schemaResolver) ViewerFeatureFlags(ctx context.Context) []*EvaluatedFeatureFlagResolver {
	f := featureflag.FromContext(ctx)
	return evaluatedFlagsToResolvers(f)
}

func evaluatedFlagsToResolvers(input map[string]bool) []*EvaluatedFeatureFlagResolver {
	res := make([]*EvaluatedFeatureFlagResolver, 0, len(input))
	for k, v := range input {
		res = append(res, &EvaluatedFeatureFlagResolver{name: k, value: v})
	}
	return res
}

func (r *schemaResolver) FeatureFlags(ctx context.Context) ([]*FeatureFlagResolver, error) {
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return nil, err
	}
	flags, err := database.FeatureFlags(r.db).GetFeatureFlags(ctx)
	if err != nil {
		return nil, err
	}
	return flagsToResolvers(r.db, flags), nil
}

func flagsToResolvers(db dbutil.DB, flags []*featureflag.FeatureFlag) []*FeatureFlagResolver {
	res := make([]*FeatureFlagResolver, 0, len(flags))
	for _, flag := range flags {
		res = append(res, &FeatureFlagResolver{db, flag})
	}
	return res
}

func (r *schemaResolver) CreateFeatureFlag(ctx context.Context, args struct {
	Name               string
	Value              *bool
	RolloutBasisPoints *int32
}) (*FeatureFlagResolver, error) {
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return nil, err
	}

	ff := database.FeatureFlags(r.db)

	var res *featureflag.FeatureFlag
	var err error
	if args.Value != nil {
		res, err = ff.CreateBool(ctx, args.Name, *args.Value)
	} else if args.RolloutBasisPoints != nil {
		res, err = ff.CreateRollout(ctx, args.Name, *args.RolloutBasisPoints)
	} else {
		return nil, errors.Errorf("either 'value' or 'rolloutBasisPoints' must be set")
	}

	return &FeatureFlagResolver{r.db, res}, err
}

func (r *schemaResolver) DeleteFeatureFlag(ctx context.Context, args struct {
	Name string
}) (*EmptyResponse, error) {
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return nil, err
	}
	return &EmptyResponse{}, database.FeatureFlags(r.db).DeleteFeatureFlag(ctx, args.Name)
}

func (r *schemaResolver) UpdateFeatureFlag(ctx context.Context, args struct {
	Name               string
	Value              *bool
	RolloutBasisPoints *int32
}) (*FeatureFlagResolver, error) {
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return nil, err
	}
	ff := &featureflag.FeatureFlag{Name: args.Name}
	if args.Value != nil {
		ff.Bool = &featureflag.FeatureFlagBool{Value: *args.Value}
	} else if args.RolloutBasisPoints != nil {
		ff.Rollout = &featureflag.FeatureFlagRollout{Rollout: *args.RolloutBasisPoints}
	} else {
		return nil, errors.Errorf("either 'value' or 'rolloutBasisPoints' must be set")
	}

	res, err := database.FeatureFlags(r.db).UpdateFeatureFlag(ctx, ff)
	return &FeatureFlagResolver{r.db, res}, err
}

func (r *schemaResolver) CreateFeatureFlagOverride(ctx context.Context, args struct {
	Namespace graphql.ID
	FlagName  string
	Value     bool
}) (*FeatureFlagOverrideResolver, error) {
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return nil, err
	}

	fo := &featureflag.Override{
		FlagName: args.FlagName,
		Value:    args.Value,
	}

	var uid, oid int32
	if err := UnmarshalNamespaceID(args.Namespace, &uid, &oid); err != nil {
		return nil, err
	}

	if uid != 0 {
		fo.UserID = &uid
	} else if oid != 0 {
		fo.OrgID = &oid
	}
	res, err := database.FeatureFlags(r.db).CreateOverride(ctx, fo)
	return &FeatureFlagOverrideResolver{r.db, res}, err
}

func (r *schemaResolver) DeleteFeatureFlagOverride(ctx context.Context, args struct {
	ID graphql.ID
}) (*EmptyResponse, error) {
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return nil, err
	}
	spec, err := unmarshalOverrideID(args.ID)
	if err != nil {
		return &EmptyResponse{}, err
	}
	return &EmptyResponse{}, database.FeatureFlags(r.db).DeleteOverride(ctx, spec.OrgID, spec.UserID, spec.FlagName)
}

func (r *schemaResolver) UpdateFeatureFlagOverride(ctx context.Context, args struct {
	ID    graphql.ID
	Value bool
}) (*FeatureFlagOverrideResolver, error) {
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return nil, err
	}
	spec, err := unmarshalOverrideID(args.ID)
	if err != nil {
		return nil, err
	}

	res, err := database.FeatureFlags(r.db).UpdateOverride(ctx, spec.OrgID, spec.UserID, spec.FlagName, args.Value)
	return &FeatureFlagOverrideResolver{r.db, res}, err
}
