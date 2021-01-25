package shared

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/inconshreveable/log15"
)

func maybeMinio() []string {
	if os.Getenv("DISABLE_MINIO") != "" {
		log15.Info("WARNING: Running with minio disabled")
		return []string{}
	}

	// Set default for MinIO auth and point at local MinIO endpoint
	// All other variables will default to contacting a MinIO instance
	// with our default credentials running in a sibling container.
	SetDefaultEnv("MINIO_ACCESS_KEY", "AKIAIOSFODNN7EXAMPLE")
	SetDefaultEnv("MINIO_SECRET_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	SetDefaultEnv("PRECISE_CODE_INTEL_UPLOAD_AWS_ENDPOINT", "http://127.0.0.1:9000")

	// Configure MinIO service
	dataDir := filepath.Join(os.Getenv("DATA_DIR"), "minio")
	procline := fmt.Sprintf(`minio: /usr/local/bin/minio server %s >> /var/opt/sourcegraph/minio.log 2>&1`, dataDir)
	return []string{procline}
}
