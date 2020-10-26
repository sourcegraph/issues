package search

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/zoekt/ignore"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

func TestNewIgnoreMatcher(t *testing.T) {
	git.Mocks.ReadFile = func(commit api.CommitID, name string) ([]byte, error) {
		return []byte("foo/"), nil
	}
	defer func() { git.Mocks.ReadFile = nil }()

	ig, err := newIgnoreMatcher(context.Background(), gitserver.Repo{}, "")
	if err != nil {
		t.Error(err)
	}
	if !ig.Match("foo/bar.go") {
		t.Errorf("ignore.Matcher should have matched")
	}
}

func TestMissingIgnoreFile(t *testing.T) {
	git.Mocks.ReadFile = func(commit api.CommitID, name string) ([]byte, error) {
		return nil, fmt.Errorf("err open .sourcegraph/ignore: file does not exist")
	}
	defer func() { git.Mocks.ReadFile = nil }()

	ig, err := newIgnoreMatcher(context.Background(), gitserver.Repo{}, "")
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(ig, &ignore.Matcher{}) {
		t.Error("newIgnoreMatchers should have returned &ignore.Matcher{} if the ignore-file is missing")
	}
}
