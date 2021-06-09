// Code generated by go-mockgen 0.1.0; DO NOT EDIT.

package background

import (
	"context"
	"sync"

	api "github.com/sourcegraph/sourcegraph/internal/api"
	types "github.com/sourcegraph/sourcegraph/internal/types"
)

// MockRepoStore is a mock implementation of the RepoStore interface (from
// the package
// github.com/sourcegraph/sourcegraph/enterprise/internal/insights/background)
// used for unit testing.
type MockRepoStore struct {
	// GetByNameFunc is an instance of a mock function object controlling
	// the behavior of the method GetByName.
	GetByNameFunc *RepoStoreGetByNameFunc
}

// NewMockRepoStore creates a new mock of the RepoStore interface. All
// methods return zero values for all results, unless overwritten.
func NewMockRepoStore() *MockRepoStore {
	return &MockRepoStore{
		GetByNameFunc: &RepoStoreGetByNameFunc{
			defaultHook: func(context.Context, api.RepoName) (*types.Repo, error) {
				return nil, nil
			},
		},
	}
}

// NewMockRepoStoreFrom creates a new mock of the MockRepoStore interface.
// All methods delegate to the given implementation, unless overwritten.
func NewMockRepoStoreFrom(i RepoStore) *MockRepoStore {
	return &MockRepoStore{
		GetByNameFunc: &RepoStoreGetByNameFunc{
			defaultHook: i.GetByName,
		},
	}
}

// RepoStoreGetByNameFunc describes the behavior when the GetByName method
// of the parent MockRepoStore instance is invoked.
type RepoStoreGetByNameFunc struct {
	defaultHook func(context.Context, api.RepoName) (*types.Repo, error)
	hooks       []func(context.Context, api.RepoName) (*types.Repo, error)
	history     []RepoStoreGetByNameFuncCall
	mutex       sync.Mutex
}

// GetByName delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockRepoStore) GetByName(v0 context.Context, v1 api.RepoName) (*types.Repo, error) {
	r0, r1 := m.GetByNameFunc.nextHook()(v0, v1)
	m.GetByNameFunc.appendCall(RepoStoreGetByNameFuncCall{v0, v1, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the GetByName method of
// the parent MockRepoStore instance is invoked and the hook queue is empty.
func (f *RepoStoreGetByNameFunc) SetDefaultHook(hook func(context.Context, api.RepoName) (*types.Repo, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// GetByName method of the parent MockRepoStore instance invokes the hook at
// the front of the queue and discards it. After the queue is empty, the
// default hook function is invoked for any future action.
func (f *RepoStoreGetByNameFunc) PushHook(hook func(context.Context, api.RepoName) (*types.Repo, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *RepoStoreGetByNameFunc) SetDefaultReturn(r0 *types.Repo, r1 error) {
	f.SetDefaultHook(func(context.Context, api.RepoName) (*types.Repo, error) {
		return r0, r1
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *RepoStoreGetByNameFunc) PushReturn(r0 *types.Repo, r1 error) {
	f.PushHook(func(context.Context, api.RepoName) (*types.Repo, error) {
		return r0, r1
	})
}

func (f *RepoStoreGetByNameFunc) nextHook() func(context.Context, api.RepoName) (*types.Repo, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *RepoStoreGetByNameFunc) appendCall(r0 RepoStoreGetByNameFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of RepoStoreGetByNameFuncCall objects
// describing the invocations of this function.
func (f *RepoStoreGetByNameFunc) History() []RepoStoreGetByNameFuncCall {
	f.mutex.Lock()
	history := make([]RepoStoreGetByNameFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// RepoStoreGetByNameFuncCall is an object that describes an invocation of
// method GetByName on an instance of MockRepoStore.
type RepoStoreGetByNameFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 api.RepoName
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 *types.Repo
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c RepoStoreGetByNameFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c RepoStoreGetByNameFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}
