package commits

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	storemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store/mocks"
)

func TestUpdate(t *testing.T) {
	graph := map[string][]string{
		"a": nil,
		"b": {"a"},
	}

	unlocked := false
	unlock := func(err error) error {
		unlocked = true
		return err
	}

	mockStore := storemocks.NewMockStore()
	mockStore.LockFunc.SetDefaultReturn(true, unlock, nil)

	mockGitserverClient := NewMockGitserverClient()
	mockGitserverClient.CommitGraphFunc.SetDefaultReturn(graph, nil)
	mockGitserverClient.HeadFunc.SetDefaultReturn("b", nil)

	if err := NewUpdater(mockStore, mockGitserverClient).Update(context.Background(), 42, nil); err != nil {
		t.Fatalf("unexpected error updating commit graph: %s", err)
	}

	if len(mockStore.LockFunc.History()) != 1 {
		t.Fatalf("unexpected lock call count. want=%d have=%d", 1, len(mockStore.LockFunc.History()))
	} else {
		call := mockStore.LockFunc.History()[0]
		if call.Arg1 != 42 {
			t.Errorf("unexpected repository id argument. want=%d have=%d", 42, call.Arg1)
		}
		if !call.Arg2 {
			t.Errorf("unexpected blocking argument. want=%v have=%v", true, call.Arg2)
		}
	}
	if !unlocked {
		t.Errorf("advisory lock not released")
	}

	if len(mockStore.CalculateVisibleUploadsFunc.History()) != 1 {
		t.Fatalf("unexpected calculate visible uploads call count. want=%d have=%d", 1, len(mockStore.CalculateVisibleUploadsFunc.History()))
	} else {
		call := mockStore.CalculateVisibleUploadsFunc.History()[0]
		if call.Arg1 != 42 {
			t.Errorf("unexpected repository id argument. want=%d have=%d", 42, call.Arg1)
		}
		if diff := cmp.Diff(graph, call.Arg2); diff != "" {
			t.Errorf("unexpected graph (-want +got):\n%s", diff)
		}
		if call.Arg3 != "b" {
			t.Errorf("unexpected repository id argument. want=%s have=%s", "b", call.Arg3)
		}
	}
}

func TestUpdateCheckFunc(t *testing.T) {
	unlocked := false
	unlock := func(err error) error {
		unlocked = true
		return err
	}

	mockStore := storemocks.NewMockStore()
	mockStore.LockFunc.SetDefaultReturn(true, unlock, nil)
	mockGitserverClient := NewMockGitserverClient()

	if err := NewUpdater(mockStore, mockGitserverClient).Update(context.Background(), 42, func(ctx context.Context) (bool, error) { return false, nil }); err != nil {
		t.Fatalf("unexpected error updating commit graph: %s", err)
	}

	if len(mockStore.LockFunc.History()) != 1 {
		t.Fatalf("unexpected lock call count. want=%d have=%d", 1, len(mockStore.LockFunc.History()))
	} else {
		call := mockStore.LockFunc.History()[0]
		if call.Arg1 != 42 {
			t.Errorf("unexpected repository id argument. want=%d have=%d", 42, call.Arg1)
		}
		if !call.Arg2 {
			t.Errorf("unexpected blocking argument. want=%v have=%v", true, call.Arg2)
		}
	}
	if !unlocked {
		t.Errorf("advisory lock not released")
	}

	if len(mockStore.CalculateVisibleUploadsFunc.History()) != 0 {
		t.Fatalf("unexpected calculate visible uploads call count. want=%d have=%d", 0, len(mockStore.CalculateVisibleUploadsFunc.History()))
	}
}

func TestTryUpdate(t *testing.T) {
	graph := map[string][]string{
		"a": nil,
		"b": {"a"},
	}

	mockStore := storemocks.NewMockStore()
	mockStore.LockFunc.SetDefaultReturn(false, nil, nil)

	mockGitserverClient := NewMockGitserverClient()
	mockGitserverClient.CommitGraphFunc.SetDefaultReturn(graph, nil)
	mockGitserverClient.HeadFunc.SetDefaultReturn("b", nil)

	if err := NewUpdater(mockStore, mockGitserverClient).TryUpdate(context.Background(), 42, 15); err != nil {
		t.Fatalf("unexpected error updating commit graph: %s", err)
	}

	if len(mockStore.LockFunc.History()) != 1 {
		t.Fatalf("unexpected lock call count. want=%d have=%d", 1, len(mockStore.LockFunc.History()))
	} else {
		call := mockStore.LockFunc.History()[0]
		if call.Arg1 != 42 {
			t.Errorf("unexpected repository id argument. want=%d have=%d", 42, call.Arg1)
		}
		if call.Arg2 {
			t.Errorf("unexpected blocking argument. want=%v have=%v", false, call.Arg2)
		}
	}

	if len(mockStore.CalculateVisibleUploadsFunc.History()) != 0 {
		t.Fatalf("unexpected calculate visible uploads call count. want=%d have=%d", 0, len(mockStore.CalculateVisibleUploadsFunc.History()))
	}
}
