package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	amclient "github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/general"
	"github.com/prometheus/common/model"
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
			return fmt.Errorf("ping: malformed health response: %+v", resp)
		}
		return nil
	}

	var lastErr error
	for {
		err := ping(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("alertmanager not reachable: %s (last error: %v)", err, lastErr)
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
		return fmt.Errorf("failed to create reload request: %w", err)
	}
	resp, err := http.DefaultClient.Do(reloadReq.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("reload request failed: %w", err)
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reload failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("reload failed with status %d: %s", resp.StatusCode, string(data))
	}
	return nil
}

func duration(dur time.Duration) *model.Duration {
	d := model.Duration(dur)
	return &d
}
