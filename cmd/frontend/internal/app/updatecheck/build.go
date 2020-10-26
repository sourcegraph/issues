package updatecheck

import "github.com/coreos/go-semver/semver"

// build is the JSON shape of the update check handler's response body.
type build struct {
	Version semver.Version `json:"version"`
}

func newBuild(version string) build {
	return build{
		Version: *semver.New(version),
	}
}
