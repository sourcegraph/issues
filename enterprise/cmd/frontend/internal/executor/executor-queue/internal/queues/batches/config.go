package batches

import (
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/config"
	"github.com/sourcegraph/sourcegraph/internal/env"
)

type Config struct {
	env.BaseConfig

	Shared *config.SharedConfig
}

func (c *Config) Load() {}
