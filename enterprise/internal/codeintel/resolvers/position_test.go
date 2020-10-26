package resolvers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/go-diff/diff"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

func TestAdjustPath(t *testing.T) {
	adjuster := NewPositionAdjuster(&types.Repo{ID: 50}, "deadbeef1", nil)
	path, ok, err := adjuster.AdjustPath(context.Background(), "deadbeef2", "/foo/bar.go", false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !ok {
		t.Errorf("expected translation to succeed")
	}
	if path != "/foo/bar.go" {
		t.Errorf("unexpected path. want=%s have=%s", "/foo/bar.go", path)
	}
}

func TestAdjustPosition(t *testing.T) {
	t.Cleanup(func() {
		git.Mocks.ExecReader = nil
	})
	git.Mocks.ExecReader = func(args []string) (reader io.ReadCloser, err error) {
		expectedArgs := []string{"diff", "deadbeef1", "deadbeef2", "--", "/foo/bar.go"}
		if diff := cmp.Diff(expectedArgs, args); diff != "" {
			t.Errorf("unexpected exec reader args (-want +got):\n%s", diff)
		}

		return ioutil.NopCloser(bytes.NewReader([]byte(hugoDiff))), nil
	}

	posIn := bundles.Position{Line: 302, Character: 15}

	adjuster := NewPositionAdjuster(&types.Repo{ID: 50}, "deadbeef1", nil)
	path, posOut, ok, err := adjuster.AdjustPosition(context.Background(), "deadbeef2", "/foo/bar.go", posIn, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !ok {
		t.Errorf("expected translation to succeed")
	}
	if path != "/foo/bar.go" {
		t.Errorf("unexpected path. want=%s have=%s", "/foo/bar.go", path)
	}

	expectedPos := bundles.Position{Line: 294, Character: 15}
	if diff := cmp.Diff(expectedPos, posOut); diff != "" {
		t.Errorf("unexpected position (-want +got):\n%s", diff)
	}
}

func TestAdjustPositionEmptyDiff(t *testing.T) {
	t.Cleanup(func() {
		git.Mocks.ExecReader = nil
	})
	git.Mocks.ExecReader = func(args []string) (reader io.ReadCloser, err error) {
		return ioutil.NopCloser(bytes.NewReader(nil)), nil
	}

	posIn := bundles.Position{Line: 10, Character: 15}

	adjuster := NewPositionAdjuster(&types.Repo{ID: 50}, "deadbeef1", nil)
	path, posOut, ok, err := adjuster.AdjustPosition(context.Background(), "deadbeef2", "/foo/bar.go", posIn, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !ok {
		t.Errorf("expected translation to succeed")
	}
	if path != "/foo/bar.go" {
		t.Errorf("unexpected path. want=%s have=%s", "/foo/bar.go", path)
	}
	if diff := cmp.Diff(posOut, posIn); diff != "" {
		t.Errorf("unexpected position (-want +got):\n%s", diff)
	}
}

func TestAdjustPositionReverse(t *testing.T) {
	t.Cleanup(func() {
		git.Mocks.ExecReader = nil
	})
	git.Mocks.ExecReader = func(args []string) (reader io.ReadCloser, err error) {
		expectedArgs := []string{"diff", "deadbeef2", "deadbeef1", "--", "/foo/bar.go"}
		if diff := cmp.Diff(expectedArgs, args); diff != "" {
			t.Errorf("unexpected exec reader args (-want +got):\n%s", diff)
		}

		return ioutil.NopCloser(bytes.NewReader([]byte(hugoDiff))), nil
	}

	posIn := bundles.Position{Line: 302, Character: 15}

	adjuster := NewPositionAdjuster(&types.Repo{ID: 50}, "deadbeef1", nil)
	path, posOut, ok, err := adjuster.AdjustPosition(context.Background(), "deadbeef2", "/foo/bar.go", posIn, true)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !ok {
		t.Errorf("expected translation to succeed")
	}
	if path != "/foo/bar.go" {
		t.Errorf("unexpected path. want=%s have=%s", "/foo/bar.go", path)
	}

	expectedPos := bundles.Position{Line: 294, Character: 15}
	if diff := cmp.Diff(expectedPos, posOut); diff != "" {
		t.Errorf("unexpected position (-want +got):\n%s", diff)
	}
}

func TestAdjustRange(t *testing.T) {
	t.Cleanup(func() {
		git.Mocks.ExecReader = nil
	})
	git.Mocks.ExecReader = func(args []string) (reader io.ReadCloser, err error) {
		expectedArgs := []string{"diff", "deadbeef1", "deadbeef2", "--", "/foo/bar.go"}
		if diff := cmp.Diff(expectedArgs, args); diff != "" {
			t.Errorf("unexpected exec reader args (-want +got):\n%s", diff)
		}

		return ioutil.NopCloser(bytes.NewReader([]byte(hugoDiff))), nil
	}

	rIn := bundles.Range{
		Start: bundles.Position{Line: 302, Character: 15},
		End:   bundles.Position{Line: 305, Character: 20},
	}

	adjuster := NewPositionAdjuster(&types.Repo{ID: 50}, "deadbeef1", nil)
	path, rOut, ok, err := adjuster.AdjustRange(context.Background(), "deadbeef2", "/foo/bar.go", rIn, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !ok {
		t.Errorf("expected translation to succeed")
	}
	if path != "/foo/bar.go" {
		t.Errorf("unexpected path. want=%s have=%s", "/foo/bar.go", path)
	}

	expectedRange := bundles.Range{
		Start: bundles.Position{Line: 294, Character: 15},
		End:   bundles.Position{Line: 297, Character: 20},
	}
	if diff := cmp.Diff(expectedRange, rOut); diff != "" {
		t.Errorf("unexpected position (-want +got):\n%s", diff)
	}
}

func TestAdjustRangeEmptyDiff(t *testing.T) {
	t.Cleanup(func() {
		git.Mocks.ExecReader = nil
	})
	git.Mocks.ExecReader = func(args []string) (reader io.ReadCloser, err error) {
		return ioutil.NopCloser(bytes.NewReader(nil)), nil
	}

	rIn := bundles.Range{
		Start: bundles.Position{Line: 302, Character: 15},
		End:   bundles.Position{Line: 305, Character: 20},
	}

	adjuster := NewPositionAdjuster(&types.Repo{ID: 50}, "deadbeef1", nil)
	path, rOut, ok, err := adjuster.AdjustRange(context.Background(), "deadbeef2", "/foo/bar.go", rIn, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !ok {
		t.Errorf("expected translation to succeed")
	}
	if path != "/foo/bar.go" {
		t.Errorf("unexpected path. want=%s have=%s", "/foo/bar.go", path)
	}
	if diff := cmp.Diff(rOut, rIn); diff != "" {
		t.Errorf("unexpected position (-want +got):\n%s", diff)
	}
}

func TestAdjustRangeReverse(t *testing.T) {
	t.Cleanup(func() {
		git.Mocks.ExecReader = nil
	})
	git.Mocks.ExecReader = func(args []string) (reader io.ReadCloser, err error) {
		expectedArgs := []string{"diff", "deadbeef2", "deadbeef1", "--", "/foo/bar.go"}
		if diff := cmp.Diff(expectedArgs, args); diff != "" {
			t.Errorf("unexpected exec reader args (-want +got):\n%s", diff)
		}

		return ioutil.NopCloser(bytes.NewReader([]byte(hugoDiff))), nil
	}

	rIn := bundles.Range{
		Start: bundles.Position{Line: 302, Character: 15},
		End:   bundles.Position{Line: 305, Character: 20},
	}

	adjuster := NewPositionAdjuster(&types.Repo{ID: 50}, "deadbeef1", nil)
	path, rOut, ok, err := adjuster.AdjustRange(context.Background(), "deadbeef2", "/foo/bar.go", rIn, true)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !ok {
		t.Errorf("expected translation to succeed")
	}
	if path != "/foo/bar.go" {
		t.Errorf("unexpected path. want=%s have=%s", "/foo/bar.go", path)
	}

	expectedRange := bundles.Range{
		Start: bundles.Position{Line: 294, Character: 15},
		End:   bundles.Position{Line: 297, Character: 20},
	}
	if diff := cmp.Diff(expectedRange, rOut); diff != "" {
		t.Errorf("unexpected position (-want +got):\n%s", diff)
	}
}

type adjustPositionTestCase struct {
	diff         string // The git diff output
	diffName     string // The git diff output name
	description  string // The description of the test
	line         int    // The target line (one-indexed)
	expectedOk   bool   // Whether the operation should succeed
	expectedLine int    // The expected adjusted line (one-indexed)
}

// hugoDiff is a diff from github.com/gohugoio/hugo generated via the following command.
// git diff 8947c3fa0beec021e14b3f8040857335e1ecd473 3e9db2ad951dbb1000cd0f8f25e4a95445046679 -- resources/image.go
const hugoDiff = `
diff --git a/resources/image.go b/resources/image.go
index d1d9f650d673..076f2ae4d63b 100644
--- a/resources/image.go
+++ b/resources/image.go
@@ -36,7 +36,6 @@ import (

        "github.com/gohugoio/hugo/resources/resource"

-       "github.com/pkg/errors"
        _errors "github.com/pkg/errors"

        "github.com/gohugoio/hugo/helpers"
@@ -235,7 +234,7 @@ const imageProcWorkers = 1
 var imageProcSem = make(chan bool, imageProcWorkers)

 func (i *imageResource) doWithImageConfig(conf images.ImageConfig, f func(src image.Image) (image.Image, error)) (resource.Image, error) {
-       img, err := i.getSpec().imageCache.getOrCreate(i, conf, func() (*imageResource, image.Image, error) {
+       return i.getSpec().imageCache.getOrCreate(i, conf, func() (*imageResource, image.Image, error) {
                imageProcSem <- true
                defer func() {
                        <-imageProcSem
@@ -292,13 +291,6 @@ func (i *imageResource) doWithImageConfig(conf images.ImageConfig, f func(src im

                return ci, converted, nil
        })
-
-       if err != nil {
-               if i.root != nil && i.root.getFileInfo() != nil {
-                       return nil, errors.Wrapf(err, "image %q", i.root.getFileInfo().Meta().Filename())
-               }
-       }
-       return img, nil
 }

 func (i *imageResource) decodeImageConfig(action, spec string) (images.ImageConfig, error) {
`

var hugoTestCases = []adjustPositionTestCase{
	// Between hunks
	{hugoDiff, "hugo", "before first hunk", 10, true, 10},
	{hugoDiff, "hugo", "between hunks (1x deletion)", 150, true, 149},
	{hugoDiff, "hugo", "between hunks (1x deletion, 1x edit)", 250, true, 249},
	{hugoDiff, "hugo", "after last hunk (2x deletions, 1x edit)", 350, true, 342},

	// Hunk 1
	{hugoDiff, "hugo", "before first hunk deletion", 38, true, 38},
	{hugoDiff, "hugo", "on first hunk deletion", 39, false, 0},
	{hugoDiff, "hugo", "after first hunk deletion", 40, true, 39},

	// Hunk 1 (lower border)
	{hugoDiff, "hugo", "inside first hunk context (last line)", 43, true, 42},
	{hugoDiff, "hugo", "directly after first hunk", 44, true, 43},

	// Hunk 2
	{hugoDiff, "hugo", "before second hunk edit", 237, true, 236},
	{hugoDiff, "hugo", "on edited hunk edit", 238, false, 0},
	{hugoDiff, "hugo", "after second hunk edit", 239, true, 238},

	// Hunk 3
	{hugoDiff, "hugo", "before third hunk deletion", 294, true, 293},
	{hugoDiff, "hugo", "on third hunk deletion", 295, false, 0},
	{hugoDiff, "hugo", "on third hunk deletion", 301, false, 0},
	{hugoDiff, "hugo", "after third hunk deletion", 302, true, 294},
}

// prometheusDiff is a diff from github.com/prometheus/prometheus generated via the following command.
// git diff 52025bd7a9446c3178bf01dd2949d4874dd45f24 45fbed94d6ee17840254e78cfc421ab1db78f734 -- discovery/manager.go
const prometheusDiff = `
diff --git a/discovery/manager.go b/discovery/manager.go
index 49bcbf86b7ba..d135cd54e700 100644
--- a/discovery/manager.go
+++ b/discovery/manager.go
@@ -293,11 +293,11 @@ func (m *Manager) updateGroup(poolKey poolKey, tgs []*targetgroup.Group) {
        m.mtx.Lock()
        defer m.mtx.Unlock()

-       if _, ok := m.targets[poolKey]; !ok {
-               m.targets[poolKey] = make(map[string]*targetgroup.Group)
-       }
        for _, tg := range tgs {
                if tg != nil { // Some Discoverers send nil target group so need to check for it to avoid panics.
+                       if _, ok := m.targets[poolKey]; !ok {
+                               m.targets[poolKey] = make(map[string]*targetgroup.Group)
+                       }
                        m.targets[poolKey][tg.Source] = tg
                }
        }
`

var prometheusTestCases = []adjustPositionTestCase{
	{prometheusDiff, "prometheus", "before hunk", 100, true, 100},
	{prometheusDiff, "prometheus", "before deletion", 295, true, 295},
	{prometheusDiff, "prometheus", "on deletion 1", 296, false, 0},
	{prometheusDiff, "prometheus", "on deletion 2", 297, false, 0},
	{prometheusDiff, "prometheus", "on deletion 3", 298, false, 0},
	{prometheusDiff, "prometheus", "after deletion", 299, true, 296},
	{prometheusDiff, "prometheus", "before insertion", 300, true, 297},
	{prometheusDiff, "prometheus", "after insertion", 301, true, 301},
	{prometheusDiff, "prometheus", "after hunk", 500, true, 500},
}

func TestRawAdjustPosition(t *testing.T) {
	for _, testCase := range append(append([]adjustPositionTestCase(nil), hugoTestCases...), prometheusTestCases...) {
		name := fmt.Sprintf("%s : %s", testCase.diffName, testCase.description)

		t.Run(name, func(t *testing.T) {
			diff, err := diff.NewFileDiffReader(bytes.NewReader([]byte(testCase.diff))).Read()
			if err != nil {
				t.Fatalf("unexpected error reading file diff: %s", err)
			}
			hunks := diff.Hunks

			pos := bundles.Position{
				Line:      testCase.line - 1, // 1-index -> 0-index
				Character: 10,
			}

			if adjusted, ok := adjustPosition(hunks, pos); ok != testCase.expectedOk {
				t.Errorf("unexpected ok. want=%v have=%v", testCase.expectedOk, ok)
			} else if ok {
				// Adjust from zero-index to one-index
				if adjusted.Line+1 != testCase.expectedLine {
					t.Errorf("unexpected line. want=%d have=%d", testCase.expectedLine, adjusted.Line+1) // 0-index -> 1-index
				}
				if adjusted.Character != 10 {
					t.Errorf("unexpected character. want=%d have=%d", 10, adjusted.Character)
				}
			}
		})
	}
}
