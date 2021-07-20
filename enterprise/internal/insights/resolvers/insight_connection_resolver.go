package resolvers

import (
	"context"
	"strconv"
	"sync"

	"github.com/sourcegraph/sourcegraph/internal/insights"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/discovery"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/store"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
)

var _ graphqlbackend.InsightConnectionResolver = &insightConnectionResolver{}

type insightConnectionResolver struct {
	insightsStore   store.Interface
	workerBaseStore *basestore.Store
	settingStore    discovery.SettingStore

	// arguments from query
	ids []string

	// cache results because they are used by multiple fields
	once     sync.Once
	insights []insights.SearchInsight
	next     int64
	err      error
}

func (r *insightConnectionResolver) Nodes(ctx context.Context) ([]graphqlbackend.InsightResolver, error) {
	nodes, _, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}
	resolvers := make([]graphqlbackend.InsightResolver, 0, len(nodes))
	for _, insight := range nodes {
		resolvers = append(resolvers, &insightResolver{
			insightsStore:   r.insightsStore,
			workerBaseStore: r.workerBaseStore,
			insight:         insight,
		})
	}
	return resolvers, nil
}

func (r *insightConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	results, _, err := r.compute(ctx)
	return int32(len(results)), err
}

func (r *insightConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	_, next, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}
	if next != 0 {
		return graphqlutil.NextPageCursor(strconv.Itoa(int(next))), nil
	}
	return graphqlutil.HasNextPage(false), nil
}

func (r *insightConnectionResolver) compute(ctx context.Context) ([]insights.SearchInsight, int64, error) {
	r.once.Do(func() {
		r.insights, r.err = discovery.Discover(ctx, r.settingStore, insights.NewLoader(r.workerBaseStore.Handle().DB()), discovery.InsightFilterArgs{Ids: r.ids})
	})
	return r.insights, r.next, r.err
}

// InsightResolver is also defined here as it is covered by the same tests.

var _ graphqlbackend.InsightResolver = &insightResolver{}

type insightResolver struct {
	insightsStore   store.Interface
	workerBaseStore *basestore.Store
	insight         insights.SearchInsight
}

func (r *insightResolver) ID() string {
	return r.insight.ID
}

func (r *insightResolver) Title() string { return r.insight.Title }

func (r *insightResolver) Description() string { return r.insight.Description }

func (r *insightResolver) Series() []graphqlbackend.InsightSeriesResolver {
	series := r.insight.Series
	resolvers := make([]graphqlbackend.InsightSeriesResolver, 0, len(series))
	for _, series := range series {
		resolvers = append(resolvers, &insightSeriesResolver{
			insightsStore:   r.insightsStore,
			workerBaseStore: r.workerBaseStore,
			series:          series,
		})
	}
	return resolvers
}
