package api

import (
	"context"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
)

func lookupMoniker(
	store store.Store,
	bundleManagerClient bundles.BundleManagerClient,
	dumpID int,
	path string,
	modelType string,
	moniker bundles.MonikerData,
	skip int,
	take int,
) ([]ResolvedLocation, int, error) {
	if moniker.PackageInformationID == "" {
		return nil, 0, nil
	}

	pid, err := bundleManagerClient.BundleClient(dumpID).PackageInformation(context.Background(), path, moniker.PackageInformationID)
	if err != nil {
		if err == bundles.ErrNotFound {
			log15.Warn("Bundle does not exist")
			return nil, 0, nil
		}
		return nil, 0, errors.Wrap(err, "bundleManagerClient.BundleClient")
	}

	dump, exists, err := store.GetPackage(context.Background(), moniker.Scheme, pid.Name, pid.Version)
	if err != nil || !exists {
		return nil, 0, errors.Wrap(err, "store.GetPackage")
	}

	locations, count, err := bundleManagerClient.BundleClient(dump.ID).MonikerResults(context.Background(), modelType, moniker.Scheme, moniker.Identifier, skip, take)
	if err != nil {
		if err == bundles.ErrNotFound {
			log15.Warn("Bundle does not exist")
			return nil, 0, nil
		}
		return nil, 0, errors.Wrap(err, "bundleManagerClient.BundleClient")
	}

	return resolveLocationsWithDump(dump, locations), count, nil
}
