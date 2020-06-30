package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/inconshreveable/log15"
	amclient "github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/general"
	amconfig "github.com/prometheus/alertmanager/config"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/schema"
	"gopkg.in/yaml.v2"
)

type siteEmailConfig struct {
	SMTP    *schema.SMTPServerConfig
	Address string
}

// subscribedSiteConfig contains fields from SiteConfiguration relevant to the siteConfigSubscriber.
type subscribedSiteConfig struct {
	Alerts    []*schema.ObservabilityAlerts
	alertsSum [32]byte

	Email    *siteEmailConfig
	emailSum [32]byte
}

// newSubscribedSiteConfig creates a subscribedSiteConfig with sha256 sums calculated.
func newSubscribedSiteConfig(config schema.SiteConfiguration) *subscribedSiteConfig {
	alertsBytes, err := json.Marshal(config.ObservabilityAlerts)
	if err != nil {
		return nil
	}
	email := &siteEmailConfig{config.EmailSmtp, config.EmailAddress}
	emailBytes, err := json.Marshal(email)
	if err != nil {
		return nil
	}
	return &subscribedSiteConfig{
		Alerts:    config.ObservabilityAlerts,
		alertsSum: sha256.Sum256(alertsBytes),

		Email:    email,
		emailSum: sha256.Sum256(emailBytes),
	}
}

type siteConfigDiff struct {
	Type   string
	Change Change
}

// Diff returns a set of changes to apply.
func (c *subscribedSiteConfig) Diff(other *subscribedSiteConfig) []siteConfigDiff {
	var changes []siteConfigDiff

	if !bytes.Equal(c.alertsSum[:], other.alertsSum[:]) {
		changes = append(changes, siteConfigDiff{Type: "alerts", Change: changeReceivers})
	}

	if !bytes.Equal(c.emailSum[:], other.emailSum[:]) {
		changes = append(changes, siteConfigDiff{Type: "email", Change: changeSMTP})
	}

	return changes
}

// SiteConfigSubscriber is a sidecar service that subscribes to Sourcegraph site configuration and
// applies relevant (subscribedSiteConfig) changes to Grafana.
type SiteConfigSubscriber struct {
	log          log15.Logger
	alertmanager *amclient.Alertmanager

	mux      sync.RWMutex
	config   *subscribedSiteConfig
	problems conf.Problems // exported by handler
}

func NewSiteConfigSubscriber(ctx context.Context, logger log15.Logger, alertmanager *amclient.Alertmanager) (*SiteConfigSubscriber, error) {
	log := logger.New("logger", "config-subscriber")

	log.Info("waiting for alertmanager")
	if err := waitForAlertmanager(ctx, alertmanager); err != nil {
		return nil, err
	}
	log.Debug("detected alertmanager ready")

	zeroConfig := newSubscribedSiteConfig(schema.SiteConfiguration{})
	return &SiteConfigSubscriber{
		log:          log,
		alertmanager: alertmanager,
		config:       zeroConfig,
	}, nil
}

func (c *SiteConfigSubscriber) Handler() http.Handler {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		c.mux.RLock()
		defer c.mux.RUnlock()

		problems := c.problems

		if _, err := c.alertmanager.General.GetStatus(&general.GetStatusParams{
			Context: req.Context(),
		}); err != nil {
			c.log.Error("unable to get Alertmanager status", "error", err)
			problems = append(problems,
				conf.NewSiteProblem("`observability`: unable to reach Alertmanager - please refer to the Prometheus logs for more details"))
		}

		b, err := json.Marshal(map[string]interface{}{
			"problems": problems,
		})
		if err != nil {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		_, _ = w.Write(b)
	})
	return handler
}

func (c *SiteConfigSubscriber) Subscribe(ctx context.Context) {
	// Syncing relies on access to frontend, so wait until it is ready before subscribing.
	// At this point, everything else should have started as normal, so it's safe to block
	// here for however long is needed.
	c.log.Info("waiting for frontend", "url", api.InternalClient.URL)
	if err := api.InternalClient.WaitForFrontend(ctx); err != nil {
		c.log.Error("unable to connect to frontend, proceeding with existing configuration",
			"error", err)
	} else {
		c.log.Debug("detected frontend ready, loading initial configuration")

		// Load initial alerts configuration
		siteConfig := newSubscribedSiteConfig(conf.Get().SiteConfiguration)
		diffs := siteConfig.Diff(c.config)
		if len(diffs) > 0 {
			c.execDiffs(ctx, siteConfig, diffs)
		} else {
			c.log.Debug("no relevant configuration to init")
		}
	}

	// Watch for future changes
	conf.Watch(func() {
		c.mux.RLock()
		newSiteConfig := newSubscribedSiteConfig(conf.Get().SiteConfiguration)
		diffs := newSiteConfig.Diff(c.config)
		c.mux.RUnlock()

		// ignore irrelevant changes
		if len(diffs) == 0 {
			c.log.Debug("config update contained no relevant changes - ignoring")
			return
		}

		// update configuration
		configUpdateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		c.execDiffs(configUpdateCtx, newSiteConfig, diffs)
		cancel()
	})
}

// execDiffs updates grafanaAlertsSubscriber state and writes it to disk. It never returns an error,
// instead all errors are reported as problems
func (c *SiteConfigSubscriber) execDiffs(ctx context.Context, newConfig *subscribedSiteConfig, diffs []siteConfigDiff) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.log.Debug("applying configuration diffs", "diffs", diffs)
	c.problems = nil

	amConfig, err := amconfig.LoadFile(alertmanagerConfigPath)
	if err != nil {
		c.log.Error("failed to load Alertmanager configuration", "error", err)
		c.problems = append(c.problems, conf.NewSiteProblem("`observability`: failed to load Alertmanager configuration, please refer to Prometheus logs for more details"))
		return
	}

	// run changeset and aggregate results
	changeContext := ChangeContext{
		AMConfig: amConfig,
	}
	for _, diff := range diffs {
		c.log.Info(fmt.Sprintf("applying changes for %q diff", diff.Type))
		result := diff.Change(ctx, c.log, changeContext, newConfig)
		c.problems = append(c.problems, result.Problems...)
	}

	// persist configuration to disk
	c.log.Debug("reloading with new configuration", "change_context", changeContext) // *amconfig.Config automatically removes secrets
	updateProblem := conf.NewSiteProblem("`observability`: failed to update Alertmanager configuration, please refer to Prometheus logs for more details")
	amConfigData, err := yaml.Marshal(amConfig)
	if err != nil {
		c.log.Error("failed to generate Alertmanager configuration", "error", err)
		c.problems = append(c.problems, updateProblem)
		return
	}
	if err := ioutil.WriteFile(alertmanagerConfigPath, amConfigData, os.ModePerm); err != nil {
		c.log.Error("failed to write Alertmanager configuration", "error", err)
		c.problems = append(c.problems, updateProblem)
		return
	}
	if err := reloadAlertmanager(ctx); err != nil {
		c.log.Error("failed to reload Alertmanager configuration", "error", err)
		// this error can include useful information relevant to configuration, so include it in problem
		c.problems = append(c.problems, conf.NewSiteProblem(fmt.Sprintf("`observability`: failed to update Alertmanager configuration: %v", err)))
	}

	// update state
	c.config = newConfig
	c.log.Debug("configuration diffs applied", "diffs", diffs)
}
