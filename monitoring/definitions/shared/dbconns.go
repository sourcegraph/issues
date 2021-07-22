package shared

import (
	"fmt"
	"time"

	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

// Database connections monitoring overview.
const TitleDatabaseConnectionsMonitoring = "Database connections"

func DatabaseConnectionsMonitoring(app string) []monitoring.Row {
	return []monitoring.Row{
		{
			{
				Name:           "max_open_conns",
				Description:    "maximum open",
				Query:          fmt.Sprintf(`sum by (app_name, db_name) (src_pgsql_conns_max_open{app_name=%q})`, app),
				Panel:          monitoring.Panel().LegendFormat("dbname={{db_name}}"),
				NoAlert:        true,
				Owner:          monitoring.ObservableOwnerCoreApplication,
				Interpretation: "none",
			},
			{
				Name:           "open_conns",
				Description:    "established",
				Query:          fmt.Sprintf(`sum by (app_name, db_name) (src_pgsql_conns_open{app_name=%q})`, app),
				Panel:          monitoring.Panel().LegendFormat("dbname={{db_name}}"),
				NoAlert:        true,
				Owner:          monitoring.ObservableOwnerCoreApplication,
				Interpretation: "none",
			},
		},
		{
			{
				Name:           "in_use",
				Description:    "used",
				Query:          fmt.Sprintf(`sum by (app_name, db_name) (src_pgsql_conns_in_use{app_name=%q})`, app),
				Panel:          monitoring.Panel().LegendFormat("dbname={{db_name}}"),
				NoAlert:        true,
				Owner:          monitoring.ObservableOwnerCoreApplication,
				Interpretation: "none",
			},
			{
				Name:           "idle",
				Description:    "idle",
				Query:          fmt.Sprintf(`sum by (app_name, db_name) (src_pgsql_conns_idle{app_name=%q})`, app),
				Panel:          monitoring.Panel().LegendFormat("dbname={{db_name}}"),
				NoAlert:        true,
				Owner:          monitoring.ObservableOwnerCoreApplication,
				Interpretation: "none",
			},
		},
		{
			{
				// The stats produced by the database/sql package don't allow us to maintain a histogram of blocked
				// durations. The best we can do with two ever increasing counters is an average / mean, which alright
				// to detect trends, although it doesn't give us a good sense of outliers (which we'd want to use high
				// percentiles for).
				Name:        "mean_blocked_seconds_per_conn_request",
				Description: "mean blocked seconds per conn request",
				Query: fmt.Sprintf(`sum by (app_name, db_name) (increase(src_pgsql_conns_blocked_seconds{app_name=%q}[5m])) / `+
					`sum by (app_name, db_name) (increase(src_pgsql_conns_waited_for{app_name=%q}[5m]))`, app, app),
				Panel:    monitoring.Panel().LegendFormat("dbname={{db_name}}").Unit(monitoring.Seconds),
				Warning:  monitoring.Alert().GreaterOrEqual(0.05, nil).For(5 * time.Minute),
				Critical: monitoring.Alert().GreaterOrEqual(0.10, nil).For(10 * time.Minute),
				Owner:    monitoring.ObservableOwnerCoreApplication,
				PossibleSolutions: `
					- Increase SRC_PGSQL_MAX_OPEN together with giving more memory to the database if needed
					- Scale up Postgres memory / cpus [See our scaling guide](https://docs.sourcegraph.com/admin/config/postgres-conf)
				`,
			},
		},
		{
			{
				Name:           "closed_max_idle",
				Description:    "closed by SetMaxIdleConns",
				Query:          fmt.Sprintf(`sum by (app_name, db_name) (increase(src_pgsql_conns_closed_max_idle{app_name=%q}[5m]))`, app),
				Panel:          monitoring.Panel().LegendFormat("dbname={{db_name}}"),
				NoAlert:        true,
				Owner:          monitoring.ObservableOwnerCoreApplication,
				Interpretation: "none",
			},
			{
				Name:           "closed_max_lifetime",
				Description:    "closed by SetConnMaxLifetime",
				Query:          fmt.Sprintf(`sum by (app_name, db_name) (increase(src_pgsql_conns_closed_max_lifetime{app_name=%q}[5m]))`, app),
				Panel:          monitoring.Panel().LegendFormat("dbname={{db_name}}"),
				NoAlert:        true,
				Owner:          monitoring.ObservableOwnerCoreApplication,
				Interpretation: "none",
			},
			{
				Name:           "closed_max_idle_time",
				Description:    "closed by SetConnMaxIdleTime",
				Query:          fmt.Sprintf(`sum by (app_name, db_name) (increase(src_pgsql_conns_closed_max_idle_time{app_name=%q}[5m]))`, app),
				Panel:          monitoring.Panel().LegendFormat("dbname={{db_name}}"),
				NoAlert:        true,
				Owner:          monitoring.ObservableOwnerCoreApplication,
				Interpretation: "none",
			},
		},
	}
}

// NewDatabaseConnectionsMonitoringGroup creates a group containing panels displaying
// database monitoring metrics for the given container.
func NewDatabaseConnectionsMonitoringGroup(containerName string) monitoring.Group {
	return monitoring.Group{
		Title:  TitleDatabaseConnectionsMonitoring,
		Hidden: true,
		Rows:   DatabaseConnectionsMonitoring(containerName),
	}
}
