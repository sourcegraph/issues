// Code generated by go-mockgen 0.1.0; DO NOT EDIT.

package discovery

import (
	"context"
	types "github.com/sourcegraph/sourcegraph/internal/types"
	"sync"
)

// MockDefaultRepoLister is a mock implementation of the DefaultRepoLister
// interface (from the package
// github.com/sourcegraph/sourcegraph/enterprise/internal/insights/discovery)
// used for unit testing.
type MockDefaultRepoLister struct {
	// ListFunc is an instance of a mock function object controlling the
	// behavior of the method List.
	ListFunc *DefaultRepoListerListFunc
}

// NewMockDefaultRepoLister creates a new mock of the DefaultRepoLister
// interface. All methods return zero values for all results, unless
// overwritten.
func NewMockDefaultRepoLister() *MockDefaultRepoLister {
	return &MockDefaultRepoLister{
		ListFunc: &DefaultRepoListerListFunc{
			defaultHook: func(context.Context) ([]types.RepoName, error) {
				return nil, nil
			},
		},
	}
}

// NewMockDefaultRepoListerFrom creates a new mock of the
// MockDefaultRepoLister interface. All methods delegate to the given
// implementation, unless overwritten.
func NewMockDefaultRepoListerFrom(i DefaultRepoLister) *MockDefaultRepoLister {
	return &MockDefaultRepoLister{
		ListFunc: &DefaultRepoListerListFunc{
			defaultHook: i.List,
		},
	}
}

// DefaultRepoListerListFunc describes the behavior when the List method of
// the parent MockDefaultRepoLister instance is invoked.
type DefaultRepoListerListFunc struct {
	defaultHook func(context.Context) ([]types.RepoName, error)
	hooks       []func(context.Context) ([]types.RepoName, error)
	history     []DefaultRepoListerListFuncCall
	mutex       sync.Mutex
}

// List delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockDefaultRepoLister) List(v0 context.Context) ([]types.RepoName, error) {
	r0, r1 := m.ListFunc.nextHook()(v0)
	m.ListFunc.appendCall(DefaultRepoListerListFuncCall{v0, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the List method of the
// parent MockDefaultRepoLister instance is invoked and the hook queue is
// empty.
func (f *DefaultRepoListerListFunc) SetDefaultHook(hook func(context.Context) ([]types.RepoName, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// List method of the parent MockDefaultRepoLister instance invokes the hook
// at the front of the queue and discards it. After the queue is empty, the
// default hook function is invoked for any future action.
func (f *DefaultRepoListerListFunc) PushHook(hook func(context.Context) ([]types.RepoName, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *DefaultRepoListerListFunc) SetDefaultReturn(r0 []types.RepoName, r1 error) {
	f.SetDefaultHook(func(context.Context) ([]types.RepoName, error) {
		return r0, r1
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *DefaultRepoListerListFunc) PushReturn(r0 []types.RepoName, r1 error) {
	f.PushHook(func(context.Context) ([]types.RepoName, error) {
		return r0, r1
	})
}

func (f *DefaultRepoListerListFunc) nextHook() func(context.Context) ([]types.RepoName, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *DefaultRepoListerListFunc) appendCall(r0 DefaultRepoListerListFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of DefaultRepoListerListFuncCall objects
// describing the invocations of this function.
func (f *DefaultRepoListerListFunc) History() []DefaultRepoListerListFuncCall {
	f.mutex.Lock()
	history := make([]DefaultRepoListerListFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// DefaultRepoListerListFuncCall is an object that describes an invocation
// of method List on an instance of MockDefaultRepoLister.
type DefaultRepoListerListFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 []types.RepoName
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c DefaultRepoListerListFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c DefaultRepoListerListFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}
