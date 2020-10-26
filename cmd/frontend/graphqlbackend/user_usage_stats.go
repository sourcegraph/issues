package graphqlbackend

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/usagestats"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/usagestatsdeprecated"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/conf"
)

func (r *UserResolver) UsageStatistics(ctx context.Context) (*userUsageStatisticsResolver, error) {
	if envvar.SourcegraphDotComMode() {
		if err := backend.CheckSiteAdminOrSameUser(ctx, r.user.ID); err != nil {
			return nil, err
		}
	}

	stats, err := usagestatsdeprecated.GetByUserID(r.user.ID)
	if err != nil {
		return nil, err
	}
	return &userUsageStatisticsResolver{stats}, nil
}

type userUsageStatisticsResolver struct {
	userUsageStatistics *types.UserUsageStatistics
}

func (s *userUsageStatisticsResolver) PageViews() int32 { return s.userUsageStatistics.PageViews }

func (s *userUsageStatisticsResolver) SearchQueries() int32 {
	return s.userUsageStatistics.SearchQueries
}

func (s *userUsageStatisticsResolver) CodeIntelligenceActions() int32 {
	return s.userUsageStatistics.CodeIntelligenceActions
}

func (s *userUsageStatisticsResolver) FindReferencesActions() int32 {
	return s.userUsageStatistics.FindReferencesActions
}

func (s *userUsageStatisticsResolver) LastActiveTime() *string {
	if s.userUsageStatistics.LastActiveTime != nil {
		t := s.userUsageStatistics.LastActiveTime.Format(time.RFC3339)
		return &t
	}
	return nil
}

func (s *userUsageStatisticsResolver) LastActiveCodeHostIntegrationTime() *string {
	if s.userUsageStatistics.LastCodeHostIntegrationTime != nil {
		t := s.userUsageStatistics.LastCodeHostIntegrationTime.Format(time.RFC3339)
		return &t
	}
	return nil
}

func (*schemaResolver) LogUserEvent(ctx context.Context, args *struct {
	Event        string
	UserCookieID string
}) (*EmptyResponse, error) {
	actor := actor.FromContext(ctx)
	return nil, usagestatsdeprecated.LogActivity(actor.IsAuthenticated(), actor.UID, args.UserCookieID, args.Event)
}

func (*schemaResolver) LogEvent(ctx context.Context, args *struct {
	Event        string
	UserCookieID string
	URL          string
	Source       string
	Argument     *string
}) (*EmptyResponse, error) {
	if !conf.EventLoggingEnabled() {
		return nil, nil
	}

	var payload json.RawMessage
	if args.Argument != nil {
		if err := json.Unmarshal([]byte(*args.Argument), &payload); err != nil {
			return nil, err
		}
	}

	actor := actor.FromContext(ctx)
	return nil, usagestats.LogEvent(ctx, usagestats.Event{
		EventName:    args.Event,
		URL:          args.URL,
		UserID:       actor.UID,
		UserCookieID: args.UserCookieID,
		Source:       args.Source,
		Argument:     payload,
	})
}
