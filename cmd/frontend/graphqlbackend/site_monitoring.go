package graphqlbackend

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go/ext"
	prometheus "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sourcegraph/sourcegraph/internal/prometheusutil"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
)

// MonitoringAlert implements the GraphQL type MonitoringAlert.
type MonitoringAlert struct {
	TimestampValue   DateTime
	NameValue        string
	ServiceNameValue string
	OwnerValue       string
	AverageValue     float64
}

func (r *MonitoringAlert) Timestamp() DateTime { return r.TimestampValue }
func (r *MonitoringAlert) Name() string        { return r.NameValue }
func (r *MonitoringAlert) ServiceName() string { return r.ServiceNameValue }
func (r *MonitoringAlert) Owner() string       { return r.OwnerValue }
func (r *MonitoringAlert) Average() float64    { return r.AverageValue }

type MonitoringAlerts []*MonitoringAlert

// Less determined by timestamp -> serviceName -> alert name
func (a MonitoringAlerts) Less(i, j int) bool {
	if a[i].Timestamp().Equal(a[j].Timestamp().Time) {
		if a[i].ServiceName() == a[j].ServiceName() {
			return a[i].Name() < a[j].Name()
		}
		return a[i].ServiceName() < a[j].ServiceName()
	}
	return a[i].Timestamp().Before(a[j].Timestamp().Time)
}
func (a MonitoringAlerts) Swap(i, j int) {
	tmp := a[i]
	a[i] = a[j]
	a[j] = tmp
}
func (a MonitoringAlerts) Len() int { return len(a) }

func (r *siteResolver) MonitoringStatistics(ctx context.Context, args *struct {
	Days *int32
}) (*siteMonitoringStatisticsResolver, error) {
	prom, err := prometheusutil.NewPrometheusQuerier()
	if err != nil {
		return nil, err
	}
	return &siteMonitoringStatisticsResolver{
		prom:     prom,
		timespan: time.Duration(*args.Days) * 24 * time.Hour,
	}, nil
}

type siteMonitoringStatisticsResolver struct {
	prom     prometheusutil.PrometheusQuerier
	timespan time.Duration
}

func (r *siteMonitoringStatisticsResolver) Alerts(ctx context.Context) ([]*MonitoringAlert, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	span, ctx := ot.StartSpanFromContext(ctx, "site.MonitoringStatistics.alerts")

	var err error
	defer func() {
		if err != nil {
			ext.Error.Set(span, true)
			span.SetTag("err", err.Error())
		}
		cancel()
		span.Finish()
	}()

	results, warn, err := r.prom.QueryRange(ctx, `max by (level,name,service_name,owner)(avg_over_time(alert_count{name!=""}[12h]))`,
		prometheus.Range{
			Start: time.Now().Add(-r.timespan),
			End:   time.Now(),
			Step:  12 * time.Hour,
		})
	if err != nil {
		return nil, errors.Wrap(err, "prometheus query failed")
	}
	if len(warn) > 0 {
		log15.Warn("site.monitoring.alerts: warnings encountered on prometheus query",
			"timespan", r.timespan.String(),
			"warnings", warn)
	}
	if results.Type() != model.ValMatrix {
		return nil, fmt.Errorf("received unexpected result type %q from prometheus", results.Type())
	}

	data := results.(model.Matrix)
	var alerts MonitoringAlerts
	for _, sample := range data {
		var (
			name        = string(sample.Metric["name"])
			serviceName = string(sample.Metric["service_name"])
			level       = string(sample.Metric["level"])
			owner       = string(sample.Metric["owner"])
			prevVal     *model.SampleValue
		)
		for _, p := range sample.Values {
			// Check for nil so that we don't ignore the first occurrence of an alert - even if the
			// alert is never >0, we want to be aware that it is at least configured correctly and
			// being tracked. Otherwise, if the value in this window is the same as in the previous
			// window just discard it.
			if prevVal != nil && p.Value == *prevVal {
				continue
			}
			// copy value for comparison later
			v := p.Value
			prevVal = &v
			// record alert in results
			alerts = append(alerts, &MonitoringAlert{
				NameValue:        fmt.Sprintf("%s: %s", level, name),
				ServiceNameValue: serviceName,
				OwnerValue:       owner,
				TimestampValue:   DateTime{p.Timestamp.Time().UTC().Truncate(time.Hour)},
				AverageValue:     float64(p.Value),
			})
		}
	}

	sort.Sort(alerts)
	return alerts, err
}
