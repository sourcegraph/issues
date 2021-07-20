package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/cockroachdb/errors"
	amclient "github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/general"
	amconfig "github.com/prometheus/alertmanager/config"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v2"
)

// Prefix to serve alertmanager on. If you change this, make sure you update prometheus.yml as well
const alertmanagerPathPrefix = "alertmanager"

func waitForAlertmanager(ctx context.Context, alertmanager *amclient.Alertmanager) error {
	ping := func(ctx context.Context) error {
		resp, err := alertmanager.General.GetStatus(&general.GetStatusParams{Context: ctx})
		if err != nil {
			return err
		}
		if resp.Payload == nil || resp.Payload.Config == nil {
			return errors.Errorf("ping: malformed health response: %+v", resp)
		}
		return nil
	}

	var lastErr error
	for {
		err := ping(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return errors.Errorf("alertmanager not reachable: %s (last error: %v)", err, lastErr)
			}

			// Keep trying.
			lastErr = err
			time.Sleep(250 * time.Millisecond)
			continue
		}
		break
	}
	return nil
}

// reloadAlertmanager triggers a realod of the Alertmanager configuration file, because package alertmanager/api/v2 does not have a wrapper for reload
// See https://prometheus.io/docs/alerting/latest/management_api/#reload
func reloadAlertmanager(ctx context.Context) error {
	reloadReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%s/%s/-/reload", alertmanagerPort, alertmanagerPathPrefix), nil)
	if err != nil {
		return errors.Errorf("failed to create reload request: %w", err)
	}
	resp, err := http.DefaultClient.Do(reloadReq.WithContext(ctx))
	if err != nil {
		return errors.Errorf("reload request failed: %w", err)
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Errorf("reload failed with status %d", resp.StatusCode)
		}
		return errors.Errorf("reload failed with status %d: %s", resp.StatusCode, string(data))
	}
	return nil
}

// renderConfiguration marshals the given Alertmanager configuration to a format accepted
// by Alertmanager, and also validates that it will be accepted by Alertmanager.
func renderConfiguration(cfg *amconfig.Config) ([]byte, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, errors.Errorf("failed to marshal: %w", err)
	}
	_, err = amconfig.Load(string(data))
	return data, err
}

// applyConfiguration writes validates and writes the given Alertmanager configuration
// to disk, and triggers a reload.
func applyConfiguration(ctx context.Context, cfg *amconfig.Config) error {
	amConfigData, err := renderConfiguration(cfg)
	if err != nil {
		return errors.Errorf("failed to generate Alertmanager configuration: %w", err)
	}
	if err := os.WriteFile(alertmanagerConfigPath, amConfigData, os.ModePerm); err != nil {
		return errors.Errorf("failed to write Alertmanager configuration: %w", err)
	}
	if err := reloadAlertmanager(ctx); err != nil {
		return errors.Errorf("failed to reload Alertmanager configuration: %w", err)
	}
	return nil
}

func duration(dur time.Duration) *model.Duration {
	d := model.Duration(dur)
	return &d
}
