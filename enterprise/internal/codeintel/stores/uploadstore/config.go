package uploadstore

import (
	"strings"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/env"
)

type Config struct {
	env.BaseConfig

	Backend      string
	ManageBucket bool
	Bucket       string
	TTL          time.Duration
	S3           S3Config
	GCS          GCSConfig
}

type loader interface {
	load(parent *env.BaseConfig)
}

func (c *Config) Load() {
	c.Backend = strings.ToLower(c.Get("PRECISE_CODE_INTEL_UPLOAD_BACKEND", "MinIO", "The target file service for code intelligence uploads. S3, GCS, and MinIO are supported."))
	c.ManageBucket = c.GetBool("PRECISE_CODE_INTEL_UPLOAD_MANAGE_BUCKET", "false", "Whether or not the client should manage the target bucket configuration.")
	c.Bucket = c.Get("PRECISE_CODE_INTEL_UPLOAD_BUCKET", "lsif-uploads", "The name of the bucket to store LSIF uploads in.")
	c.TTL = c.GetInterval("PRECISE_CODE_INTEL_UPLOAD_TTL", "168h", "The maximum age of an upload before deletion.")

	if c.Backend == "minio" {
		// No manual provisioning
		c.ManageBucket = true
	}

	loaders := map[string]loader{
		"s3":    &c.S3,
		"minio": &c.S3,
		"gcs":   &c.GCS,
	}

	config, ok := loaders[c.Backend]
	if !ok {
		c.AddError(errors.Errorf("invalid backend %q for PRECISE_CODE_INTEL_UPLOAD_BACKEND: must be S3, GCS, or MinIO", c.Backend))
		return
	}

	config.load(&c.BaseConfig)
}
