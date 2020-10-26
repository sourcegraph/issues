package usagestats

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

func GetSiteUsageStats(ctx context.Context, monthsOnly bool) (*types.SiteUsageStatistics, error) {
	summary, err := db.EventLogs.SiteUsage(ctx)
	if err != nil {
		return nil, err
	}

	stats := groupSiteUsageStats(summary, monthsOnly)
	return stats, nil
}

func groupSiteUsageStats(summary types.SiteUsageSummary, monthsOnly bool) *types.SiteUsageStatistics {
	stats := &types.SiteUsageStatistics{
		DAUs: []*types.SiteActivityPeriod{
			{
				StartTime:            summary.Day,
				UserCount:            summary.UniquesDay,
				RegisteredUserCount:  summary.RegisteredUniquesDay,
				AnonymousUserCount:   summary.UniquesDay - summary.RegisteredUniquesDay,
				IntegrationUserCount: summary.IntegrationUniquesDay,
			},
		},
		WAUs: []*types.SiteActivityPeriod{
			{
				StartTime:            summary.Week,
				UserCount:            summary.UniquesWeek,
				RegisteredUserCount:  summary.RegisteredUniquesWeek,
				AnonymousUserCount:   summary.UniquesWeek - summary.RegisteredUniquesWeek,
				IntegrationUserCount: summary.IntegrationUniquesWeek,
			},
		},
		MAUs: []*types.SiteActivityPeriod{
			{
				StartTime:            summary.Month,
				UserCount:            summary.UniquesMonth,
				RegisteredUserCount:  summary.RegisteredUniquesMonth,
				AnonymousUserCount:   summary.UniquesMonth - summary.RegisteredUniquesMonth,
				IntegrationUserCount: summary.IntegrationUniquesMonth,
			},
		},
	}

	if monthsOnly {
		stats.DAUs = []*types.SiteActivityPeriod{}
		stats.WAUs = []*types.SiteActivityPeriod{}
	}

	return stats
}

// GetAggregatedStats returns aggregates statistics for code intel and search usage.
func GetAggregatedStats(ctx context.Context) (*types.CodeIntelUsageStatistics, *types.SearchUsageStatistics, error) {
	events, err := db.EventLogs.AggregatedEvents(ctx)
	if err != nil {
		return nil, nil, err
	}

	codeIntelStats, searchStats := groupAggreatedStats(events)
	return codeIntelStats, searchStats, nil
}

func groupAggreatedStats(events []types.AggregatedEvent) (*types.CodeIntelUsageStatistics, *types.SearchUsageStatistics) {
	codeIntelUsageStats := &types.CodeIntelUsageStatistics{
		Daily:   []*types.CodeIntelUsagePeriod{newCodeIntelUsagePeriod()},
		Weekly:  []*types.CodeIntelUsagePeriod{newCodeIntelUsagePeriod()},
		Monthly: []*types.CodeIntelUsagePeriod{newCodeIntelUsagePeriod()},
	}

	searchUsageStats := &types.SearchUsageStatistics{
		Daily:   []*types.SearchUsagePeriod{newSearchUsagePeriod()},
		Weekly:  []*types.SearchUsagePeriod{newSearchUsagePeriod()},
		Monthly: []*types.SearchUsagePeriod{newSearchUsagePeriod()},
	}

	for _, event := range events {
		insertCodeIntelEventStatistics(event, codeIntelUsageStats)
		insertSearchEventStatistics(event, searchUsageStats)
	}

	return codeIntelUsageStats, searchUsageStats
}

func newCodeIntelUsagePeriod() *types.CodeIntelUsagePeriod {
	return &types.CodeIntelUsagePeriod{
		Hover:       newCodeIntelEventCategory(),
		Definitions: newCodeIntelEventCategory(),
		References:  newCodeIntelEventCategory(),
	}
}

func insertCodeIntelEventStatistics(event types.AggregatedEvent, statistics *types.CodeIntelUsageStatistics) {
	extractors := map[string]func(p *types.CodeIntelUsagePeriod) *types.CodeIntelEventStatistics{
		"codeintel.lsifHover":         func(p *types.CodeIntelUsagePeriod) *types.CodeIntelEventStatistics { return p.Hover.LSIF },
		"codeintel.searchHover":       func(p *types.CodeIntelUsagePeriod) *types.CodeIntelEventStatistics { return p.Hover.Search },
		"codeintel.lsifDefinitions":   func(p *types.CodeIntelUsagePeriod) *types.CodeIntelEventStatistics { return p.Definitions.LSIF },
		"codeintel.searchDefinitions": func(p *types.CodeIntelUsagePeriod) *types.CodeIntelEventStatistics { return p.Definitions.Search },
		"codeintel.lsifReferences":    func(p *types.CodeIntelUsagePeriod) *types.CodeIntelEventStatistics { return p.References.LSIF },
		"codeintel.searchReferences":  func(p *types.CodeIntelUsagePeriod) *types.CodeIntelEventStatistics { return p.References.Search },
	}

	extractor, ok := extractors[event.Name]
	if !ok {
		return
	}

	statistics.Monthly[0].StartTime = event.Month
	month := extractor(statistics.Monthly[0])
	month.EventsCount = &event.TotalMonth
	month.UsersCount = event.UniquesMonth

	statistics.Weekly[0].StartTime = event.Week
	week := extractor(statistics.Weekly[0])
	week.EventsCount = &event.TotalWeek
	week.UsersCount = event.UniquesWeek

	statistics.Daily[0].StartTime = event.Day
	day := extractor(statistics.Daily[0])
	day.EventsCount = &event.TotalDay
	day.UsersCount = event.UniquesDay
}

func newSearchUsagePeriod() *types.SearchUsagePeriod {
	return newSearchEventPeriod()
}

func insertSearchEventStatistics(event types.AggregatedEvent, statistics *types.SearchUsageStatistics) {
	extractors := map[string]func(p *types.SearchUsagePeriod) *types.SearchEventStatistics{
		"search.latencies.literal":    func(p *types.SearchUsagePeriod) *types.SearchEventStatistics { return p.Literal },
		"search.latencies.regexp":     func(p *types.SearchUsagePeriod) *types.SearchEventStatistics { return p.Regexp },
		"search.latencies.structural": func(p *types.SearchUsagePeriod) *types.SearchEventStatistics { return p.Structural },
		"search.latencies.file":       func(p *types.SearchUsagePeriod) *types.SearchEventStatistics { return p.File },
		"search.latencies.repo":       func(p *types.SearchUsagePeriod) *types.SearchEventStatistics { return p.Repo },
		"search.latencies.diff":       func(p *types.SearchUsagePeriod) *types.SearchEventStatistics { return p.Diff },
		"search.latencies.commit":     func(p *types.SearchUsagePeriod) *types.SearchEventStatistics { return p.Commit },
		"search.latencies.symbol":     func(p *types.SearchUsagePeriod) *types.SearchEventStatistics { return p.Symbol },
	}

	extractor, ok := extractors[event.Name]
	if !ok {
		return
	}

	makeLatencies := func(values []float64) *types.SearchEventLatencies {
		for len(values) < 3 {
			// If event logs didn't have samples, add zero values
			values = append(values, 0)
		}

		return &types.SearchEventLatencies{P50: values[0], P90: values[1], P99: values[2]}
	}

	statistics.Monthly[0].StartTime = event.Month
	month := extractor(statistics.Monthly[0])
	month.EventsCount = &event.TotalMonth
	month.UserCount = &event.UniquesMonth
	month.EventLatencies = makeLatencies(event.LatenciesMonth)

	statistics.Weekly[0].StartTime = event.Week
	week := extractor(statistics.Weekly[0])
	week.EventsCount = &event.TotalWeek
	week.UserCount = &event.UniquesWeek
	week.EventLatencies = makeLatencies(event.LatenciesWeek)

	statistics.Daily[0].StartTime = event.Day
	day := extractor(statistics.Daily[0])
	day.EventsCount = &event.TotalDay
	day.UserCount = &event.UniquesDay
	day.EventLatencies = makeLatencies(event.LatenciesDay)
}

func newCodeIntelEventCategory() *types.CodeIntelEventCategoryStatistics {
	return &types.CodeIntelEventCategoryStatistics{
		LSIF:   codeIntelEventStatistics(),
		LSP:    codeIntelEventStatistics(),
		Search: codeIntelEventStatistics(),
	}
}

func codeIntelEventStatistics() *types.CodeIntelEventStatistics {
	return &types.CodeIntelEventStatistics{}
}

func newSearchEventPeriod() *types.SearchUsagePeriod {
	return &types.SearchUsagePeriod{
		Literal:            newSearchEventStatistics(),
		Regexp:             newSearchEventStatistics(),
		Structural:         newSearchEventStatistics(),
		File:               newSearchEventStatistics(),
		Repo:               newSearchEventStatistics(),
		Diff:               newSearchEventStatistics(),
		Commit:             newSearchEventStatistics(),
		Symbol:             newSearchEventStatistics(),
		Case:               newSearchCountStatistics(),
		Committer:          newSearchCountStatistics(),
		Lang:               newSearchCountStatistics(),
		Fork:               newSearchCountStatistics(),
		Archived:           newSearchCountStatistics(),
		Count:              newSearchCountStatistics(),
		Timeout:            newSearchCountStatistics(),
		Content:            newSearchCountStatistics(),
		Before:             newSearchCountStatistics(),
		After:              newSearchCountStatistics(),
		Author:             newSearchCountStatistics(),
		Message:            newSearchCountStatistics(),
		Index:              newSearchCountStatistics(),
		Repogroup:          newSearchCountStatistics(),
		Repohasfile:        newSearchCountStatistics(),
		Repohascommitafter: newSearchCountStatistics(),
		PatternType:        newSearchCountStatistics(),
		Type:               newSearchCountStatistics(),
		SearchModes:        newSearchModeUsageStatistics(),
	}
}

func newSearchEventStatistics() *types.SearchEventStatistics {
	return &types.SearchEventStatistics{EventLatencies: &types.SearchEventLatencies{}}
}

func newSearchCountStatistics() *types.SearchCountStatistics {
	return &types.SearchCountStatistics{}
}

func newSearchModeUsageStatistics() *types.SearchModeUsageStatistics {
	return &types.SearchModeUsageStatistics{Interactive: &types.SearchCountStatistics{}, PlainText: &types.SearchCountStatistics{}}
}
