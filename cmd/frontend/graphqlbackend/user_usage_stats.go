package graphqlbackend

import (
	"context"
	"errors"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/pkg/usagestats"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/pkg/actor"
	"github.com/sourcegraph/sourcegraph/pkg/conf"
)

func (r *UserResolver) UsageStatistics(ctx context.Context) (*userUsageStatisticsResolver, error) {
	if envvar.SourcegraphDotComMode() {
		return nil, errors.New("usage statistics are not available on sourcegraph.com")
	}

	stats, err := usagestats.GetByUserID(r.user.ID)
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

type logUserEventInput struct {
	Event        string
	UserCookieID string
	URL          string
	Argument     *string
}

func (*schemaResolver) LogUserEvent(ctx context.Context, args *struct {
	Input *logUserEventInput
}) (*EmptyResponse, error) {
	if envvar.SourcegraphDotComMode() {
		return nil, nil
	}
	actor := actor.FromContext(ctx)
	err := usagestats.LogActivity(actor.IsAuthenticated(), actor.UID, args.Input.UserCookieID, args.Input.Event)
	if err != nil {
		return nil, err
	}

	if conf.EnableEventLogging() {
		return nil, usagestats.LogEvent(
			args.Input.Event,
			args.Input.URL,
			actor.UID,
			args.Input.UserCookieID,
			args.Input.Argument,
		)
	}
	return nil, nil
}
