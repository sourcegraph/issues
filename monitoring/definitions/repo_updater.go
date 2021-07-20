package definitions

import (
	"fmt"
	"time"

	"github.com/sourcegraph/sourcegraph/monitoring/definitions/shared"
	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

func RepoUpdater() *monitoring.Container {
	// This is set a bit longer than maxSyncInterval in internal/repos/syncer.go
	const syncDurationThreshold = 9 * time.Hour

	return &monitoring.Container{
		Name:        "repo-updater",
		Title:       "Repo Updater",
		Description: "Manages interaction with code hosts, instructs Gitserver to update repositories.",
		Groups: []monitoring.Group{
			{
				Title: "General",
				Rows: []monitoring.Row{
					{
						shared.FrontendInternalAPIErrorResponses("repo-updater", monitoring.ObservableOwnerCoreApplication).Observable(),
					},
				},
			},
			{
				Title: "Repositories",
				Rows: []monitoring.Row{
					{
						{
							Name:        "syncer_sync_last_time",
							Description: "time since last sync",
							Query:       `max(timestamp(vector(time()))) - max(src_repoupdater_syncer_sync_last_time)`,
							NoAlert:     true,
							Panel:       monitoring.Panel().Unit(monitoring.Seconds),
							Owner:       monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
								A high value here indicates issues synchronizing repo metadata.
								If the value is persistently high, make sure all external services have valid tokens.
							`,
						},
						{
							Name:        "src_repoupdater_max_sync_backoff",
							Description: "time since oldest sync",
							Query:       `max(src_repoupdater_max_sync_backoff)`,
							Critical:    monitoring.Alert().GreaterOrEqual(syncDurationThreshold.Seconds(), nil).For(10 * time.Minute),
							Panel:       monitoring.Panel().Unit(monitoring.Seconds),
							Owner:       monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: fmt.Sprintf(`
								An alert here indicates that no code host connections have synced in at least %v. This indicates that there could be a configuration issue
								with your code hosts connections or networking issues affecting communication with your code hosts.
								- Check the code host status indicator (cloud icon in top right of Sourcegraph homepage) for errors.
								- Make sure external services do not have invalid tokens by navigating to them in the web UI and clicking save. If there are no errors, they are valid.
								- Check the repo-updater logs for errors about syncing.
								- Confirm that outbound network connections are allowed where repo-updater is deployed.
								- Check back in an hour to see if the issue has resolved itself.
							`, syncDurationThreshold),
						},
						{
							Name:        "src_repoupdater_syncer_sync_errors_total",
							Description: "site level external service sync error rate",
							Query:       `max by (family) (rate(src_repoupdater_syncer_sync_errors_total{owner!="user"}[5m]))`,
							Critical:    monitoring.Alert().Greater(0, nil).For(10 * time.Minute),
							Panel:       monitoring.Panel().Unit(monitoring.Number).With(monitoring.PanelOptions.ZeroIfNoData()),
							Owner:       monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `
								An alert here indicates errors syncing site level repo metadata with code hosts. This indicates that there could be a configuration issue
								with your code hosts connections or networking issues affecting communication with your code hosts.
								- Check the code host status indicator (cloud icon in top right of Sourcegraph homepage) for errors.
								- Make sure external services do not have invalid tokens by navigating to them in the web UI and clicking save. If there are no errors, they are valid.
								- Check the repo-updater logs for errors about syncing.
								- Confirm that outbound network connections are allowed where repo-updater is deployed.
								- Check back in an hour to see if the issue has resolved itself.
							`,
						},
					},
					{
						{
							Name:              "syncer_sync_start",
							Description:       "repo metadata sync was started",
							Query:             fmt.Sprintf(`max by (family) (rate(src_repoupdater_syncer_start_sync{family="Syncer.SyncExternalService"}[%s]))`, syncDurationThreshold.String()),
							Warning:           monitoring.Alert().LessOrEqual(0, nil).For(syncDurationThreshold),
							Panel:             monitoring.Panel().LegendFormat("Family: {{family}} Owner: {{owner}}").Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check repo-updater logs for errors.",
						},
						{
							Name:              "syncer_sync_duration",
							Description:       "95th repositories sync duration",
							Query:             `histogram_quantile(0.95, max by (le, family, success) (rate(src_repoupdater_syncer_sync_duration_seconds_bucket[1m])))`,
							Warning:           monitoring.Alert().GreaterOrEqual(30, nil).For(5 * time.Minute),
							Panel:             monitoring.Panel().LegendFormat("{{family}}-{{success}}").Unit(monitoring.Seconds),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check the network latency is reasonable (<50ms) between the Sourcegraph and the code host",
						},
						{
							Name:              "source_duration",
							Description:       "95th repositories source duration",
							Query:             `histogram_quantile(0.95, max by (le) (rate(src_repoupdater_source_duration_seconds_bucket[1m])))`,
							Warning:           monitoring.Alert().GreaterOrEqual(30, nil).For(5 * time.Minute),
							Panel:             monitoring.Panel().Unit(monitoring.Seconds),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check the network latency is reasonable (<50ms) between the Sourcegraph and the code host",
						},
					},
					{
						{
							Name:              "syncer_synced_repos",
							Description:       "repositories synced",
							Query:             `max by (state) (rate(src_repoupdater_syncer_synced_repos_total[1m]))`,
							Warning:           monitoring.Alert().LessOrEqual(0, monitoring.StringPtr("max")).For(syncDurationThreshold),
							Panel:             monitoring.Panel().LegendFormat("{{state}}").Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check network connectivity to code hosts",
						},
						{
							Name:              "sourced_repos",
							Description:       "repositories sourced",
							Query:             `max(rate(src_repoupdater_source_repos_total[1m]))`,
							Warning:           monitoring.Alert().LessOrEqual(0, nil).For(syncDurationThreshold),
							Panel:             monitoring.Panel().Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check network connectivity to code hosts",
						},
						{
							Name:        "user_added_repos",
							Description: "total number of user added repos",
							Query:       `max(src_repoupdater_user_repos_total)`,
							// 90% of our enforced limit
							Critical:          monitoring.Alert().GreaterOrEqual(200000*0.9, nil).For(5 * time.Minute),
							Panel:             monitoring.Panel().Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check for unusual spikes in user added repos. Each user is only allowed to add 2000",
						},
					},
					{
						{
							Name:              "purge_failed",
							Description:       "repositories purge failed",
							Query:             `max(rate(src_repoupdater_purge_failed[1m]))`,
							Warning:           monitoring.Alert().Greater(0, nil).For(5 * time.Minute),
							Panel:             monitoring.Panel().Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check repo-updater's connectivity with gitserver and gitserver logs",
						},
					},
					{
						{
							Name:              "sched_auto_fetch",
							Description:       "repositories scheduled due to hitting a deadline",
							Query:             `max(rate(src_repoupdater_sched_auto_fetch[1m]))`,
							Warning:           monitoring.Alert().LessOrEqual(0, nil).For(syncDurationThreshold),
							Panel:             monitoring.Panel().Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check repo-updater logs. This is expected to fire if there are no user added code hosts",
						},
						{
							Name:        "sched_manual_fetch",
							Description: "repositories scheduled due to user traffic",
							Query:       `max(rate(src_repoupdater_sched_manual_fetch[1m]))`,
							NoAlert:     true,
							Panel:       monitoring.Panel().Unit(monitoring.Number),
							Owner:       monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
								Check repo-updater logs if this value is persistently high.
								This does not indicate anything if there are no user added code hosts.
							`,
						},
					},
					{
						{
							Name:              "sched_known_repos",
							Description:       "repositories managed by the scheduler",
							Query:             `max(src_repoupdater_sched_known_repos)`,
							Warning:           monitoring.Alert().LessOrEqual(0, nil).For(10 * time.Minute),
							Panel:             monitoring.Panel().Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check repo-updater logs. This is expected to fire if there are no user added code hosts",
						},
						{
							Name:        "sched_update_queue_length",
							Description: "rate of growth of update queue length over 5 minutes",
							Query:       `max(deriv(src_repoupdater_sched_update_queue_length[5m]))`,
							// Alert if the derivative is positive for longer than 30 minutes
							Critical:          monitoring.Alert().Greater(0, nil).For(120 * time.Minute),
							Panel:             monitoring.Panel().Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check repo-updater logs for indications that the queue is not being processed. The queue length should trend downwards over time as items are sent to GitServer",
						},
						{
							Name:              "sched_loops",
							Description:       "scheduler loops",
							Query:             `max(rate(src_repoupdater_sched_loops[1m]))`,
							Warning:           monitoring.Alert().LessOrEqual(0, nil).For(syncDurationThreshold),
							Panel:             monitoring.Panel().Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check repo-updater logs for errors. This is expected to fire if there are no user added code hosts",
						},
					},
					{
						{
							Name:              "sched_error",
							Description:       "repositories schedule error rate",
							Query:             `max(rate(src_repoupdater_sched_error[1m]))`,
							Critical:          monitoring.Alert().GreaterOrEqual(1, nil).For(time.Minute),
							Panel:             monitoring.Panel().Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check repo-updater logs for errors",
						},
					},
				},
			},
			{
				Title:  "Permissions",
				Hidden: true,
				Rows: []monitoring.Row{
					{
						{
							Name:              "perms_syncer_perms",
							Description:       "time gap between least and most up to date permissions",
							Query:             `max by (type) (src_repoupdater_perms_syncer_perms_gap_seconds)`,
							Warning:           monitoring.Alert().GreaterOrEqual((3 * 24 * time.Hour).Seconds(), nil).For(5 * time.Minute), // 3 days
							Panel:             monitoring.Panel().LegendFormat("{{type}}").Unit(monitoring.Seconds),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Increase the API rate limit to [GitHub](https://docs.sourcegraph.com/admin/external_service/github#github-com-rate-limits), [GitLab](https://docs.sourcegraph.com/admin/external_service/gitlab#internal-rate-limits) or [Bitbucket Server](https://docs.sourcegraph.com/admin/external_service/bitbucket_server#internal-rate-limits).",
						},
						{
							Name:              "perms_syncer_stale_perms",
							Description:       "number of entities with stale permissions",
							Query:             `max by (type) (src_repoupdater_perms_syncer_stale_perms)`,
							Warning:           monitoring.Alert().GreaterOrEqual(100, nil).For(5 * time.Minute),
							Panel:             monitoring.Panel().LegendFormat("{{type}}").Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Increase the API rate limit to [GitHub](https://docs.sourcegraph.com/admin/external_service/github#github-com-rate-limits), [GitLab](https://docs.sourcegraph.com/admin/external_service/gitlab#internal-rate-limits) or [Bitbucket Server](https://docs.sourcegraph.com/admin/external_service/bitbucket_server#internal-rate-limits).",
						},
						{
							Name:        "perms_syncer_no_perms",
							Description: "number of entities with no permissions",
							Query:       `max by (type) (src_repoupdater_perms_syncer_no_perms)`,
							Warning:     monitoring.Alert().GreaterOrEqual(100, nil).For(5 * time.Minute),
							Panel:       monitoring.Panel().LegendFormat("{{type}}").Unit(monitoring.Number),
							Owner:       monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `
								- **Enabled permissions for the first time:** Wait for few minutes and see if the number goes down.
								- **Otherwise:** Increase the API rate limit to [GitHub](https://docs.sourcegraph.com/admin/external_service/github#github-com-rate-limits), [GitLab](https://docs.sourcegraph.com/admin/external_service/gitlab#internal-rate-limits) or [Bitbucket Server](https://docs.sourcegraph.com/admin/external_service/bitbucket_server#internal-rate-limits).
							`,
						},
					},
					{
						{
							Name:              "perms_syncer_sync_duration",
							Description:       "95th permissions sync duration",
							Query:             `histogram_quantile(0.95, max by (le, type) (rate(src_repoupdater_perms_syncer_sync_duration_seconds_bucket[1m])))`,
							Warning:           monitoring.Alert().GreaterOrEqual(30, nil).For(5 * time.Minute),
							Panel:             monitoring.Panel().LegendFormat("{{type}}").Unit(monitoring.Seconds),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check the network latency is reasonable (<50ms) between the Sourcegraph and the code host.",
						},
						{
							Name:        "perms_syncer_queue_size",
							Description: "permissions sync queued items",
							Query:       `max(src_repoupdater_perms_syncer_queue_size)`,
							Warning:     monitoring.Alert().GreaterOrEqual(100, nil).For(5 * time.Minute),
							Panel:       monitoring.Panel().Unit(monitoring.Number),
							Owner:       monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `
								- **Enabled permissions for the first time:** Wait for few minutes and see if the number goes down.
								- **Otherwise:** Increase the API rate limit to [GitHub](https://docs.sourcegraph.com/admin/external_service/github#github-com-rate-limits), [GitLab](https://docs.sourcegraph.com/admin/external_service/gitlab#internal-rate-limits) or [Bitbucket Server](https://docs.sourcegraph.com/admin/external_service/bitbucket_server#internal-rate-limits).
							`,
						},
					},
					{
						{
							Name:        "perms_syncer_sync_errors",
							Description: "permissions sync error rate",
							Query:       `max by (type) (ceil(rate(src_repoupdater_perms_syncer_sync_errors_total[1m])))`,
							Critical:    monitoring.Alert().GreaterOrEqual(1, nil).For(time.Minute),
							Panel:       monitoring.Panel().LegendFormat("{{type}}").Unit(monitoring.Number),
							Owner:       monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `
								- Check the network connectivity the Sourcegraph and the code host.
								- Check if API rate limit quota is exhausted on the code host.
							`,
						},
					},
				},
			},
			{
				Title:  "External services",
				Hidden: true,
				Rows: []monitoring.Row{
					{
						{
							Name:              "src_repoupdater_external_services_total",
							Description:       "the total number of external services",
							Query:             `max(src_repoupdater_external_services_total)`,
							Critical:          monitoring.Alert().GreaterOrEqual(20000, nil).For(1 * time.Hour),
							Panel:             monitoring.Panel().Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check for spikes in external services, could be abuse",
						},
						{
							Name:              "src_repoupdater_user_external_services_total",
							Description:       "the total number of user added external services",
							Query:             `max(src_repoupdater_user_external_services_total)`,
							Warning:           monitoring.Alert().GreaterOrEqual(20000, nil).For(1 * time.Hour),
							Panel:             monitoring.Panel().Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check for spikes in external services, could be abuse",
						},
					},
					{
						{
							Name:        "repoupdater_queued_sync_jobs_total",
							Description: "the total number of queued sync jobs",
							Query:       `max(src_repoupdater_queued_sync_jobs_total)`,
							Warning:     monitoring.Alert().GreaterOrEqual(100, nil).For(1 * time.Hour),
							Panel:       monitoring.Panel().Unit(monitoring.Number),
							Owner:       monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `
								- **Check if jobs are failing to sync:** "SELECT * FROM external_service_sync_jobs WHERE state = 'errored'";
								- **Increase the number of workers** using the 'repoConcurrentExternalServiceSyncers' site config.
							`,
						},
						{
							Name:              "repoupdater_completed_sync_jobs_total",
							Description:       "the total number of completed sync jobs",
							Query:             `max(src_repoupdater_completed_sync_jobs_total)`,
							Warning:           monitoring.Alert().GreaterOrEqual(100000, nil).For(1 * time.Hour),
							Panel:             monitoring.Panel().Unit(monitoring.Number),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check repo-updater logs. Jobs older than 1 day should have been removed.",
						},
						{
							Name:              "repoupdater_errored_sync_jobs_percentage",
							Description:       "the percentage of external services that have failed their most recent sync",
							Query:             `max(src_repoupdater_errored_sync_jobs_percentage)`,
							Warning:           monitoring.Alert().Greater(10, nil).For(1 * time.Hour),
							Panel:             monitoring.Panel().Unit(monitoring.Percentage),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "Check repo-updater logs. Check code host connectivity",
						},
					},
					{
						{
							Name:        "github_graphql_rate_limit_remaining",
							Description: "remaining calls to GitHub graphql API before hitting the rate limit",
							Query:       `max by (name) (src_github_rate_limit_remaining_v2{resource="graphql"})`,
							// 5% of initial limit of 5000
							Critical:          monitoring.Alert().LessOrEqual(250, nil),
							Panel:             monitoring.Panel().LegendFormat("{{name}}"),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `Try restarting the pod to get a different public IP.`,
						},
						{
							Name:        "github_rest_rate_limit_remaining",
							Description: "remaining calls to GitHub rest API before hitting the rate limit",
							Query:       `max by (name) (src_github_rate_limit_remaining_v2{resource="rest"})`,
							// 5% of initial limit of 5000
							Critical:          monitoring.Alert().LessOrEqual(250, nil),
							Panel:             monitoring.Panel().LegendFormat("{{name}}"),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `Try restarting the pod to get a different public IP.`,
						},
						{
							Name:              "github_search_rate_limit_remaining",
							Description:       "remaining calls to GitHub search API before hitting the rate limit",
							Query:             `max by (name) (src_github_rate_limit_remaining_v2{resource="search"})`,
							Critical:          monitoring.Alert().LessOrEqual(5, nil),
							Panel:             monitoring.Panel().LegendFormat("{{name}}"),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `Try restarting the pod to get a different public IP.`,
						},
					},
					{
						{
							Name:           "github_graphql_rate_limit_wait_duration",
							Description:    "time spent waiting for the GitHub graphql API rate limiter",
							Query:          `max by(name) (rate(src_github_rate_limit_wait_duration_seconds{resource="graphql"}[5m]))`,
							Panel:          monitoring.Panel().LegendFormat("{{name}}").Unit(monitoring.Seconds),
							Owner:          monitoring.ObservableOwnerCoreApplication,
							NoAlert:        true,
							Interpretation: "Indicates how long we're waiting on the rate limit once it has been exceeded",
						},
						{
							Name:           "github_rest_rate_limit_wait_duration",
							Description:    "time spent waiting for the GitHub rest API rate limiter",
							Query:          `max by(name) (rate(src_github_rate_limit_wait_duration_seconds{resource="rest"}[5m]))`,
							Panel:          monitoring.Panel().LegendFormat("{{name}}").Unit(monitoring.Seconds),
							Owner:          monitoring.ObservableOwnerCoreApplication,
							NoAlert:        true,
							Interpretation: "Indicates how long we're waiting on the rate limit once it has been exceeded",
						},
						{
							Name:           "github_search_rate_limit_wait_duration",
							Description:    "time spent waiting for the GitHub search API rate limiter",
							Query:          `max by(name) (rate(src_github_rate_limit_wait_duration_seconds{resource="search"}[5m]))`,
							Panel:          monitoring.Panel().LegendFormat("{{name}}").Unit(monitoring.Seconds),
							Owner:          monitoring.ObservableOwnerCoreApplication,
							NoAlert:        true,
							Interpretation: "Indicates how long we're waiting on the rate limit once it has been exceeded",
						},
					},
					{
						{
							Name:        "gitlab_rest_rate_limit_remaining",
							Description: "remaining calls to GitLab rest API before hitting the rate limit",
							Query:       `max by (name) (src_gitlab_rate_limit_remaining{resource="rest"})`,
							// 5% of initial limit of 600
							Critical:          monitoring.Alert().LessOrEqual(30, nil),
							Panel:             monitoring.Panel().LegendFormat("{{name}}"),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `Try restarting the pod to get a different public IP.`,
						},
						{
							Name:           "gitlab_rest_rate_limit_wait_duration",
							Description:    "time spent waiting for the GitLab rest API rate limiter",
							Query:          `max by(name) (rate(src_gitlab_rate_limit_wait_duration_seconds{resource="rest"}[5m]))`,
							Panel:          monitoring.Panel().LegendFormat("{{name}}").Unit(monitoring.Seconds),
							Owner:          monitoring.ObservableOwnerCoreApplication,
							NoAlert:        true,
							Interpretation: "Indicates how long we're waiting on the rate limit once it has been exceeded",
						},
					},
				},
			},
			{
				Title:  shared.TitleDatabaseConnectionsMonitoring,
				Hidden: true,
				Rows:   shared.DatabaseConnectionsMonitoring("repo-updater"),
			},
			{
				Title:  shared.TitleContainerMonitoring,
				Hidden: true,
				Rows: []monitoring.Row{
					{
						shared.ContainerCPUUsage("repo-updater", monitoring.ObservableOwnerCoreApplication).Observable(),
						shared.ContainerMemoryUsage("repo-updater", monitoring.ObservableOwnerCoreApplication).
							WithWarning(nil).
							WithCritical(monitoring.Alert().GreaterOrEqual(90, nil).For(10 * time.Minute)).
							Observable(),
					},
					{
						shared.ContainerMissing("repo-updater", monitoring.ObservableOwnerCoreApplication).Observable(),
					},
				},
			},
			{
				Title:  shared.TitleProvisioningIndicators,
				Hidden: true,
				Rows: []monitoring.Row{
					{
						shared.ProvisioningCPUUsageLongTerm("repo-updater", monitoring.ObservableOwnerCoreApplication).Observable(),
						shared.ProvisioningMemoryUsageLongTerm("repo-updater", monitoring.ObservableOwnerCoreApplication).Observable(),
					},
					{
						shared.ProvisioningCPUUsageShortTerm("repo-updater", monitoring.ObservableOwnerCoreApplication).Observable(),
						shared.ProvisioningMemoryUsageShortTerm("repo-updater", monitoring.ObservableOwnerCoreApplication).Observable(),
					},
				},
			},
			{
				Title:  shared.TitleGolangMonitoring,
				Hidden: true,
				Rows: []monitoring.Row{
					{
						shared.GoGoroutines("repo-updater", monitoring.ObservableOwnerCoreApplication).Observable(),
						shared.GoGcDuration("repo-updater", monitoring.ObservableOwnerCoreApplication).Observable(),
					},
				},
			},
			{
				Title:  shared.TitleKubernetesMonitoring,
				Hidden: true,
				Rows: []monitoring.Row{
					{
						shared.KubernetesPodsAvailable("repo-updater", monitoring.ObservableOwnerCoreApplication).Observable(),
					},
				},
			},
		},
	}
}
