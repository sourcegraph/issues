package main

import (
	"time"

	apiserver "github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/server"
	"github.com/sourcegraph/sourcegraph/internal/env"
)

type Config struct {
	env.BaseConfig

	Port                       int
	MaximumNumTransactions     int
	JobRequeueDelay            time.Duration
	JobCleanupInterval         time.Duration
	MaximumNumMissedHeartbeats int
}

func (c *Config) Load() {
	c.Port = c.GetInt("EXECUTOR_QUEUE_API_PORT", "3191", "The port to listen on.")
	c.MaximumNumTransactions = c.GetInt("EXECUTOR_QUEUE_MAXIMUM_NUM_TRANSACTIONS", "10", "Number of jobs that can be processing at one time.")
	c.JobRequeueDelay = c.GetInterval("EXECUTOR_QUEUE_JOB_REQUEUE_DELAY", "1m", "The requeue delay of jobs assigned to an unreachable executor.")
	c.JobCleanupInterval = c.GetInterval("EXECUTOR_QUEUE_JOB_CLEANUP_INTERVAL", "10s", "Interval between cleanup runs.")
	c.MaximumNumMissedHeartbeats = c.GetInt("EXECUTOR_QUEUE_MAXIMUM_NUM_MISSED_HEARTBEATS", "5", "The number of heartbeats an executor must miss to be considered unreachable.")
}

func (c *Config) ServerOptions(queueOptions map[string]apiserver.QueueOptions) apiserver.Options {
	return apiserver.Options{
		Port:                   c.Port,
		QueueOptions:           queueOptions,
		MaximumNumTransactions: c.MaximumNumTransactions,
		RequeueDelay:           c.JobRequeueDelay,
		UnreportedMaxAge:       c.JobCleanupInterval * time.Duration(c.MaximumNumMissedHeartbeats),
		DeathThreshold:         c.JobCleanupInterval * time.Duration(c.MaximumNumMissedHeartbeats),
		CleanupInterval:        c.JobCleanupInterval,
	}
}
