package correlation

import (
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-worker/internal/correlation/datastructures"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-worker/internal/correlation/lsif"
)

// State is an in-memory representation of an uploaded LSIF index.
type State struct {
	LSIFVersion            string
	ProjectRoot            string
	DocumentData           map[int]string
	RangeData              map[int]lsif.Range
	ResultSetData          map[int]lsif.ResultSet
	DefinitionData         map[int]*datastructures.DefaultIDSetMap
	ReferenceData          map[int]*datastructures.DefaultIDSetMap
	HoverData              map[int]string
	MonikerData            map[int]lsif.Moniker
	PackageInformationData map[int]lsif.PackageInformation
	DiagnosticResults      map[int][]lsif.Diagnostic
	NextData               map[int]int                     // maps range/result sets related via next edges
	ImportedMonikers       *datastructures.IDSet           // moniker ids that have kind "import"
	ExportedMonikers       *datastructures.IDSet           // moniker ids that have kind "export"
	LinkedMonikers         *datastructures.DisjointIDSet   // tracks which moniker ids are related via next edges
	LinkedReferenceResults *datastructures.DisjointIDSet   // tracks which reference result ids are related via next edges
	Monikers               *datastructures.DefaultIDSetMap // maps items to their monikers
	Contains               *datastructures.DefaultIDSetMap // maps ranges to containing documents
	Diagnostics            *datastructures.DefaultIDSetMap // maps diagnostics to their documents
}

// newState create a new State with zero-valued map fields.
func newState() *State {
	return &State{
		DocumentData:           map[int]string{},
		RangeData:              map[int]lsif.Range{},
		ResultSetData:          map[int]lsif.ResultSet{},
		DefinitionData:         map[int]*datastructures.DefaultIDSetMap{},
		ReferenceData:          map[int]*datastructures.DefaultIDSetMap{},
		HoverData:              map[int]string{},
		MonikerData:            map[int]lsif.Moniker{},
		PackageInformationData: map[int]lsif.PackageInformation{},
		DiagnosticResults:      map[int][]lsif.Diagnostic{},
		NextData:               map[int]int{},
		ImportedMonikers:       datastructures.NewIDSet(),
		ExportedMonikers:       datastructures.NewIDSet(),
		LinkedMonikers:         datastructures.NewDisjointIDSet(),
		LinkedReferenceResults: datastructures.NewDisjointIDSet(),
		Monikers:               datastructures.NewDefaultIDSetMap(),
		Contains:               datastructures.NewDefaultIDSetMap(),
		Diagnostics:            datastructures.NewDefaultIDSetMap(),
	}
}
