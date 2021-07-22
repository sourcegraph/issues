package main

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func makeLocation(s1, c1, s2, c2 int) Location {
	return Location{
		URI: "file://example.c",
		Range: Range{
			Start: Position{Line: s1, Character: c1},
			End:   Position{Line: s2, Character: c2},
		},
	}
}

var contents = `
/// Some documentation above
int exported_function(int a) {
  return a + 1;
}`

func TestRequiresSameURI(t *testing.T) {
	_, err := DrawLocations(contents, Location{URI: "file://a"}, Location{URI: "file://b"}, 0)
	if err == nil {
		t.Errorf("Should have errored because differing URIs")
	}
}

func TestDrawsWithOneLineDiff(t *testing.T) {
	res, _ := DrawLocations(
		contents,
		makeLocation(2, 4, 2, 20),
		makeLocation(2, 4, 2, 21),
		0,
	)

	expected := strings.Join([]string{
		"file://example.c:2",
		"|2| int exported_function(int a) {",
		"| |     ^^^^^^^^^^^^^^^^ expected",
		"| |     ^^^^^^^^^^^^^^^^^ actual",
	}, "\n")

	if diff := cmp.Diff(res, expected); diff != "" {
		t.Error(diff)
	}
}

func TestDrawsWithOneLineDiffContext(t *testing.T) {
	res, _ := DrawLocations(
		contents,
		makeLocation(2, 4, 2, 20),
		makeLocation(2, 4, 2, 21),
		1,
	)

	expected := strings.Join([]string{
		"file://example.c:2",
		"|1| /// Some documentation above",
		"|2| int exported_function(int a) {",
		"| |     ^^^^^^^^^^^^^^^^ expected",
		"| |     ^^^^^^^^^^^^^^^^^ actual",
		"|3|   return a + 1;",
	}, "\n")

	if diff := cmp.Diff(res, expected); diff != "" {
		t.Error(diff)
	}
}
