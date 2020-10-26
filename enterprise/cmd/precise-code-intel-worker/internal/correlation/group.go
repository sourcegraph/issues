package correlation

import (
	"context"
	"math"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-worker/internal/correlation/datastructures"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-worker/internal/correlation/lsif"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bloomfilter"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
)

// GroupedBundleData is a view of a correlation State that sorts data by it containing document
// and shared data into shareded result chunks. The fields of this type are what is written to
// persistent storage and what is read in the query path.
type GroupedBundleData struct {
	Meta              types.MetaData
	Documents         chan persistence.KeyedDocumentData
	ResultChunks      chan persistence.IndexedResultChunkData
	Definitions       chan types.MonikerLocations
	References        chan types.MonikerLocations
	Packages          []types.Package
	PackageReferences []types.PackageReference
}

const MaxNumResultChunks = 1000
const ResultsPerResultChunk = 500

// groupBundleData converts a raw (but canonicalized) correlation State into a GroupedBundleData.
func groupBundleData(ctx context.Context, state *State, dumpID int) (*GroupedBundleData, error) {
	numResults := len(state.DefinitionData) + len(state.ReferenceData)
	numResultChunks := int(math.Min(
		MaxNumResultChunks,
		math.Max(
			1,
			math.Floor(float64(numResults)/ResultsPerResultChunk),
		),
	))

	meta := types.MetaData{NumResultChunks: numResultChunks}
	documents := serializeBundleDocuments(ctx, state)
	resultChunks := serializeResultChunks(ctx, state, numResultChunks)
	definitionRows := gatherMonikersLocations(ctx, state, state.DefinitionData, getDefinitionResultID)
	referenceRows := gatherMonikersLocations(ctx, state, state.ReferenceData, getReferenceResultID)
	packages := gatherPackages(state, dumpID)
	packageReferences, err := gatherPackageReferences(state, dumpID)
	if err != nil {
		return nil, err
	}

	return &GroupedBundleData{
		Meta:              meta,
		Documents:         documents,
		ResultChunks:      resultChunks,
		Definitions:       definitionRows,
		References:        referenceRows,
		Packages:          packages,
		PackageReferences: packageReferences,
	}, nil
}

func serializeBundleDocuments(ctx context.Context, state *State) chan persistence.KeyedDocumentData {
	ch := make(chan persistence.KeyedDocumentData)

	go func() {
		defer close(ch)

		for documentID, uri := range state.DocumentData {
			if strings.HasPrefix(uri, "..") {
				continue
			}

			data := persistence.KeyedDocumentData{
				Path:     uri,
				Document: serializeDocument(state, documentID),
			}

			select {
			case ch <- data:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

func serializeDocument(state *State, documentID int) types.DocumentData {
	document := types.DocumentData{
		Ranges:             make(map[types.ID]types.RangeData, state.Contains.SetLen(documentID)),
		HoverResults:       map[types.ID]string{},
		Monikers:           map[types.ID]types.MonikerData{},
		PackageInformation: map[types.ID]types.PackageInformationData{},
		Diagnostics:        make([]types.DiagnosticData, 0, state.Diagnostics.SetLen(documentID)),
	}

	state.Contains.SetEach(documentID, func(rangeID int) {
		rangeData := state.RangeData[rangeID]

		monikerIDs := make([]types.ID, 0, state.Monikers.SetLen(rangeID))
		state.Monikers.SetEach(rangeID, func(monikerID int) {
			moniker := state.MonikerData[monikerID]
			monikerIDs = append(monikerIDs, toID(monikerID))

			document.Monikers[toID(monikerID)] = types.MonikerData{
				Kind:                 moniker.Kind,
				Scheme:               moniker.Scheme,
				Identifier:           moniker.Identifier,
				PackageInformationID: toID(moniker.PackageInformationID),
			}

			if moniker.PackageInformationID != 0 {
				packageInformation := state.PackageInformationData[moniker.PackageInformationID]
				document.PackageInformation[toID(moniker.PackageInformationID)] = types.PackageInformationData{
					Name:    packageInformation.Name,
					Version: packageInformation.Version,
				}
			}
		})

		document.Ranges[toID(rangeID)] = types.RangeData{
			StartLine:          rangeData.StartLine,
			StartCharacter:     rangeData.StartCharacter,
			EndLine:            rangeData.EndLine,
			EndCharacter:       rangeData.EndCharacter,
			DefinitionResultID: toID(rangeData.DefinitionResultID),
			ReferenceResultID:  toID(rangeData.ReferenceResultID),
			HoverResultID:      toID(rangeData.HoverResultID),
			MonikerIDs:         monikerIDs,
		}

		if rangeData.HoverResultID != 0 {
			hoverData := state.HoverData[rangeData.HoverResultID]
			document.HoverResults[toID(rangeData.HoverResultID)] = hoverData
		}
	})

	state.Diagnostics.SetEach(documentID, func(diagnosticID int) {
		for _, diagnostic := range state.DiagnosticResults[diagnosticID] {
			document.Diagnostics = append(document.Diagnostics, types.DiagnosticData{
				Severity:       diagnostic.Severity,
				Code:           diagnostic.Code,
				Message:        diagnostic.Message,
				Source:         diagnostic.Source,
				StartLine:      diagnostic.StartLine,
				StartCharacter: diagnostic.StartCharacter,
				EndLine:        diagnostic.EndLine,
				EndCharacter:   diagnostic.EndCharacter,
			})
		}
	})

	return document
}

func serializeResultChunks(ctx context.Context, state *State, numResultChunks int) chan persistence.IndexedResultChunkData {
	chunkAssignments := make(map[int][]int, numResultChunks)
	for id := range state.DefinitionData {
		index := types.HashKey(toID(id), numResultChunks)
		chunkAssignments[index] = append(chunkAssignments[index], id)
	}
	for id := range state.ReferenceData {
		index := types.HashKey(toID(id), numResultChunks)
		chunkAssignments[index] = append(chunkAssignments[index], id)
	}

	ch := make(chan persistence.IndexedResultChunkData)

	go func() {
		defer close(ch)

		for index, resultIDs := range chunkAssignments {
			if len(resultIDs) == 0 {
				continue
			}

			documentPaths := map[types.ID]string{}
			documentIDRangeIDs := map[types.ID][]types.DocumentIDRangeID{}

			for _, resultID := range resultIDs {
				documentRanges, ok := state.DefinitionData[resultID]
				if !ok {
					documentRanges = state.ReferenceData[resultID]
				}

				// Ensure we always make an assignment for every definition and reference result,
				// even if we've pruned all of the referenced documents and ranges. This prevents
				// us from throwing an error in the bundle manager because we try to dereference
				// a missing identifier.
				documentIDRangeIDs[toID(resultID)] = nil

				documentRanges.Each(func(documentID int, rangeIDs *datastructures.IDSet) {
					documentPaths[toID(documentID)] = state.DocumentData[documentID]

					rangeIDs.Each(func(rangeID int) {
						documentIDRangeIDs[toID(resultID)] = append(documentIDRangeIDs[toID(resultID)], types.DocumentIDRangeID{
							DocumentID: toID(documentID),
							RangeID:    toID(rangeID),
						})
					})
				})
			}

			data := persistence.IndexedResultChunkData{
				Index: index,
				ResultChunk: types.ResultChunkData{
					DocumentPaths:      documentPaths,
					DocumentIDRangeIDs: documentIDRangeIDs,
				},
			}

			select {
			case ch <- data:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

var (
	getDefinitionResultID = func(r lsif.Range) int { return r.DefinitionResultID }
	getReferenceResultID  = func(r lsif.Range) int { return r.ReferenceResultID }
)

func gatherMonikersLocations(ctx context.Context, state *State, data map[int]*datastructures.DefaultIDSetMap, getResultID func(r lsif.Range) int) chan types.MonikerLocations {
	monikers := datastructures.NewDefaultIDSetMap()
	for rangeID, r := range state.RangeData {
		if resultID := getResultID(r); resultID != 0 {
			monikers.SetUnion(resultID, state.Monikers.Get(rangeID))
		}
	}

	idsBySchemeByIdentifier := map[string]map[string][]int{}
	for id := range data {
		monikerIDs := monikers.Get(id)
		if monikerIDs == nil {
			continue
		}

		monikerIDs.Each(func(monikerID int) {
			moniker := state.MonikerData[monikerID]
			idsByIdentifier, ok := idsBySchemeByIdentifier[moniker.Scheme]
			if !ok {
				idsByIdentifier = map[string][]int{}
				idsBySchemeByIdentifier[moniker.Scheme] = idsByIdentifier
			}
			idsByIdentifier[moniker.Identifier] = append(idsByIdentifier[moniker.Identifier], id)
		})
	}

	ch := make(chan types.MonikerLocations)

	go func() {
		defer close(ch)

		for scheme, idsByIdentifier := range idsBySchemeByIdentifier {
			for identifier, ids := range idsByIdentifier {
				var locations []types.Location
				for _, id := range ids {
					data[id].Each(func(documentID int, rangeIDs *datastructures.IDSet) {
						uri := state.DocumentData[documentID]
						if strings.HasPrefix(uri, "..") {
							return
						}

						rangeIDs.Each(func(id int) {
							r := state.RangeData[id]

							locations = append(locations, types.Location{
								URI:            uri,
								StartLine:      r.StartLine,
								StartCharacter: r.StartCharacter,
								EndLine:        r.EndLine,
								EndCharacter:   r.EndCharacter,
							})
						})
					})
				}

				if len(locations) == 0 {
					continue
				}

				data := types.MonikerLocations{
					Scheme:     scheme,
					Identifier: identifier,
					Locations:  locations,
				}

				select {
				case ch <- data:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch
}

func gatherPackages(state *State, dumpID int) []types.Package {
	uniques := make(map[string]types.Package, state.ExportedMonikers.Len())
	state.ExportedMonikers.Each(func(id int) {
		source := state.MonikerData[id]
		packageInfo := state.PackageInformationData[source.PackageInformationID]

		uniques[makeKey(source.Scheme, packageInfo.Name, packageInfo.Version)] = types.Package{
			DumpID:  dumpID,
			Scheme:  source.Scheme,
			Name:    packageInfo.Name,
			Version: packageInfo.Version,
		}
	})

	packages := make([]types.Package, 0, len(uniques))
	for _, v := range uniques {
		packages = append(packages, v)
	}

	return packages
}

func gatherPackageReferences(state *State, dumpID int) ([]types.PackageReference, error) {
	type ExpandedPackageReference struct {
		Scheme      string
		Name        string
		Version     string
		Identifiers []string
	}

	uniques := make(map[string]ExpandedPackageReference, state.ImportedMonikers.Len())
	state.ImportedMonikers.Each(func(id int) {
		source := state.MonikerData[id]
		packageInfo := state.PackageInformationData[source.PackageInformationID]

		key := makeKey(source.Scheme, packageInfo.Name, packageInfo.Version)
		uniques[key] = ExpandedPackageReference{
			Scheme:      source.Scheme,
			Name:        packageInfo.Name,
			Version:     packageInfo.Version,
			Identifiers: append(uniques[key].Identifiers, source.Identifier),
		}
	})

	packageReferences := make([]types.PackageReference, 0, len(uniques))
	for _, v := range uniques {
		filter, err := bloomfilter.CreateFilter(v.Identifiers)
		if err != nil {
			return nil, errors.Wrap(err, "bloomfilter.CreateFilter")
		}

		packageReferences = append(packageReferences, types.PackageReference{
			DumpID:  dumpID,
			Scheme:  v.Scheme,
			Name:    v.Name,
			Version: v.Version,
			Filter:  filter,
		})
	}

	return packageReferences, nil
}

func makeKey(parts ...string) string {
	return strings.Join(parts, ":")
}

func toID(id int) types.ID {
	if id == 0 {
		return types.ID("")
	}

	return types.ID(strconv.FormatInt(int64(id), 10))
}
