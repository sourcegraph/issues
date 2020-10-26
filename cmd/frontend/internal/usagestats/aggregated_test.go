package usagestats

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
)

func TestGroupSiteUsageStats(t *testing.T) {
	t1 := time.Now().UTC()
	t2 := t1.Add(time.Hour)
	t3 := t2.Add(time.Hour)

	summary := types.SiteUsageSummary{
		Month:                   t1,
		Week:                    t2,
		Day:                     t3,
		UniquesMonth:            4,
		UniquesWeek:             5,
		UniquesDay:              6,
		RegisteredUniquesMonth:  1,
		RegisteredUniquesWeek:   2,
		RegisteredUniquesDay:    3,
		IntegrationUniquesMonth: 7,
		IntegrationUniquesWeek:  8,
		IntegrationUniquesDay:   9,
		ManageUniquesMonth:      10,
		CodeUniquesMonth:        11,
		VerifyUniquesMonth:      12,
		MonitorUniquesMonth:     13,
		ManageUniquesWeek:       14,
		CodeUniquesWeek:         15,
		VerifyUniquesWeek:       16,
		MonitorUniquesWeek:      17,
	}
	siteUsageStats := groupSiteUsageStats(summary, false)

	expectedSiteUsageStats := &types.SiteUsageStatistics{
		DAUs: []*types.SiteActivityPeriod{
			{
				StartTime:            t3,
				UserCount:            6,
				RegisteredUserCount:  3,
				AnonymousUserCount:   3,
				IntegrationUserCount: 9,
			},
		},
		WAUs: []*types.SiteActivityPeriod{
			{
				StartTime:            t2,
				UserCount:            5,
				RegisteredUserCount:  2,
				AnonymousUserCount:   3,
				IntegrationUserCount: 8,
			},
		},
		MAUs: []*types.SiteActivityPeriod{
			{
				StartTime:            t1,
				UserCount:            4,
				RegisteredUserCount:  1,
				AnonymousUserCount:   3,
				IntegrationUserCount: 7,
			},
		},
	}
	if diff := cmp.Diff(expectedSiteUsageStats, siteUsageStats); diff != "" {
		t.Fatal(diff)
	}
}

func TestGroupSiteUsageStatsMonthsOnly(t *testing.T) {
	t1 := time.Now().UTC()
	t2 := t1.Add(time.Hour)
	t3 := t2.Add(time.Hour)

	summary := types.SiteUsageSummary{
		Month:                   t1,
		Week:                    t2,
		Day:                     t3,
		UniquesMonth:            4,
		UniquesWeek:             5,
		UniquesDay:              6,
		RegisteredUniquesMonth:  1,
		RegisteredUniquesWeek:   2,
		RegisteredUniquesDay:    3,
		IntegrationUniquesMonth: 7,
		IntegrationUniquesWeek:  8,
		IntegrationUniquesDay:   9,
		ManageUniquesMonth:      10,
		CodeUniquesMonth:        11,
		VerifyUniquesMonth:      12,
		MonitorUniquesMonth:     13,
		ManageUniquesWeek:       14,
		CodeUniquesWeek:         15,
		VerifyUniquesWeek:       16,
		MonitorUniquesWeek:      17,
	}
	siteUsageStats := groupSiteUsageStats(summary, true)

	expectedSiteUsageStats := &types.SiteUsageStatistics{
		DAUs: []*types.SiteActivityPeriod{},
		WAUs: []*types.SiteActivityPeriod{},
		MAUs: []*types.SiteActivityPeriod{
			{
				StartTime:            t1,
				UserCount:            4,
				RegisteredUserCount:  1,
				AnonymousUserCount:   3,
				IntegrationUserCount: 7,
			},
		},
	}
	if diff := cmp.Diff(expectedSiteUsageStats, siteUsageStats); diff != "" {
		t.Fatal(diff)
	}
}

func TestGroupAggregatedStats(t *testing.T) {
	t1 := time.Now().UTC()
	t2 := t1.Add(time.Hour)
	t3 := t2.Add(time.Hour)

	codeIntelStats, searchStats := groupAggreatedStats([]types.AggregatedEvent{
		{
			Name:           "codeintel.lsifHover",
			Month:          t1,
			Week:           t2,
			Day:            t3,
			TotalMonth:     11,
			TotalWeek:      12,
			TotalDay:       13,
			UniquesMonth:   14,
			UniquesWeek:    15,
			UniquesDay:     16,
			LatenciesMonth: []float64{11, 12, 13},
			LatenciesWeek:  []float64{14, 15, 16},
			LatenciesDay:   []float64{17, 18, 19},
		},
		{
			Name:           "codeintel.lsifDefinitions",
			Month:          t1,
			Week:           t2,
			Day:            t3,
			TotalMonth:     21,
			TotalWeek:      22,
			TotalDay:       23,
			UniquesMonth:   24,
			UniquesWeek:    25,
			UniquesDay:     26,
			LatenciesMonth: []float64{21, 22, 23},
			LatenciesWeek:  []float64{24, 25, 26},
			LatenciesDay:   []float64{27, 28, 29},
		},
		{
			Name:           "search.latencies.structural",
			Month:          t1,
			Week:           t2,
			Day:            t3,
			TotalMonth:     31,
			TotalWeek:      32,
			TotalDay:       33,
			UniquesMonth:   34,
			UniquesWeek:    35,
			UniquesDay:     36,
			LatenciesMonth: []float64{31, 32, 33},
			LatenciesWeek:  []float64{34, 35, 36},
			LatenciesDay:   []float64{37, 38, 39},
		},
		{
			Name:           "search.latencies.commit",
			Month:          t1,
			Week:           t2,
			Day:            t3,
			TotalMonth:     41,
			TotalWeek:      42,
			TotalDay:       43,
			UniquesMonth:   44,
			UniquesWeek:    45,
			UniquesDay:     46,
			LatenciesMonth: []float64{41, 42, 43},
			LatenciesWeek:  []float64{44, 45, 46},
			LatenciesDay:   []float64{47, 48, 49},
		},
	})

	intptr := func(i int32) *int32 {
		return &i
	}

	expectedCodeIntelStats := &types.CodeIntelUsageStatistics{
		Daily: []*types.CodeIntelUsagePeriod{
			{
				StartTime: t3,
				Hover: &types.CodeIntelEventCategoryStatistics{
					LSIF: &types.CodeIntelEventStatistics{
						UsersCount:  16,
						EventsCount: intptr(13),
					},
					LSP:    codeIntelEventStatistics(),
					Search: codeIntelEventStatistics(),
				},
				Definitions: &types.CodeIntelEventCategoryStatistics{
					LSIF: &types.CodeIntelEventStatistics{
						UsersCount:  26,
						EventsCount: intptr(23),
					},
					LSP:    codeIntelEventStatistics(),
					Search: codeIntelEventStatistics(),
				},
				References: newCodeIntelEventCategory(),
			},
		},
		Weekly: []*types.CodeIntelUsagePeriod{
			{
				StartTime: t2,
				Hover: &types.CodeIntelEventCategoryStatistics{
					LSIF: &types.CodeIntelEventStatistics{
						UsersCount:  15,
						EventsCount: intptr(12),
					},
					LSP:    codeIntelEventStatistics(),
					Search: codeIntelEventStatistics(),
				},
				Definitions: &types.CodeIntelEventCategoryStatistics{
					LSIF: &types.CodeIntelEventStatistics{
						UsersCount:  25,
						EventsCount: intptr(22),
					},
					LSP:    codeIntelEventStatistics(),
					Search: codeIntelEventStatistics(),
				},
				References: newCodeIntelEventCategory(),
			},
		},
		Monthly: []*types.CodeIntelUsagePeriod{
			{
				StartTime: t1,
				Hover: &types.CodeIntelEventCategoryStatistics{
					LSIF: &types.CodeIntelEventStatistics{
						UsersCount:  14,
						EventsCount: intptr(11),
					},
					LSP:    codeIntelEventStatistics(),
					Search: codeIntelEventStatistics(),
				},
				Definitions: &types.CodeIntelEventCategoryStatistics{
					LSIF: &types.CodeIntelEventStatistics{
						UsersCount:  24,
						EventsCount: intptr(21),
					},
					LSP:    codeIntelEventStatistics(),
					Search: codeIntelEventStatistics(),
				},
				References: newCodeIntelEventCategory(),
			},
		},
	}
	if diff := cmp.Diff(expectedCodeIntelStats, codeIntelStats); diff != "" {
		t.Fatal(diff)
	}

	expectedSearchStats := &types.SearchUsageStatistics{
		Daily: []*types.SearchUsagePeriod{
			{
				StartTime: t3,
				Literal:   newSearchEventStatistics(),
				Regexp:    newSearchEventStatistics(),
				Structural: &types.SearchEventStatistics{
					EventsCount:    intptr(33),
					UserCount:      intptr(36),
					EventLatencies: &types.SearchEventLatencies{P50: 37, P90: 38, P99: 39},
				},
				File: newSearchEventStatistics(),
				Repo: newSearchEventStatistics(),
				Diff: newSearchEventStatistics(),
				Commit: &types.SearchEventStatistics{
					EventsCount:    intptr(43),
					UserCount:      intptr(46),
					EventLatencies: &types.SearchEventLatencies{P50: 47, P90: 48, P99: 49},
				},
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
			},
		},
		Weekly: []*types.SearchUsagePeriod{
			{
				StartTime: t2,
				Literal:   newSearchEventStatistics(),
				Regexp:    newSearchEventStatistics(),
				Structural: &types.SearchEventStatistics{
					EventsCount:    intptr(32),
					UserCount:      intptr(35),
					EventLatencies: &types.SearchEventLatencies{P50: 34, P90: 35, P99: 36},
				},
				File: newSearchEventStatistics(),
				Repo: newSearchEventStatistics(),
				Diff: newSearchEventStatistics(),
				Commit: &types.SearchEventStatistics{
					EventsCount:    intptr(42),
					UserCount:      intptr(45),
					EventLatencies: &types.SearchEventLatencies{P50: 44, P90: 45, P99: 46},
				},
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
			},
		},
		Monthly: []*types.SearchUsagePeriod{
			{
				StartTime: t1,
				Literal:   newSearchEventStatistics(),
				Regexp:    newSearchEventStatistics(),
				Structural: &types.SearchEventStatistics{
					EventsCount:    intptr(31),
					UserCount:      intptr(34),
					EventLatencies: &types.SearchEventLatencies{P50: 31, P90: 32, P99: 33},
				},
				File: newSearchEventStatistics(),
				Repo: newSearchEventStatistics(),
				Diff: newSearchEventStatistics(),
				Commit: &types.SearchEventStatistics{
					EventsCount:    intptr(41),
					UserCount:      intptr(44),
					EventLatencies: &types.SearchEventLatencies{P50: 41, P90: 42, P99: 43},
				},
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
			},
		},
	}
	if diff := cmp.Diff(expectedSearchStats, searchStats); diff != "" {
		t.Fatal(diff)
	}
}
