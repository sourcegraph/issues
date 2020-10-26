package graphqlbackend

import (
	"context"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/usagestatsdeprecated"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
)

func (r *siteResolver) UsageStatistics(ctx context.Context, args *struct {
	Days   *int32
	Weeks  *int32
	Months *int32
}) (*siteUsageStatisticsResolver, error) {
	opt := &usagestatsdeprecated.SiteUsageStatisticsOptions{}
	if args.Days != nil {
		d := int(*args.Days)
		opt.DayPeriods = &d
	}
	if args.Weeks != nil {
		w := int(*args.Weeks)
		opt.WeekPeriods = &w
	}
	if args.Months != nil {
		m := int(*args.Months)
		opt.MonthPeriods = &m
	}
	activity, err := usagestatsdeprecated.GetSiteUsageStatistics(opt)
	if err != nil {
		return nil, err
	}
	return &siteUsageStatisticsResolver{activity}, nil
}

type siteUsageStatisticsResolver struct {
	siteUsageStatistics *types.SiteUsageStatistics
}

func (s *siteUsageStatisticsResolver) DAUs() []*siteUsagePeriodResolver {
	return s.activities(s.siteUsageStatistics.DAUs)
}

func (s *siteUsageStatisticsResolver) WAUs() []*siteUsagePeriodResolver {
	return s.activities(s.siteUsageStatistics.WAUs)
}

func (s *siteUsageStatisticsResolver) MAUs() []*siteUsagePeriodResolver {
	return s.activities(s.siteUsageStatistics.MAUs)
}

func (s *siteUsageStatisticsResolver) activities(periods []*types.SiteActivityPeriod) []*siteUsagePeriodResolver {
	resolvers := make([]*siteUsagePeriodResolver, 0, len(periods))
	for _, p := range periods {
		resolvers = append(resolvers, &siteUsagePeriodResolver{siteUsagePeriod: p})
	}
	return resolvers
}

type siteUsagePeriodResolver struct {
	siteUsagePeriod *types.SiteActivityPeriod
}

func (s *siteUsagePeriodResolver) StartTime() string {
	return s.siteUsagePeriod.StartTime.Format(time.RFC3339)
}

func (s *siteUsagePeriodResolver) UserCount() int32 {
	return s.siteUsagePeriod.UserCount
}

func (s *siteUsagePeriodResolver) RegisteredUserCount() int32 {
	return s.siteUsagePeriod.RegisteredUserCount
}

func (s *siteUsagePeriodResolver) AnonymousUserCount() int32 {
	return s.siteUsagePeriod.AnonymousUserCount
}

func (s *siteUsagePeriodResolver) IntegrationUserCount() int32 {
	return s.siteUsagePeriod.IntegrationUserCount
}
