package conversion

import (
	"github.com/sourcegraph/sourcegraph/lib/codeintel/lsif/conversion/datastructures"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/lsif/protocol"
)

// State is an in-memory representation of an uploaded LSIF index.
type State struct {
	LSIFVersion            string
	ProjectRoot            string
	DocumentData           map[int]string
	RangeData              map[int]Range
	ResultSetData          map[int]ResultSet
	DefinitionData         map[int]*datastructures.DefaultIDSetMap
	ReferenceData          map[int]*datastructures.DefaultIDSetMap
	HoverData              map[int]string
	MonikerData            map[int]Moniker
	PackageInformationData map[int]PackageInformation
	DiagnosticResults      map[int][]Diagnostic
	NextData               map[int]int                     // maps range/result sets related via next edges
	ImportedMonikers       *datastructures.IDSet           // moniker ids that have kind "import"
	ExportedMonikers       *datastructures.IDSet           // moniker ids that have kind "export"
	LinkedMonikers         *datastructures.DisjointIDSet   // tracks which moniker ids are related via next edges
	LinkedReferenceResults map[int][]int                   // tracks which reference result ids are related via item edges
	Monikers               *datastructures.DefaultIDSetMap // maps items to their monikers
	Contains               *datastructures.DefaultIDSetMap // maps ranges to containing documents
	Diagnostics            *datastructures.DefaultIDSetMap // maps diagnostics to their documents

	// Sourcegraph extensions
	DocumentationResultsData  map[int]protocol.Documentation // maps documentationResult vertices -> their data
	DocumentationStringsData  map[int]protocol.MarkupContent // maps documentationString vertices -> their data
	DocumentationResultRoot   int                            // the documentationResult vertex corresponding to the project root.
	DocumentationChildren     map[int][]int                  // maps documentationResult vertex -> ordered list of children documentationResult vertices
	DocumentationStringLabel  map[int]int                    // maps documentationResult vertex -> label documentationString vertex
	DocumentationStringDetail map[int]int                    // maps documentationResult vertex -> detail documentationString vertex
}

// newState create a new State with zero-valued map fields.
func newState() *State {
	return &State{
		DocumentData:           map[int]string{},
		RangeData:              map[int]Range{},
		ResultSetData:          map[int]ResultSet{},
		DefinitionData:         map[int]*datastructures.DefaultIDSetMap{},
		ReferenceData:          map[int]*datastructures.DefaultIDSetMap{},
		HoverData:              map[int]string{},
		MonikerData:            map[int]Moniker{},
		PackageInformationData: map[int]PackageInformation{},
		DiagnosticResults:      map[int][]Diagnostic{},
		NextData:               map[int]int{},
		ImportedMonikers:       datastructures.NewIDSet(),
		ExportedMonikers:       datastructures.NewIDSet(),
		LinkedMonikers:         datastructures.NewDisjointIDSet(),
		LinkedReferenceResults: map[int][]int{},
		Monikers:               datastructures.NewDefaultIDSetMap(),
		Contains:               datastructures.NewDefaultIDSetMap(),
		Diagnostics:            datastructures.NewDefaultIDSetMap(),

		// Sourcegraph extensions
		DocumentationResultsData:  map[int]protocol.Documentation{},
		DocumentationStringsData:  map[int]protocol.MarkupContent{},
		DocumentationResultRoot:   -1,
		DocumentationChildren:     map[int][]int{},
		DocumentationStringLabel:  map[int]int{},
		DocumentationStringDetail: map[int]int{},
	}
}
