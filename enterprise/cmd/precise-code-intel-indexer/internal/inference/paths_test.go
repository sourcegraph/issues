package inference

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDirWithoutDot(t *testing.T) {
	testCases := []struct {
		actual   string
		expected string
	}{
		{dirWithoutDot("foo.txt"), ""},
		{dirWithoutDot("foo/bar.txt"), "foo"},
		{dirWithoutDot("foo/baz"), "foo"},
	}

	for _, testCase := range testCases {
		if testCase.actual != testCase.expected {
			t.Errorf("unexpected dirname: want=%s got=%s", testCase.expected, testCase.actual)
		}
	}
}

func TestAncestorDirs(t *testing.T) {
	expectedAncestors := []string{
		"foo/bar/baz",
		"foo/bar",
		"foo",
		"",
	}
	if diff := cmp.Diff(expectedAncestors, ancestorDirs("foo/bar/baz/bonk.txt")); diff != "" {
		t.Errorf("unexpected ancestor dirs (-want +got):\n%s", diff)

	}
}

func TestContainsSegment(t *testing.T) {
	testCases := []struct {
		path     string
		segment  string
		expected bool
	}{
		{path: "foo/bar/baz/bonk", segment: "foo", expected: true},
		{path: "foo/bar/baz/bonk", segment: "bar", expected: true},
		{path: "foo/bar/baz/bonk", segment: "baz", expected: true},
		{path: "foo/bar/baz/bonk", segment: "bonk", expected: true},
		{path: "foo/bar/baz/bonk", segment: "quux", expected: false},
		{path: "foo/bar/baz/bonk", segment: "honk", expected: false},
	}

	for _, testCase := range testCases {
		name := fmt.Sprintf("%s in %s", testCase.segment, testCase.path)

		t.Run(name, func(t *testing.T) {
			if value := containsSegment(testCase.path, testCase.segment); value != testCase.expected {
				t.Errorf("unexpected result. want=%v have=%v", testCase.expected, value)
			}
		})
	}
}
