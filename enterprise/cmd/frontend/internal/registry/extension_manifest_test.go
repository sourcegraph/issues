package registry

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
)

func TestGetExtensionManifestWithBundleURL(t *testing.T) {
	resetMocks()
	ctx := context.Background()
	t0 := time.Unix(1234, 0)

	t.Run(`manifest with "url"`, func(t *testing.T) {
		mocks.releases.GetLatest = func(registryExtensionID int32, releaseTag string, includeArtifacts bool) (*dbRelease, error) {
			return &dbRelease{
				Manifest:  `{"name":"x","url":"u"}`,
				CreatedAt: t0,
			}, nil
		}
		defer func() { mocks.releases.GetLatest = nil }()
		release, err := getLatestRelease(ctx, &dbtesting.MockDB{}, "x", 1, "t")
		if err != nil {
			t.Fatal(err)
		}
		if want := `{"name":"x","url":"u"}`; !jsonDeepEqual(release.Manifest, want) {
			t.Errorf("got %q, want %q", release.Manifest, want)
		}
		if release.CreatedAt != t0 {
			t.Errorf("got %v, want %v", release.CreatedAt, t0)
		}
	})

	t.Run(`manifest without "url"`, func(t *testing.T) {
		mocks.releases.GetLatest = func(registryExtensionID int32, releaseTag string, includeArtifacts bool) (*dbRelease, error) {
			return &dbRelease{
				Manifest:  `{"name":"x"}`,
				CreatedAt: t0,
			}, nil
		}
		defer func() { mocks.releases.GetLatest = nil }()
		release, err := getLatestRelease(ctx, &dbtesting.MockDB{}, "x", 1, "t")
		if err != nil {
			t.Fatal(err)
		}
		if want := `{"name":"x","url":"/-/static/extension/0-x.js?fqw3qlts--x"}`; !jsonDeepEqual(release.Manifest, want) {
			t.Errorf("got %q, want %q", release.Manifest, want)
		}
		if release.CreatedAt != t0 {
			t.Errorf("got %v, want %v", release.CreatedAt, t0)
		}
	})
}

func jsonDeepEqual(a, b string) bool {
	var va, vb interface{}
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		panic(err)
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		panic(err)
	}
	return reflect.DeepEqual(va, vb)
}
