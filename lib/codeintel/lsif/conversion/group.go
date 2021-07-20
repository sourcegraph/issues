package conversion

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/lib/codeintel/bloomfilter"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/lsif/conversion/datastructures"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/semantic"
)

// resultsPerResultChunk is the number of target keys in a single result chunk. This may
// not reflect the actual number of keys in a result sets, as result chunk identifiers
// are hashed into buckets based on the total number of result sets (and this value).
//
// This number does not prevent pathological cases where a single result chunk will have
// very large values, as only the number of keys (not total values within the keyspace)
// are used to determine the hashing scheme.
const resultsPerResultChunk = 512

// groupBundleData converts a raw (but canonicalized) correlation State into a GroupedBundleData.
func groupBundleData(ctx context.Context, state *State) (*semantic.GroupedBundleDataChans, error) {
	numResults := len(state.DefinitionData) + len(state.ReferenceData)
	numResultChunks := int(math.Max(1, math.Floor(float64(numResults)/resultsPerResultChunk)))

	meta := semantic.MetaData{NumResultChunks: numResultChunks}
	documents := serializeBundleDocuments(ctx, state)
	resultChunks := serializeResultChunks(ctx, state, numResultChunks)
	definitionRows := gatherMonikersLocations(ctx, state, state.DefinitionData, func(r Range) int { return r.DefinitionResultID })
	referenceRows := gatherMonikersLocations(ctx, state, state.ReferenceData, func(r Range) int { return r.ReferenceResultID })
	documentation := collectDocumentation(ctx, state)
	packages := gatherPackages(state)
	packageReferences, err := gatherPackageReferences(state, packages)
	if err != nil {
		return nil, err
	}

	return &semantic.GroupedBundleDataChans{
		Meta:                  meta,
		Documents:             documents,
		ResultChunks:          resultChunks,
		Definitions:           definitionRows,
		References:            referenceRows,
		DocumentationPages:    documentation.pages,
		DocumentationPathInfo: documentation.pathInfo,
		DocumentationMappings: documentation.mappings,
		Packages:              packages,
		PackageReferences:     packageReferences,
	}, nil
}

func serializeBundleDocuments(ctx context.Context, state *State) chan semantic.KeyedDocumentData {
	ch := make(chan semantic.KeyedDocumentData)

	go func() {
		defer close(ch)

		for documentID, uri := range state.DocumentData {
			if strings.HasPrefix(uri, "..") {
				continue
			}

			data := semantic.KeyedDocumentData{
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

func serializeDocument(state *State, documentID int) semantic.DocumentData {
	document := semantic.DocumentData{
		Ranges:             make(map[semantic.ID]semantic.RangeData, state.Contains.SetLen(documentID)),
		HoverResults:       map[semantic.ID]string{},
		Monikers:           map[semantic.ID]semantic.MonikerData{},
		PackageInformation: map[semantic.ID]semantic.PackageInformationData{},
		Diagnostics:        make([]semantic.DiagnosticData, 0, state.Diagnostics.SetLen(documentID)),
	}

	state.Contains.SetEach(documentID, func(rangeID int) {
		rangeData := state.RangeData[rangeID]

		monikerIDs := make([]semantic.ID, 0, state.Monikers.SetLen(rangeID))
		state.Monikers.SetEach(rangeID, func(monikerID int) {
			moniker := state.MonikerData[monikerID]
			monikerIDs = append(monikerIDs, toID(monikerID))

			document.Monikers[toID(monikerID)] = semantic.MonikerData{
				Kind:                 moniker.Kind,
				Scheme:               moniker.Scheme,
				Identifier:           moniker.Identifier,
				PackageInformationID: toID(moniker.PackageInformationID),
			}

			if moniker.PackageInformationID != 0 {
				packageInformation := state.PackageInformationData[moniker.PackageInformationID]
				document.PackageInformation[toID(moniker.PackageInformationID)] = semantic.PackageInformationData{
					Name:    packageInformation.Name,
					Version: packageInformation.Version,
				}
			}
		})

		document.Ranges[toID(rangeID)] = semantic.RangeData{
			StartLine:          rangeData.Start.Line,
			StartCharacter:     rangeData.Start.Character,
			EndLine:            rangeData.End.Line,
			EndCharacter:       rangeData.End.Character,
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
			document.Diagnostics = append(document.Diagnostics, semantic.DiagnosticData{
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

func serializeResultChunks(ctx context.Context, state *State, numResultChunks int) chan semantic.IndexedResultChunkData {
	chunkAssignments := make(map[int][]int, numResultChunks)
	for id := range state.DefinitionData {
		index := semantic.HashKey(toID(id), numResultChunks)
		chunkAssignments[index] = append(chunkAssignments[index], id)
	}
	for id := range state.ReferenceData {
		index := semantic.HashKey(toID(id), numResultChunks)
		chunkAssignments[index] = append(chunkAssignments[index], id)
	}

	ch := make(chan semantic.IndexedResultChunkData)

	go func() {
		defer close(ch)

		for index, resultIDs := range chunkAssignments {
			if len(resultIDs) == 0 {
				continue
			}

			documentPaths := map[semantic.ID]string{}
			rangeIDsByResultID := make(map[semantic.ID][]semantic.DocumentIDRangeID, len(resultIDs))

			for _, resultID := range resultIDs {
				documentRanges, ok := state.DefinitionData[resultID]
				if !ok {
					documentRanges = state.ReferenceData[resultID]
				}

				rangeIDMap := map[semantic.ID]int{}
				var documentIDRangeIDs []semantic.DocumentIDRangeID

				documentRanges.Each(func(documentID int, rangeIDs *datastructures.IDSet) {
					docID := toID(documentID)
					documentPaths[docID] = state.DocumentData[documentID]

					rangeIDs.Each(func(rangeID int) {
						rangeIDMap[toID(rangeID)] = rangeID

						documentIDRangeIDs = append(documentIDRangeIDs, semantic.DocumentIDRangeID{
							DocumentID: docID,
							RangeID:    toID(rangeID),
						})
					})
				})

				// Sort locations by containing document path then by offset within the text
				// document (in reading order). This provides us with an obvious and deterministic
				// ordering of a result set over multiple API requests.

				sort.Sort(sortableDocumentIDRangeIDs{
					state:         state,
					documentPaths: documentPaths,
					rangeIDMap:    rangeIDMap,
					s:             documentIDRangeIDs,
				})

				rangeIDsByResultID[toID(resultID)] = documentIDRangeIDs
			}

			data := semantic.IndexedResultChunkData{
				Index: index,
				ResultChunk: semantic.ResultChunkData{
					DocumentPaths:      documentPaths,
					DocumentIDRangeIDs: rangeIDsByResultID,
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

// sortableDocumentIDRangeIDs implements sort.Interface for document/range id pairs.
type sortableDocumentIDRangeIDs struct {
	state         *State
	documentPaths map[semantic.ID]string
	rangeIDMap    map[semantic.ID]int
	s             []semantic.DocumentIDRangeID
}

func (s sortableDocumentIDRangeIDs) Len() int      { return len(s.s) }
func (s sortableDocumentIDRangeIDs) Swap(i, j int) { s.s[i], s.s[j] = s.s[j], s.s[i] }
func (s sortableDocumentIDRangeIDs) Less(i, j int) bool {
	iDocumentID := s.s[i].DocumentID
	jDocumentID := s.s[j].DocumentID
	iRange := s.state.RangeData[s.rangeIDMap[s.s[i].RangeID]]
	jRange := s.state.RangeData[s.rangeIDMap[s.s[j].RangeID]]

	if s.documentPaths[iDocumentID] != s.documentPaths[jDocumentID] {
		return s.documentPaths[iDocumentID] <= s.documentPaths[jDocumentID]
	}

	if cmp := iRange.Start.Line - jRange.Start.Line; cmp != 0 {
		return cmp < 0
	}

	return iRange.Start.Character-jRange.Start.Character < 0
}

func gatherMonikersLocations(ctx context.Context, state *State, data map[int]*datastructures.DefaultIDSetMap, getResultID func(r Range) int) chan semantic.MonikerLocations {
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

	ch := make(chan semantic.MonikerLocations)

	go func() {
		defer close(ch)

		for scheme, idsByIdentifier := range idsBySchemeByIdentifier {
			for identifier, ids := range idsByIdentifier {
				var locations []semantic.LocationData
				for _, id := range ids {
					data[id].Each(func(documentID int, rangeIDs *datastructures.IDSet) {
						uri := state.DocumentData[documentID]
						if strings.HasPrefix(uri, "..") {
							return
						}

						rangeIDs.Each(func(id int) {
							r := state.RangeData[id]

							locations = append(locations, semantic.LocationData{
								URI:            uri,
								StartLine:      r.Start.Line,
								StartCharacter: r.Start.Character,
								EndLine:        r.End.Line,
								EndCharacter:   r.End.Character,
							})
						})
					})
				}

				if len(locations) == 0 {
					continue
				}

				// Sort locations by containing document path then by offset within the text
				// document (in reading order). This provides us with an obvious and deterministic
				// ordering of a result set over multiple API requests.

				sort.Sort(sortableLocations(locations))

				data := semantic.MonikerLocations{
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

// sortableLocations implements sort.Interface for locations.
type sortableLocations []semantic.LocationData

func (s sortableLocations) Len() int      { return len(s) }
func (s sortableLocations) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortableLocations) Less(i, j int) bool {
	if s[i].URI != s[j].URI {
		return s[i].URI <= s[j].URI
	}

	if cmp := s[i].StartLine - s[j].StartLine; cmp != 0 {
		return cmp < 0
	}

	return s[i].StartCharacter < s[j].StartCharacter
}

func gatherPackages(state *State) []semantic.Package {
	uniques := make(map[string]semantic.Package, state.ExportedMonikers.Len())
	state.ExportedMonikers.Each(func(id int) {
		source := state.MonikerData[id]
		packageInfo := state.PackageInformationData[source.PackageInformationID]

		uniques[makeKey(source.Scheme, packageInfo.Name, packageInfo.Version)] = semantic.Package{
			Scheme:  source.Scheme,
			Name:    packageInfo.Name,
			Version: packageInfo.Version,
		}
	})

	packages := make([]semantic.Package, 0, len(uniques))
	for _, v := range uniques {
		packages = append(packages, v)
	}

	return packages
}

func gatherPackageReferences(state *State, packageDefinitions []semantic.Package) ([]semantic.PackageReference, error) {
	type ExpandedPackageReference struct {
		Scheme      string
		Name        string
		Version     string
		Identifiers []string
	}

	packageDefinitionKeySet := make(map[string]struct{}, len(packageDefinitions))
	for _, pkg := range packageDefinitions {
		packageDefinitionKeySet[makeKey(pkg.Scheme, pkg.Name, pkg.Version)] = struct{}{}
	}

	uniques := make(map[string]ExpandedPackageReference, state.ImportedMonikers.Len())
	state.ImportedMonikers.Each(func(id int) {
		source := state.MonikerData[id]
		packageInfo := state.PackageInformationData[source.PackageInformationID]
		key := makeKey(source.Scheme, packageInfo.Name, packageInfo.Version)

		if _, ok := packageDefinitionKeySet[key]; ok {
			// We use package definitions and references as a way to link an index
			// to its remote dependency. storing self-references is a waste of space
			// and complicates our data retention path when considering the set of
			// indexes that are referred to only by relevant/visible remote indexes.
			return
		}

		uniques[key] = ExpandedPackageReference{
			Scheme:      source.Scheme,
			Name:        packageInfo.Name,
			Version:     packageInfo.Version,
			Identifiers: append(uniques[key].Identifiers, source.Identifier),
		}
	})

	packageReferences := make([]semantic.PackageReference, 0, len(uniques))
	for _, v := range uniques {
		filter, err := bloomfilter.CreateFilter(v.Identifiers)
		if err != nil {
			return nil, errors.Wrap(err, "bloomfilter.CreateFilter")
		}

		packageReferences = append(packageReferences, semantic.PackageReference{
			Package: semantic.Package{
				Scheme:  v.Scheme,
				Name:    v.Name,
				Version: v.Version,
			},
			Filter: filter,
		})
	}

	return packageReferences, nil
}
