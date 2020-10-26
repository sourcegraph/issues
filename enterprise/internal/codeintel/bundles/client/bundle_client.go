package client

import (
	"context"
	"encoding/json"
	"fmt"

	pkgerrors "github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/database"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
	postgresreader "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/postgres"
)

// BundleClient is the interface to the precise-code-intel-bundle-manager service scoped to a particular dump.
type BundleClient interface {
	// ID gets the identifier of the target bundle.
	ID() int

	// Exists determines if the given path exists in the dump.
	Exists(ctx context.Context, path string) (bool, error)

	// Ranges returns definition, reference, and hover data for each range within the given span of lines.
	Ranges(ctx context.Context, path string, startLine, endLine int) ([]CodeIntelligenceRange, error)

	// Definitions retrieves a list of definition locations for the symbol under the given location.
	Definitions(ctx context.Context, path string, line, character int) ([]Location, error)

	// Definitions retrieves a list of reference locations for the symbol under the given location.
	References(ctx context.Context, path string, line, character int) ([]Location, error)

	// Hover retrieves the hover text for the symbol under the given location.
	Hover(ctx context.Context, path string, line, character int) (string, Range, bool, error)

	// Diagnostics retrieves the diagnostics and total count of diagnostics for the documents that have the given path prefix.
	Diagnostics(ctx context.Context, prefix string, skip, take int) ([]Diagnostic, int, error)

	// MonikersByPosition retrieves a list of monikers attached to the symbol under the given location. There may
	// be multiple ranges enclosing this point. The returned monikers are partitioned such that inner ranges occur
	// first in the result, and outer ranges occur later.
	MonikersByPosition(ctx context.Context, path string, line, character int) ([][]MonikerData, error)

	// MonikerResults retrieves a page of locations attached to a moniker and a total count of such locations.
	MonikerResults(ctx context.Context, modelType, scheme, identifier string, skip, take int) ([]Location, int, error)

	// PackageInformation retrieves package information data by its identifier.
	PackageInformation(ctx context.Context, path, packageInformationID string) (PackageInformationData, error)
}

type bundleClientImpl struct {
	base           baseClient
	bundleID       int
	store          persistence.Store
	databaseOpener func(ctx context.Context, filename string, store persistence.Store) (database.Database, error)
}

var _ BundleClient = &bundleClientImpl{}

// ID gets the identifier of the target bundle.
func (c *bundleClientImpl) ID() int {
	return c.bundleID
}

// Exists determines if the given path exists in the dump.
func (c *bundleClientImpl) Exists(ctx context.Context, path string) (exists bool, err error) {
	err = c.request(ctx, "exists", map[string]interface{}{"path": path}, &exists, func(db database.Database) (err error) {
		exists, err = db.Exists(ctx, path)
		return err
	})
	return exists, err
}

// Ranges returns definition, reference, and hover data for each range within the given span of lines.
func (c *bundleClientImpl) Ranges(ctx context.Context, path string, startLine, endLine int) (codeintelRanges []CodeIntelligenceRange, err error) {
	args := map[string]interface{}{
		"path":      path,
		"startLine": startLine,
		"endLine":   endLine,
	}

	err = c.request(ctx, "ranges", args, &codeintelRanges, func(db database.Database) (err error) {
		codeintelRanges, err = db.Ranges(ctx, path, startLine, endLine)
		return err
	})
	return codeintelRanges, err
}

// Definitions retrieves a list of definition locations for the symbol under the given location.
func (c *bundleClientImpl) Definitions(ctx context.Context, path string, line, character int) (locations []Location, err error) {
	args := map[string]interface{}{
		"path":      path,
		"line":      line,
		"character": character,
	}

	err = c.request(ctx, "definitions", args, &locations, func(db database.Database) (err error) {
		locations, err = db.Definitions(ctx, path, line, character)
		return err
	})
	c.addBundleIDToLocations(locations)
	return locations, err
}

// Definitions retrieves a list of reference locations for the symbol under the given location.
func (c *bundleClientImpl) References(ctx context.Context, path string, line, character int) (locations []Location, err error) {
	args := map[string]interface{}{
		"path":      path,
		"line":      line,
		"character": character,
	}

	err = c.request(ctx, "references", args, &locations, func(db database.Database) (err error) {
		locations, err = db.References(ctx, path, line, character)
		return err
	})
	c.addBundleIDToLocations(locations)
	return locations, err
}

// Hover retrieves the hover text for the symbol under the given location.
func (c *bundleClientImpl) Hover(ctx context.Context, path string, line, character int) (string, Range, bool, error) {
	args := map[string]interface{}{
		"path":      path,
		"line":      line,
		"character": character,
	}

	type Response struct {
		Text  string `json:"text"`
		Range Range  `json:"range"`
	}

	var target *json.RawMessage
	if err := c.request(ctx, "hover", args, &target, func(db database.Database) (err error) {
		text, r, ok, err := db.Hover(ctx, path, line, character)
		if err != nil || !ok {
			return err
		}

		contents, err := json.Marshal(Response{text, r})
		if err != nil {
			return err
		}

		jsonContents := json.RawMessage(contents)
		target = &jsonContents
		return nil
	}); err != nil {
		return "", Range{}, false, err
	}

	if target == nil {
		return "", Range{}, false, nil
	}

	var payload Response
	if err := json.Unmarshal(*target, &payload); err != nil {
		return "", Range{}, false, err
	}

	return payload.Text, payload.Range, true, nil
}

// Diagnostics retrieves the diagnostics and total count of diagnostics for the documents that have the given path prefix.
func (c *bundleClientImpl) Diagnostics(ctx context.Context, prefix string, skip, take int) (diagnostics []Diagnostic, count int, err error) {
	args := map[string]interface{}{
		"prefix": prefix,
	}
	if skip != 0 {
		args["skip"] = skip
	}
	if take != 0 {
		args["take"] = take
	}

	type Response struct {
		Diagnostics []Diagnostic `json:"diagnostics"`
		Count       int          `json:"count"`
	}

	var target Response
	err = c.request(ctx, "diagnostics", args, &target, func(db database.Database) (err error) {
		diagnostics, count, err := db.Diagnostics(ctx, prefix, skip, take)
		if err != nil {
			return err
		}

		target = Response{diagnostics, count}
		return nil
	})
	diagnostics = target.Diagnostics
	count = target.Count
	c.addBundleIDToDiagnostics(diagnostics)
	return diagnostics, count, err
}

// MonikersByPosition retrieves a list of monikers attached to the symbol under the given location. There may
// be multiple ranges enclosing this point. The returned monikers are partitioned such that inner ranges occur
// first in the result, and outer ranges occur later.
func (c *bundleClientImpl) MonikersByPosition(ctx context.Context, path string, line, character int) (target [][]MonikerData, err error) {
	args := map[string]interface{}{
		"path":      path,
		"line":      line,
		"character": character,
	}

	err = c.request(ctx, "monikersByPosition", args, &target, func(db database.Database) (err error) {
		target, err = db.MonikersByPosition(ctx, path, line, character)
		return err
	})
	return target, err
}

// MonikerResults retrieves a page of locations attached to a moniker and a total count of such locations.
func (c *bundleClientImpl) MonikerResults(ctx context.Context, modelType, scheme, identifier string, skip, take int) (locations []Location, count int, err error) {
	args := map[string]interface{}{
		"modelType":  modelType,
		"scheme":     scheme,
		"identifier": identifier,
	}
	if skip != 0 {
		args["skip"] = skip
	}
	if take != 0 {
		args["take"] = take
	}

	type Response struct {
		Locations []Location `json:"locations"`
		Count     int        `json:"count"`
	}

	var target Response
	err = c.request(ctx, "monikerResults", args, &target, func(db database.Database) (err error) {
		var tableName string
		switch modelType {
		case "definition":
			tableName = "definitions"
		case "reference":
			tableName = "references"
		}

		locations, count, err = db.MonikerResults(ctx, tableName, scheme, identifier, skip, take)
		if err != nil {
			return err
		}

		target = Response{locations, count}
		return nil
	})
	locations = target.Locations
	count = target.Count
	c.addBundleIDToLocations(locations)
	return locations, count, err
}

// PackageInformation retrieves package information data by its identifier.
func (c *bundleClientImpl) PackageInformation(ctx context.Context, path, packageInformationID string) (target PackageInformationData, err error) {
	args := map[string]interface{}{
		"path":                 path,
		"packageInformationId": packageInformationID,
	}

	err = c.request(ctx, "packageInformation", args, &target, func(db database.Database) (err error) {
		target, _, err = db.PackageInformation(ctx, path, packageInformationID)
		return err
	})
	return target, err
}

func (c *bundleClientImpl) request(ctx context.Context, path string, qs map[string]interface{}, target interface{}, handler func(database.Database) error) error {
	if _, err := c.store.ReadMeta(ctx); err != nil {
		if err == postgresreader.ErrNoMetadata {
			return c.base.QueryBundle(ctx, c.bundleID, path, qs, &target)
		}

		return err
	}

	db, err := c.databaseOpener(ctx, fmt.Sprintf("%d", c.bundleID), c.store)
	if err != nil {
		return pkgerrors.Wrap(err, "database.OpenDatabase")
	}

	return handler(db)
}

func (c *bundleClientImpl) addBundleIDToLocations(locations []Location) {
	for i := range locations {
		locations[i].DumpID = c.bundleID
	}
}

func (c *bundleClientImpl) addBundleIDToDiagnostics(diagnostics []Diagnostic) {
	for i := range diagnostics {
		diagnostics[i].DumpID = c.bundleID
	}
}
