package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/inconshreveable/log15"
)

func runCmd(log log15.Logger, errs chan<- error, cmd *exec.Cmd) {
	log.Info(fmt.Sprintf("running: %+v", cmd.Args))
	if err := cmd.Run(); err != nil {
		err := fmt.Errorf("command %+v exited: %w", cmd.Args, err)
		log.Error(err.Error())
		errs <- err
	}
}

// NewPrometheusCmd instantiates a new command to run Prometheus.
// Parameter promArgs replicates "$@"
func NewPrometheusCmd(promArgs []string, promPort string) *exec.Cmd {
	promFlags := []string{
		fmt.Sprintf("--web.listen-address=0.0.0.0:%s", promPort),
	}
	cmd := exec.Command("/prometheus.sh", append(promFlags, promArgs...)...)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd
}

func NewAlertmanagerCmd(configPath string) *exec.Cmd {
	cmd := exec.Command("/alertmanager.sh",
		fmt.Sprintf("--config.file=%s", configPath),
		fmt.Sprintf("--web.route-prefix=/%s", alertmanagerPathPrefix))
	// disable clustering unless otherwise configured - it is enabled by default, but
	// can cause alertmanager to fail to start up in some environments: https://github.com/sourcegraph/sourcegraph/issues/13079
	if alertmanagerEnableCluster != "true" {
		cmd.Args = append(cmd.Args, "--cluster.listen-address=")
	}
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd
}
