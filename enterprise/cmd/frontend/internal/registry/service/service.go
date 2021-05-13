package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/registry/store"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/registry/types"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
)

const (
	// APIVersion is a string that uniquely identifies this API version.
	APIVersion = "20180621"

	// AcceptHeader is the value of the "Accept" HTTP request header sent by the client.
	AcceptHeader = "application/vnd.sourcegraph.api+json;version=" + APIVersion

	// MediaTypeHeaderName is the name of the HTTP response header whose value the client expects to
	// equal MediaType.
	MediaTypeHeaderName = "X-Sourcegraph-Media-Type"

	// MediaType is the client's expected value for the MediaTypeHeaderName HTTP response header.
	MediaType = "sourcegraph.v" + APIVersion + "; format=json"
)

type Service struct {
	estore *store.DBExtensions
	rstore *store.DBReleases
}

func New(db dbutil.DB) *Service {
	return &Service{
		estore: store.NewDBExtensions(db),
		rstore: store.NewDBReleases(db),
	}
}

// ListRegistryExtensions lists extensions on the remote registry matching the query (or all if the query is empty).
func (s *Service) ListRegistryExtensions(ctx context.Context, registry *url.URL, query string) ([]*types.Extension, error) {
	var q url.Values
	if query != "" {
		q = url.Values{"q": []string{query}}
	}

	var xs []*types.Extension
	err := httpGet(ctx, "registry.List", toURL(registry, "extensions", q), &xs)
	return xs, err
}

// GetByUUID gets the extension from the remote registry with the given UUID. If the remote registry reports
// that the extension is not found, the returned error implements errcode.NotFounder.
func (s *Service) GetByUUID(ctx context.Context, registry *url.URL, uuidStr string) (*types.Extension, error) {
	// Loosely validate the UUID here, to avoid potential security risks if it contains a ".." path
	// component. Note that this does not normalize the UUID; for example, a UUID with prefix
	// "urn:uuid:" would be accepted (and it would be harmless and result in a not-found error).
	if _, err := uuid.Parse(uuidStr); err != nil {
		return nil, err
	}
	return getBy(ctx, registry, "registry.GetByUUID", "uuid", uuidStr)
}

// GetByExtensionID gets the extension from the remote registry with the given extension ID. If the
// remote registry reports that the extension is not found, the returned error implements
// errcode.NotFounder.
func (s *Service) GetByExtensionID(ctx context.Context, registry *url.URL, extensionID string) (*types.Extension, error) {
	return getBy(ctx, registry, "registry.GetByExtensionID", "extension-id", extensionID)
}

// GetLatestRelease returns the release with the extension manifest as JSON. If there are no
// releases, it returns a nil manifest. If the manifest has no "url" field itself, a "url" field
// pointing to the extension's bundle is inserted. It also returns the date that the release was
// published.
func (s *Service) GetLatestRelease(ctx context.Context, extensionID string, registryExtensionID int32, releaseTag string) (*store.DBRelease, error) {
	release, err := s.rstore.GetLatest(ctx, registryExtensionID, releaseTag, false)
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

// GetLatestForBatch returns a map from extension identifiers to the latest DB release
// with the extension manifest as JSON for that extension. If there are no releases, it
// returns a nil manifest. If the manifest has no "url" field itself, a "url" field
// pointing to the extension's bundle is inserted. It also returns the date that the
// release was published.
func (s *Service) GetLatestForBatch(ctx context.Context, vs []*store.DBExtension) (map[int32]*store.DBRelease, error) {
	var extensionIDs []int32
	extensionIDMap := map[int32]string{}
	for _, v := range vs {
		extensionIDs = append(extensionIDs, v.ID)
		extensionIDMap[v.ID] = v.NonCanonicalExtensionID
	}
	releases, err := s.rstore.GetLatestBatch(ctx, extensionIDs, "release", false)
	if err != nil {
		return nil, err
	}

	releasesByExtensionID := map[int32]*store.DBRelease{}
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
func prepReleaseManifest(extensionID string, release *store.DBRelease) error {
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

func toURL(registry *url.URL, pathStr string, query url.Values) string {
	return registry.ResolveReference(&url.URL{Path: path.Join(registry.Path, pathStr), RawQuery: query.Encode()}).String()
}

func getBy(ctx context.Context, registry *url.URL, op, field, value string) (*types.Extension, error) {
	var x *types.Extension
	if err := httpGet(ctx, op, toURL(registry, path.Join("extensions", field, value), nil), &x); err != nil {
		if e, ok := err.(*url.Error); ok && e.Err == httpError(http.StatusNotFound) {
			err = &extensionNotFoundError{field: field, value: value}
		}
		return nil, err
	}
	return x, nil
}

type extensionNotFoundError struct{ field, value string }

func (extensionNotFoundError) NotFound() bool { return true }
func (e *extensionNotFoundError) Error() string {
	return fmt.Sprintf("extension not found with %s %q", e.field, e.value)
}

func httpGet(ctx context.Context, op, urlStr string, result interface{}) (err error) {
	defer func() { err = errors.Wrap(err, remoteRegistryErrorMessage) }()

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", AcceptHeader)
	req.Header.Set("User-Agent", "Sourcegraph registry client v"+APIVersion)
	req = req.WithContext(ctx)

	resp, err := httpcli.ExternalDoer().Do(req)
	if err != nil {
		return err
	}
	if v := strings.TrimSpace(resp.Header.Get(MediaTypeHeaderName)); v != MediaType {
		return &url.Error{Op: op, URL: urlStr, Err: fmt.Errorf("not a valid Sourcegraph registry (invalid media type %q, expected %q)", v, MediaType)}
	}
	if resp.StatusCode != http.StatusOK {
		return &url.Error{Op: op, URL: urlStr, Err: httpError(resp.StatusCode)}
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &url.Error{Op: op, URL: urlStr, Err: err}
	}
	return nil
}

const remoteRegistryErrorMessage = "unable to contact extension registry"

// IsRemoteRegistryError reports whether the err is (likely) from this package's interaction with
// the remote registry.
func IsRemoteRegistryError(err error) bool {
	return err != nil && strings.Contains(err.Error(), remoteRegistryErrorMessage)
}

type httpError int

func (e httpError) Error() string { return fmt.Sprintf("HTTP error %d", e) }
