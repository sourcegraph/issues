package main

import (
	"time"

	apiserver "github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/server"
	"github.com/sourcegraph/sourcegraph/internal/env"
)

type Config struct {
	env.BaseConfig

	Port                   int
	MaximumNumTransactions int
	JobRequeueDelay        time.Duration
}

func (c *Config) Load() {
	c.Port = c.GetInt("EXECUTOR_QUEUE_API_PORT", "3191", "The port to listen on.")
	c.JobRequeueDelay = c.GetInterval("EXECUTOR_QUEUE_JOB_REQUEUE_DELAY", "1m", "The requeue delay of jobs assigned to an unreachable executor.")
}

func (c *Config) ServerOptions() apiserver.Options {
	return apiserver.Options{
		Port:         c.Port,
		RequeueDelay: c.JobRequeueDelay,
	}
}
