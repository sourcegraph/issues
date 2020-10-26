package correlation

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-worker/internal/correlation/datastructures"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-worker/internal/correlation/lsif"
)

func TestCorrelate(t *testing.T) {
	input, err := ioutil.ReadFile("../../testdata/dump1.lsif")
	if err != nil {
		t.Fatalf("unexpected error reading test file: %s", err)
	}

	state, err := correlateFromReader(bytes.NewReader(input), "root")
	if err != nil {
		t.Fatalf("unexpected error correlating input: %s", err)
	}

	expectedState := &State{
		LSIFVersion: "0.4.3",
		ProjectRoot: "file:///test/root",
		DocumentData: map[int]string{
			2: "foo.go",
			3: "bar.go",
		},
		RangeData: map[int]lsif.Range{
			4: {
				StartLine:          1,
				StartCharacter:     2,
				EndLine:            3,
				EndCharacter:       4,
				DefinitionResultID: 13,
			},
			5: {
				StartLine:         2,
				StartCharacter:    3,
				EndLine:           4,
				EndCharacter:      5,
				ReferenceResultID: 15,
			},
			6: {
				StartLine:          3,
				StartCharacter:     4,
				EndLine:            5,
				EndCharacter:       6,
				DefinitionResultID: 13,
				HoverResultID:      17,
			},
			7: {
				StartLine:         4,
				StartCharacter:    5,
				EndLine:           6,
				EndCharacter:      7,
				ReferenceResultID: 15,
			},
			8: {
				StartLine:      5,
				StartCharacter: 6,
				EndLine:        7,
				EndCharacter:   8,
				HoverResultID:  17,
			},
			9: {
				StartLine:      6,
				StartCharacter: 7,
				EndLine:        8,
				EndCharacter:   9,
			},
		},
		ResultSetData: map[int]lsif.ResultSet{
			10: {
				DefinitionResultID: 12,
				ReferenceResultID:  14,
			},
			11: {
				HoverResultID: 16,
			},
		},
		DefinitionData: map[int]*datastructures.DefaultIDSetMap{
			12: datastructures.DefaultIDSetMapWith(map[int]*datastructures.IDSet{3: datastructures.IDSetWith(7)}),
			13: datastructures.DefaultIDSetMapWith(map[int]*datastructures.IDSet{3: datastructures.IDSetWith(8)}),
		},
		ReferenceData: map[int]*datastructures.DefaultIDSetMap{
			14: datastructures.DefaultIDSetMapWith(map[int]*datastructures.IDSet{2: datastructures.IDSetWith(4, 5)}),
			15: datastructures.DefaultIDSetMapWith(map[int]*datastructures.IDSet{}),
		},
		HoverData: map[int]string{
			16: "```go\ntext A\n```",
			17: "```go\ntext B\n```",
		},
		MonikerData: map[int]lsif.Moniker{
			18: {
				Kind:                 "import",
				Scheme:               "scheme A",
				Identifier:           "ident A",
				PackageInformationID: 22,
			},
			19: {
				Kind:                 "export",
				Scheme:               "scheme B",
				Identifier:           "ident B",
				PackageInformationID: 23,
			},
			20: {
				Kind:                 "import",
				Scheme:               "scheme C",
				Identifier:           "ident C",
				PackageInformationID: 0,
			},
			21: {
				Kind:                 "export",
				Scheme:               "scheme D",
				Identifier:           "ident D",
				PackageInformationID: 0,
			},
		},
		PackageInformationData: map[int]lsif.PackageInformation{
			22: {Name: "pkg A", Version: "v0.1.0"},
			23: {Name: "pkg B", Version: "v1.2.3"},
		},
		DiagnosticResults: map[int][]lsif.Diagnostic{
			49: {
				{
					Severity:       1,
					Code:           "2322",
					Message:        "Type '10' is not assignable to type 'string'.",
					Source:         "eslint",
					StartLine:      1,
					StartCharacter: 5,
					EndLine:        1,
					EndCharacter:   6,
				},
			},
		},
		NextData: map[int]int{
			9:  10,
			10: 11,
		},
		ImportedMonikers:       datastructures.IDSetWith(18),
		ExportedMonikers:       datastructures.IDSetWith(19),
		LinkedMonikers:         datastructures.DisjointIDSetWith(19, 21),
		LinkedReferenceResults: datastructures.DisjointIDSetWith(14, 15),
		Contains: datastructures.DefaultIDSetMapWith(map[int]*datastructures.IDSet{
			2: datastructures.IDSetWith(4, 5, 6),
			3: datastructures.IDSetWith(7, 8, 9),
		}),
		Monikers: datastructures.DefaultIDSetMapWith(map[int]*datastructures.IDSet{
			7:  datastructures.IDSetWith(18),
			9:  datastructures.IDSetWith(19),
			10: datastructures.IDSetWith(20),
			11: datastructures.IDSetWith(21),
		}),
		Diagnostics: datastructures.DefaultIDSetMapWith(map[int]*datastructures.IDSet{
			2: datastructures.IDSetWith(49),
		}),
	}

	if diff := cmp.Diff(expectedState, state, datastructures.Comparers...); diff != "" {
		t.Errorf("unexpected state (-want +got):\n%s", diff)
	}
}

func TestCorrelateMetaDataRoot(t *testing.T) {
	input, err := ioutil.ReadFile("../../testdata/dump2.lsif")
	if err != nil {
		t.Fatalf("unexpected error reading test file: %s", err)
	}

	state, err := correlateFromReader(bytes.NewReader(input), "root/")
	if err != nil {
		t.Fatalf("unexpected error correlating input: %s", err)
	}

	expectedState := &State{
		LSIFVersion: "0.4.3",
		ProjectRoot: "file:///test/root/",
		DocumentData: map[int]string{
			2: "foo.go",
		},
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
		Contains:               datastructures.NewDefaultIDSetMap(),
		Monikers:               datastructures.NewDefaultIDSetMap(),
		Diagnostics:            datastructures.NewDefaultIDSetMap(),
	}

	if diff := cmp.Diff(expectedState, state, datastructures.Comparers...); diff != "" {
		t.Errorf("unexpected state (-want +got):\n%s", diff)
	}
}

func TestCorrelateMetaDataRootX(t *testing.T) {
	input, err := ioutil.ReadFile("../../testdata/dump3.lsif")
	if err != nil {
		t.Fatalf("unexpected error reading test file: %s", err)
	}

	state, err := correlateFromReader(bytes.NewReader(input), "")
	if err != nil {
		t.Fatalf("unexpected error correlating input: %s", err)
	}

	expectedState := &State{
		LSIFVersion: "0.4.3",
		ProjectRoot: "file:///__w/sourcegraph/sourcegraph/shared/",
		DocumentData: map[int]string{
			2: "../node_modules/@types/history/index.d.ts",
		},
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
		Contains:               datastructures.NewDefaultIDSetMap(),
		Monikers:               datastructures.NewDefaultIDSetMap(),
		Diagnostics:            datastructures.NewDefaultIDSetMap(),
	}

	if diff := cmp.Diff(expectedState, state, datastructures.Comparers...); diff != "" {
		t.Errorf("unexpected state (-want +got):\n%s", diff)
	}
}
