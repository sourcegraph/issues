package registry

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/registry/store"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/registry/types"
	"github.com/sourcegraph/sourcegraph/internal/conf"
)

func Init() {
	HandleRegistry = handleRegistry
	conf.DefaultRemoteRegistry = "https://sourcegraph.com/.api/registry"
	// GetLocalExtensionByExtensionID = func(ctx context.Context, db dbutil.DB, extensionIDWithoutPrefix string) (graphqlbackend.RegistryExtension, error) {
	// 	x, err := store.NewDBExtensions(db).GetByExtensionID(ctx, extensionIDWithoutPrefix)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	if err := prefixLocalExtensionID(x); err != nil {
	// 		return nil, err
	// 	}
	// 	return &extensionDBResolver{db: db, v: x}, nil
	// }
	// ListLocalRegistryExtensions = listLocalRegistryExtensions
	// CountLocalRegistryExtensions = countLocalRegistryExtensions
	HandleRegistryExtensionBundle = handleRegistryExtensionBundle
	// IsRemoteExtensionAllowed = func(extensionID string) bool {
	// 	allowedExtensions := getAllowedExtensionsFromSiteConfig()
	// 	if allowedExtensions == nil {
	// 		// Default is to allow all extensions.
	// 		return true
	// 	}

	// 	for _, x := range allowedExtensions {
	// 		if extensionID == x {
	// 			return true
	// 		}
	// 	}
	// 	return false
	// }

	// FilterRemoteExtensions = func(extensions []*types.Extension) []*types.Extension {
	// 	allowedExtensions := getAllowedExtensionsFromSiteConfig()
	// 	if allowedExtensions == nil {
	// 		// Default is to allow all extensions.
	// 		return extensions
	// 	}

	// 	allow := make(map[string]interface{})
	// 	for _, id := range allowedExtensions {
	// 		allow[id] = struct{}{}
	// 	}
	// 	var keep []*types.Extension
	// 	for _, x := range extensions {
	// 		if _, ok := allow[x.ExtensionID]; ok {
	// 			keep = append(keep, x)
	// 		}
	// 	}
	// 	return keep
	// }
	// Allow providing fake registry data for local dev (intended for use in local dev only).
	//
	// If FAKE_REGISTRY is set and refers to a valid JSON file (of []*Extension), is used
	// by serveRegistry (instead of the DB) as the source for registry data.
	path := os.Getenv("FAKE_REGISTRY")
	if path == "" {
		return
	}

	readFakeExtensions := func() ([]*types.Extension, error) {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var xs []*types.Extension
		if err := json.Unmarshal(data, &xs); err != nil {
			return nil, err
		}
		return xs, nil
	}

	registryList = func(ctx context.Context, opt store.DBExtensionsListOptions) ([]*types.Extension, error) {
		xs, err := readFakeExtensions()
		if err != nil {
			return nil, err
		}
		return FilterRegistryExtensions(xs, opt.Query), nil
	}
	registryGetByUUID = func(ctx context.Context, uuid string) (*types.Extension, error) {
		xs, err := readFakeExtensions()
		if err != nil {
			return nil, err
		}
		return FindRegistryExtension(xs, "uuid", uuid), nil
	}
	registryGetByExtensionID = func(ctx context.Context, extensionID string) (*types.Extension, error) {
		xs, err := readFakeExtensions()
		if err != nil {
			return nil, err
		}
		return FindRegistryExtension(xs, "extensionID", extensionID), nil
	}
}
