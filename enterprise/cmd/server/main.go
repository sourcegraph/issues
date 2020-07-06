package main

import (
	"path/filepath"

	"github.com/sourcegraph/sourcegraph/cmd/server/shared"
)

func main() {
	shared.ProcfileAdditions = append(
		shared.ProcfileAdditions,
		`precise-code-intel-bundle-manager: precise-code-intel-bundle-manager`,
		`precise-code-intel-worker: precise-code-intel-worker`,
	)

	shared.SrcProfServices = append(
		shared.SrcProfServices,
		map[string]string{"Name": "precise-code-intel-bundle-manager", "Host": "127.0.0.1:6087"},
		map[string]string{"Name": "precise-code-intel-worker", "Host": "127.0.0.1:6088"},
	)

	shared.DefaultEnv["PRECISE_CODE_INTEL_BUNDLE_DIR"] = filepath.Join(shared.DataDir, "lsif-storage")
	shared.DefaultEnv["PRECISE_CODE_INTEL_BUNDLE_MANAGER_URL"] = "http://127.0.0.1:3187"

	shared.Main()
}
