package api

import (
	"context"
	"strings"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
)

// Hover returns the hover text and range for the symbol at the given position.
func (api *codeIntelAPI) Hover(ctx context.Context, file string, line, character, uploadID int) (string, bundles.Range, bool, error) {
	dump, exists, err := api.store.GetDumpByID(ctx, uploadID)
	if err != nil {
		return "", bundles.Range{}, false, errors.Wrap(err, "store.GetDumpByID")
	}
	if !exists {
		return "", bundles.Range{}, false, ErrMissingDump
	}

	pathInBundle := strings.TrimPrefix(file, dump.Root)
	bundleClient := api.bundleManagerClient.BundleClient(dump.ID)

	text, rn, exists, err := bundleClient.Hover(ctx, pathInBundle, line, character)
	if err != nil {
		if err == bundles.ErrNotFound {
			log15.Warn("Bundle does not exist")
			return "", bundles.Range{}, false, nil
		}
		return "", bundles.Range{}, false, errors.Wrap(err, "bundleClient.Hover")
	}
	if exists {
		return text, rn, true, nil
	}

	definition, exists, err := api.definitionRaw(ctx, dump, bundleClient, pathInBundle, line, character)
	if err != nil || !exists {
		return "", bundles.Range{}, false, errors.Wrap(err, "api.definitionRaw")
	}

	pathInDefinitionBundle := strings.TrimPrefix(definition.Path, definition.Dump.Root)
	definitionBundleClient := api.bundleManagerClient.BundleClient(definition.Dump.ID)

	text, rn, exists, err = definitionBundleClient.Hover(ctx, pathInDefinitionBundle, definition.Range.Start.Line, definition.Range.Start.Character)
	if err != nil {
		if err == bundles.ErrNotFound {
			log15.Warn("Bundle does not exist")
			return "", bundles.Range{}, false, nil
		}
		return "", bundles.Range{}, false, errors.Wrap(err, "definitionBundleClient.Hover")
	}

	return text, rn, exists, nil
}
