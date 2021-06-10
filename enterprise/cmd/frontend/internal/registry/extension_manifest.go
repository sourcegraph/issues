package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strconv"

	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
)

// validateExtensionManifest validates a JSON extension manifest for syntax.
//
// TODO(sqs): Also validate it against the JSON Schema.
func validateExtensionManifest(text string) error {
	var o interface{}
	return jsonc.Unmarshal(text, &o)
}

// getLatestRelease returns the release with the extension manifest as JSON. If there are no
// releases, it returns a nil manifest. If the manifest has no "url" field itself, a "url" field
// pointing to the extension's bundle is inserted. It also returns the date that the release was
// published.
func getLatestRelease(ctx context.Context, db dbutil.DB, extensionID string, registryExtensionID int32, releaseTag string) (*dbRelease, error) {
	release, err := dbReleases{db: db}.GetLatest(ctx, registryExtensionID, releaseTag, false)
	if err != nil && !errcode.IsNotFound(err) {
		return nil, err
	}
	if release == nil {
		return nil, nil
	}

	if err := prepReleaseManifest(extensionID, release); err != nil {
		return nil, fmt.Errorf("parsing extension manifest for extension with ID %d (release tag %q): %s", registryExtensionID, releaseTag, err)
	}

	return release, nil
}

// getLatestForBatch returns a map from extension identifiers to the latest DB release
// with the extension manifest as JSON for that extension. If there are no releases, it
// returns a nil manifest. If the manifest has no "url" field itself, a "url" field
// pointing to the extension's bundle is inserted. It also returns the date that the
// release was published.
func getLatestForBatch(ctx context.Context, db dbutil.DB, vs []*dbExtension) (map[int32]*dbRelease, error) {
	var extensionIDs []int32
	extensionIDMap := map[int32]string{}
	for _, v := range vs {
		extensionIDs = append(extensionIDs, v.ID)
		extensionIDMap[v.ID] = v.NonCanonicalExtensionID
	}
	releases, err := dbReleases{db: db}.GetLatestBatch(ctx, extensionIDs, "release", false)
	if err != nil {
		return nil, err
	}

	releasesByExtensionID := map[int32]*dbRelease{}
	for _, r := range releases {
		releasesByExtensionID[r.RegistryExtensionID] = r
	}

	for _, r := range releases {
		if err := prepReleaseManifest(extensionIDMap[r.RegistryExtensionID], r); err != nil {
			return nil, fmt.Errorf("parsing extension manifest for extension with ID %d (release tag %q): %s", r.RegistryExtensionID, "release", err)
		}
	}

	return releasesByExtensionID, nil
}

// prepReleaseManifest will set the Manifest field of the release. If the manifest has no "url"
// field itself, a "url" field pointing to the extension's bundle is inserted. It also returns
// the date that the release was published.
func prepReleaseManifest(extensionID string, release *dbRelease) error {
	// Add URL to bundle if necessary.
	o := make(map[string]interface{})
	if err := json.Unmarshal([]byte(release.Manifest), &o); err != nil {
		return err
	}
	urlStr, _ := o["url"].(string)
	if urlStr == "" {
		// Insert "url" field with link to bundle file on this site.
		bundleURL, err := makeExtensionBundleURL(release.ID, release.CreatedAt.UnixNano(), extensionID)
		if err != nil {
			return err
		}
		o["url"] = bundleURL
		b, err := json.MarshalIndent(o, "", "  ")
		if err != nil {
			return err
		}
		release.Manifest = string(b)
	}

	return nil
}

var nonLettersDigits = lazyregexp.New(`[^a-zA-Z0-9-]`)

func makeExtensionBundleURL(registryExtensionReleaseID int64, timestamp int64, extensionIDHint string) (string, error) {
	u, err := url.Parse(conf.Get().ExternalURL)
	if err != nil {
		return "", err
	}
	extensionIDHint = nonLettersDigits.ReplaceAllString(extensionIDHint, "-") // sanitize for URL path
	u.Path = path.Join(u.Path, fmt.Sprintf("/-/static/extension/%d-%s.js", registryExtensionReleaseID, extensionIDHint))
	u.RawQuery = strconv.FormatInt(timestamp, 36) + "--" + extensionIDHint // meaningless value, just for cache-busting
	return u.String(), nil
}
