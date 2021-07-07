package server

import (
	"io"
	"os/exec"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/mxk/go-flowrate/flowrate"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/lib/gitservice"
)

// flowrateWriter limits the write rate of w.
//
// We are cloning repositories from within the same network from another
// Sourcegraph service (zoekt-indexserver). This can end up being so fast that
// we harm our own network connectivity. In the case of zoekt-indexserver and
// gitserver running on the same host machine, we can even reach up to ~100
// Gbps and effectively DoS the Docker network, temporarily disrupting other
// containers running on the host.
//
// Google Compute Engine has a network bandwidth of about 1.64 Gbps
// between nodes, and AWS varies widely depending on instance type.
// We play it safe and default to 1 Gbps today (~119 MiB/s), which
// means we can fetch a 1 GiB archive in ~8.5 seconds.
func flowrateWriter(w io.Writer) io.Writer {
	const megabit = int64(1000 * 1000)
	limit := int64(conf.Get().GitMaxFlowRateMbps) * megabit
	return flowrate.NewWriter(w, limit)
}

func (s *Server) gitServiceHandler() *gitservice.Handler {
	return &gitservice.Handler{
		Dir: func(d string) string {
			return string(s.dir(api.RepoName(d)))
		},

		// Limit rate of stdout from git.
		CommandHook: func(cmd *exec.Cmd) {
			cmd.Stdout = flowrateWriter(cmd.Stdout)
		},

		Trace: func(svc, repo, protocol string) func(error) {
			start := time.Now()
			metricServiceRunning.WithLabelValues(svc).Inc()
			return func(err error) {
				metricServiceRunning.WithLabelValues(svc).Dec()
				metricServiceDuration.WithLabelValues(svc).Observe(time.Since(start).Seconds())

				if err != nil {
					log15.Error("gitservice.ServeHTTP", "svc", svc, "repo", repo, "protocol", protocol, "duration", time.Since(start), "error", err.Error())
				} else if traceLogs {
					log15.Debug("TRACE gitserver git service", "svc", svc, "repo", repo, "protocol", protocol, "duration", time.Since(start))
				}
			}
		},
	}
}

var (
	metricServiceDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "src_gitserver_gitservice_duration_seconds",
		Help:    "A histogram of latencies for the git service (upload-pack for internal clones) endpoint.",
		Buckets: prometheus.ExponentialBuckets(.1, 5, 5), // 100ms -> 62s
	}, []string{"type"})

	metricServiceRunning = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "src_gitserver_gitservice_running",
		Help: "A histogram of latencies for the git service (upload-pack for internal clones) endpoint.",
	}, []string{"type"})
)
