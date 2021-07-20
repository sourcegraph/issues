package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana-tools/sdk"
	"github.com/inconshreveable/log15"
	"gopkg.in/yaml.v2"
)

const (
	localGrafanaURL         = "http://127.0.0.1:3370"
	localGrafanaCredentials = "admin:admin"

	localPrometheusURL = "http://127.0.0.1:9090"
)

// GenerateOptions declares options for the monitoring generator.
type GenerateOptions struct {
	// Toggles pruning of dangling generated assets through simple heuristic, should be disabled during builds
	DisablePrune bool
	// Trigger reload of active Prometheus or Grafana instance (requires respective output directories)
	Reload bool

	// Output directory for generated Grafana assets
	GrafanaDir string
	// Output directory for generated Prometheus assets
	PrometheusDir string
	// Output directory for generated documentation
	DocsDir string
}

// Generate is the main Sourcegraph monitoring generator entrypoint.
func Generate(logger log15.Logger, opts GenerateOptions, containers ...*Container) error {
	logger.Info("Regenerating monitoring", "options", opts, "containers", len(containers))

	var generatedAssets []string

	// Generate output for all containers
	for _, container := range containers {
		// Logger for container
		clog := logger.New("container", container.Name)

		// Verify container configuration
		if err := container.validate(); err != nil {
			clog.Crit("Failed to validate Container", "err", err)
			return err
		}

		// Prepare Grafana assets
		if opts.GrafanaDir != "" {
			clog.Debug("Rendering Grafana assets")
			board := container.renderDashboard()
			data, err := json.MarshalIndent(board, "", "  ")
			if err != nil {
				clog.Crit("Invalid dashboard", "err", err)
				return err
			}
			// #nosec G306  prometheus runs as nobody
			generatedDashboard := container.Name + ".json"
			err = os.WriteFile(filepath.Join(opts.GrafanaDir, generatedDashboard), data, os.ModePerm)
			if err != nil {
				clog.Crit("Could not write dashboard to output", "error", err)
				return err
			}
			generatedAssets = append(generatedAssets, generatedDashboard)

			// Reload specific dashboard
			if opts.Reload {
				crlog := clog.New("instance", localGrafanaURL)
				crlog.Debug("Reloading Grafana instance")
				client, err := sdk.NewClient(localGrafanaURL, localGrafanaCredentials, sdk.DefaultHTTPClient)
				if err != nil {
					crlog.Crit("Failed to initialize Grafana client", "error", err)
					return err
				}
				_, err = client.SetDashboard(context.Background(), *board, sdk.SetDashboardParams{Overwrite: true})
				if err != nil {
					crlog.Crit("Could not reload Grafana instance", "error", err)
					return err
				}
				crlog.Info("Reloaded Grafana instance")
			}
		}

		// Prepare Prometheus assets
		if opts.PrometheusDir != "" {
			clog.Debug("Rendering Prometheus assets")
			promAlertsFile, err := container.renderRules()
			if err != nil {
				clog.Crit("Unable to generate alerts", "err", err)
				return err
			}
			data, err := yaml.Marshal(promAlertsFile)
			if err != nil {
				clog.Crit("Invalid rules", "err", err)
				return err
			}
			fileName := strings.ReplaceAll(container.Name, "-", "_") + alertRulesFileSuffix
			generatedAssets = append(generatedAssets, fileName)
			err = os.WriteFile(filepath.Join(opts.PrometheusDir, fileName), data, os.ModePerm)
			if err != nil {
				clog.Crit("Could not write rules to output", "error", err)
				return err
			}
		}
	}

	// Reload all Prometheus rules
	if opts.PrometheusDir != "" && opts.Reload {
		rlog := logger.New("instance", localPrometheusURL)
		// Reload all Prometheus rules
		rlog.Debug("Reloading Prometheus instance", "instance", localPrometheusURL)
		resp, err := http.Post(localPrometheusURL+"/-/reload", "", nil)
		if err != nil {
			rlog.Crit("Could not reload Prometheus", "error", err)
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			rlog.Crit("Unexpected status code while reloading Prometheus rules", "code", resp.StatusCode)
			return err
		}
		rlog.Info("Reloaded Prometheus instance")
	}

	// Generate documentation
	if opts.DocsDir != "" {
		logger.Debug("Rendering docs")
		docs, err := renderDocumentation(containers)
		if err != nil {
			logger.Crit("Failed to generate docs", "error", err)
			return err
		}
		for _, docOut := range []struct {
			path string
			data []byte
		}{
			{path: filepath.Join(opts.DocsDir, alertSolutionsFile), data: docs.alertSolutions.Bytes()},
			{path: filepath.Join(opts.DocsDir, dashboardsDocsFile), data: docs.dashboards.Bytes()},
		} {
			err = os.WriteFile(docOut.path, docOut.data, os.ModePerm)
			if err != nil {
				logger.Crit("Could not write docs to path", "path", docOut.path, "error", err)
				return err
			}
			generatedAssets = append(generatedAssets, docOut.path)
		}
	}

	// Clean up dangling assets
	if !opts.DisablePrune {
		logger.Debug("Pruning dangling assets")
		if err := pruneAssets(logger, generatedAssets, opts.GrafanaDir, opts.PrometheusDir); err != nil {
			logger.Crit("Failed to prune assets, resolve manually or disable pruning", "error", err)
			return err
		}
	}

	return nil
}
