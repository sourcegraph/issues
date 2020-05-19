// Code generated by github.com/efritz/go-mockgen 0.1.0; DO NOT EDIT.

package mocks

import (
	"context"
	"sync"

	client "github.com/sourcegraph/sourcegraph/internal/codeintel/bundles/client"
)

// MockBundleClient is a mock impelementation of the BundleClient interface
// (from the package
// github.com/sourcegraph/sourcegraph/internal/codeintel/bundles/client)
// used for unit testing.
type MockBundleClient struct {
	// DefinitionsFunc is an instance of a mock function object controlling
	// the behavior of the method Definitions.
	DefinitionsFunc *BundleClientDefinitionsFunc
	// ExistsFunc is an instance of a mock function object controlling the
	// behavior of the method Exists.
	ExistsFunc *BundleClientExistsFunc
	// HoverFunc is an instance of a mock function object controlling the
	// behavior of the method Hover.
	HoverFunc *BundleClientHoverFunc
	// MonikerResultsFunc is an instance of a mock function object
	// controlling the behavior of the method MonikerResults.
	MonikerResultsFunc *BundleClientMonikerResultsFunc
	// MonikersByPositionFunc is an instance of a mock function object
	// controlling the behavior of the method MonikersByPosition.
	MonikersByPositionFunc *BundleClientMonikersByPositionFunc
	// PackageInformationFunc is an instance of a mock function object
	// controlling the behavior of the method PackageInformation.
	PackageInformationFunc *BundleClientPackageInformationFunc
	// ReferencesFunc is an instance of a mock function object controlling
	// the behavior of the method References.
	ReferencesFunc *BundleClientReferencesFunc
}

// NewMockBundleClient creates a new mock of the BundleClient interface. All
// methods return zero values for all results, unless overwritten.
func NewMockBundleClient() *MockBundleClient {
	return &MockBundleClient{
		DefinitionsFunc: &BundleClientDefinitionsFunc{
			defaultHook: func(context.Context, string, int, int) ([]client.Location, error) {
				return nil, nil
			},
		},
		ExistsFunc: &BundleClientExistsFunc{
			defaultHook: func(context.Context, string) (bool, error) {
				return false, nil
			},
		},
		HoverFunc: &BundleClientHoverFunc{
			defaultHook: func(context.Context, string, int, int) (string, client.Range, bool, error) {
				return "", client.Range{}, false, nil
			},
		},
		MonikerResultsFunc: &BundleClientMonikerResultsFunc{
			defaultHook: func(context.Context, string, string, string, int, int) ([]client.Location, int, error) {
				return nil, 0, nil
			},
		},
		MonikersByPositionFunc: &BundleClientMonikersByPositionFunc{
			defaultHook: func(context.Context, string, int, int) ([][]client.MonikerData, error) {
				return nil, nil
			},
		},
		PackageInformationFunc: &BundleClientPackageInformationFunc{
			defaultHook: func(context.Context, string, string) (client.PackageInformationData, error) {
				return client.PackageInformationData{}, nil
			},
		},
		ReferencesFunc: &BundleClientReferencesFunc{
			defaultHook: func(context.Context, string, int, int) ([]client.Location, error) {
				return nil, nil
			},
		},
	}
}

// NewMockBundleClientFrom creates a new mock of the MockBundleClient
// interface. All methods delegate to the given implementation, unless
// overwritten.
func NewMockBundleClientFrom(i client.BundleClient) *MockBundleClient {
	return &MockBundleClient{
		DefinitionsFunc: &BundleClientDefinitionsFunc{
			defaultHook: i.Definitions,
		},
		ExistsFunc: &BundleClientExistsFunc{
			defaultHook: i.Exists,
		},
		HoverFunc: &BundleClientHoverFunc{
			defaultHook: i.Hover,
		},
		MonikerResultsFunc: &BundleClientMonikerResultsFunc{
			defaultHook: i.MonikerResults,
		},
		MonikersByPositionFunc: &BundleClientMonikersByPositionFunc{
			defaultHook: i.MonikersByPosition,
		},
		PackageInformationFunc: &BundleClientPackageInformationFunc{
			defaultHook: i.PackageInformation,
		},
		ReferencesFunc: &BundleClientReferencesFunc{
			defaultHook: i.References,
		},
	}
}

// BundleClientDefinitionsFunc describes the behavior when the Definitions
// method of the parent MockBundleClient instance is invoked.
type BundleClientDefinitionsFunc struct {
	defaultHook func(context.Context, string, int, int) ([]client.Location, error)
	hooks       []func(context.Context, string, int, int) ([]client.Location, error)
	history     []BundleClientDefinitionsFuncCall
	mutex       sync.Mutex
}

// Definitions delegates to the next hook function in the queue and stores
// the parameter and result values of this invocation.
func (m *MockBundleClient) Definitions(v0 context.Context, v1 string, v2 int, v3 int) ([]client.Location, error) {
	r0, r1 := m.DefinitionsFunc.nextHook()(v0, v1, v2, v3)
	m.DefinitionsFunc.appendCall(BundleClientDefinitionsFuncCall{v0, v1, v2, v3, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the Definitions method
// of the parent MockBundleClient instance is invoked and the hook queue is
// empty.
func (f *BundleClientDefinitionsFunc) SetDefaultHook(hook func(context.Context, string, int, int) ([]client.Location, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// Definitions method of the parent MockBundleClient instance inovkes the
// hook at the front of the queue and discards it. After the queue is empty,
// the default hook function is invoked for any future action.
func (f *BundleClientDefinitionsFunc) PushHook(hook func(context.Context, string, int, int) ([]client.Location, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *BundleClientDefinitionsFunc) SetDefaultReturn(r0 []client.Location, r1 error) {
	f.SetDefaultHook(func(context.Context, string, int, int) ([]client.Location, error) {
		return r0, r1
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *BundleClientDefinitionsFunc) PushReturn(r0 []client.Location, r1 error) {
	f.PushHook(func(context.Context, string, int, int) ([]client.Location, error) {
		return r0, r1
	})
}

func (f *BundleClientDefinitionsFunc) nextHook() func(context.Context, string, int, int) ([]client.Location, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *BundleClientDefinitionsFunc) appendCall(r0 BundleClientDefinitionsFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of BundleClientDefinitionsFuncCall objects
// describing the invocations of this function.
func (f *BundleClientDefinitionsFunc) History() []BundleClientDefinitionsFuncCall {
	f.mutex.Lock()
	history := make([]BundleClientDefinitionsFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// BundleClientDefinitionsFuncCall is an object that describes an invocation
// of method Definitions on an instance of MockBundleClient.
type BundleClientDefinitionsFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 string
	// Arg2 is the value of the 3rd argument passed to this method
	// invocation.
	Arg2 int
	// Arg3 is the value of the 4th argument passed to this method
	// invocation.
	Arg3 int
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 []client.Location
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c BundleClientDefinitionsFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1, c.Arg2, c.Arg3}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c BundleClientDefinitionsFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}

// BundleClientExistsFunc describes the behavior when the Exists method of
// the parent MockBundleClient instance is invoked.
type BundleClientExistsFunc struct {
	defaultHook func(context.Context, string) (bool, error)
	hooks       []func(context.Context, string) (bool, error)
	history     []BundleClientExistsFuncCall
	mutex       sync.Mutex
}

// Exists delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockBundleClient) Exists(v0 context.Context, v1 string) (bool, error) {
	r0, r1 := m.ExistsFunc.nextHook()(v0, v1)
	m.ExistsFunc.appendCall(BundleClientExistsFuncCall{v0, v1, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the Exists method of the
// parent MockBundleClient instance is invoked and the hook queue is empty.
func (f *BundleClientExistsFunc) SetDefaultHook(hook func(context.Context, string) (bool, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// Exists method of the parent MockBundleClient instance inovkes the hook at
// the front of the queue and discards it. After the queue is empty, the
// default hook function is invoked for any future action.
func (f *BundleClientExistsFunc) PushHook(hook func(context.Context, string) (bool, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *BundleClientExistsFunc) SetDefaultReturn(r0 bool, r1 error) {
	f.SetDefaultHook(func(context.Context, string) (bool, error) {
		return r0, r1
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *BundleClientExistsFunc) PushReturn(r0 bool, r1 error) {
	f.PushHook(func(context.Context, string) (bool, error) {
		return r0, r1
	})
}

func (f *BundleClientExistsFunc) nextHook() func(context.Context, string) (bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *BundleClientExistsFunc) appendCall(r0 BundleClientExistsFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of BundleClientExistsFuncCall objects
// describing the invocations of this function.
func (f *BundleClientExistsFunc) History() []BundleClientExistsFuncCall {
	f.mutex.Lock()
	history := make([]BundleClientExistsFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// BundleClientExistsFuncCall is an object that describes an invocation of
// method Exists on an instance of MockBundleClient.
type BundleClientExistsFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 string
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 bool
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c BundleClientExistsFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c BundleClientExistsFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}

// BundleClientHoverFunc describes the behavior when the Hover method of the
// parent MockBundleClient instance is invoked.
type BundleClientHoverFunc struct {
	defaultHook func(context.Context, string, int, int) (string, client.Range, bool, error)
	hooks       []func(context.Context, string, int, int) (string, client.Range, bool, error)
	history     []BundleClientHoverFuncCall
	mutex       sync.Mutex
}

// Hover delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockBundleClient) Hover(v0 context.Context, v1 string, v2 int, v3 int) (string, client.Range, bool, error) {
	r0, r1, r2, r3 := m.HoverFunc.nextHook()(v0, v1, v2, v3)
	m.HoverFunc.appendCall(BundleClientHoverFuncCall{v0, v1, v2, v3, r0, r1, r2, r3})
	return r0, r1, r2, r3
}

// SetDefaultHook sets function that is called when the Hover method of the
// parent MockBundleClient instance is invoked and the hook queue is empty.
func (f *BundleClientHoverFunc) SetDefaultHook(hook func(context.Context, string, int, int) (string, client.Range, bool, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// Hover method of the parent MockBundleClient instance inovkes the hook at
// the front of the queue and discards it. After the queue is empty, the
// default hook function is invoked for any future action.
func (f *BundleClientHoverFunc) PushHook(hook func(context.Context, string, int, int) (string, client.Range, bool, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *BundleClientHoverFunc) SetDefaultReturn(r0 string, r1 client.Range, r2 bool, r3 error) {
	f.SetDefaultHook(func(context.Context, string, int, int) (string, client.Range, bool, error) {
		return r0, r1, r2, r3
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *BundleClientHoverFunc) PushReturn(r0 string, r1 client.Range, r2 bool, r3 error) {
	f.PushHook(func(context.Context, string, int, int) (string, client.Range, bool, error) {
		return r0, r1, r2, r3
	})
}

func (f *BundleClientHoverFunc) nextHook() func(context.Context, string, int, int) (string, client.Range, bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *BundleClientHoverFunc) appendCall(r0 BundleClientHoverFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of BundleClientHoverFuncCall objects
// describing the invocations of this function.
func (f *BundleClientHoverFunc) History() []BundleClientHoverFuncCall {
	f.mutex.Lock()
	history := make([]BundleClientHoverFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// BundleClientHoverFuncCall is an object that describes an invocation of
// method Hover on an instance of MockBundleClient.
type BundleClientHoverFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 string
	// Arg2 is the value of the 3rd argument passed to this method
	// invocation.
	Arg2 int
	// Arg3 is the value of the 4th argument passed to this method
	// invocation.
	Arg3 int
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 string
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 client.Range
	// Result2 is the value of the 3rd result returned from this method
	// invocation.
	Result2 bool
	// Result3 is the value of the 4th result returned from this method
	// invocation.
	Result3 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c BundleClientHoverFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1, c.Arg2, c.Arg3}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c BundleClientHoverFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1, c.Result2, c.Result3}
}

// BundleClientMonikerResultsFunc describes the behavior when the
// MonikerResults method of the parent MockBundleClient instance is invoked.
type BundleClientMonikerResultsFunc struct {
	defaultHook func(context.Context, string, string, string, int, int) ([]client.Location, int, error)
	hooks       []func(context.Context, string, string, string, int, int) ([]client.Location, int, error)
	history     []BundleClientMonikerResultsFuncCall
	mutex       sync.Mutex
}

// MonikerResults delegates to the next hook function in the queue and
// stores the parameter and result values of this invocation.
func (m *MockBundleClient) MonikerResults(v0 context.Context, v1 string, v2 string, v3 string, v4 int, v5 int) ([]client.Location, int, error) {
	r0, r1, r2 := m.MonikerResultsFunc.nextHook()(v0, v1, v2, v3, v4, v5)
	m.MonikerResultsFunc.appendCall(BundleClientMonikerResultsFuncCall{v0, v1, v2, v3, v4, v5, r0, r1, r2})
	return r0, r1, r2
}

// SetDefaultHook sets function that is called when the MonikerResults
// method of the parent MockBundleClient instance is invoked and the hook
// queue is empty.
func (f *BundleClientMonikerResultsFunc) SetDefaultHook(hook func(context.Context, string, string, string, int, int) ([]client.Location, int, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// MonikerResults method of the parent MockBundleClient instance inovkes the
// hook at the front of the queue and discards it. After the queue is empty,
// the default hook function is invoked for any future action.
func (f *BundleClientMonikerResultsFunc) PushHook(hook func(context.Context, string, string, string, int, int) ([]client.Location, int, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *BundleClientMonikerResultsFunc) SetDefaultReturn(r0 []client.Location, r1 int, r2 error) {
	f.SetDefaultHook(func(context.Context, string, string, string, int, int) ([]client.Location, int, error) {
		return r0, r1, r2
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *BundleClientMonikerResultsFunc) PushReturn(r0 []client.Location, r1 int, r2 error) {
	f.PushHook(func(context.Context, string, string, string, int, int) ([]client.Location, int, error) {
		return r0, r1, r2
	})
}

func (f *BundleClientMonikerResultsFunc) nextHook() func(context.Context, string, string, string, int, int) ([]client.Location, int, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *BundleClientMonikerResultsFunc) appendCall(r0 BundleClientMonikerResultsFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of BundleClientMonikerResultsFuncCall objects
// describing the invocations of this function.
func (f *BundleClientMonikerResultsFunc) History() []BundleClientMonikerResultsFuncCall {
	f.mutex.Lock()
	history := make([]BundleClientMonikerResultsFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// BundleClientMonikerResultsFuncCall is an object that describes an
// invocation of method MonikerResults on an instance of MockBundleClient.
type BundleClientMonikerResultsFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 string
	// Arg2 is the value of the 3rd argument passed to this method
	// invocation.
	Arg2 string
	// Arg3 is the value of the 4th argument passed to this method
	// invocation.
	Arg3 string
	// Arg4 is the value of the 5th argument passed to this method
	// invocation.
	Arg4 int
	// Arg5 is the value of the 6th argument passed to this method
	// invocation.
	Arg5 int
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 []client.Location
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 int
	// Result2 is the value of the 3rd result returned from this method
	// invocation.
	Result2 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c BundleClientMonikerResultsFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1, c.Arg2, c.Arg3, c.Arg4, c.Arg5}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c BundleClientMonikerResultsFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1, c.Result2}
}

// BundleClientMonikersByPositionFunc describes the behavior when the
// MonikersByPosition method of the parent MockBundleClient instance is
// invoked.
type BundleClientMonikersByPositionFunc struct {
	defaultHook func(context.Context, string, int, int) ([][]client.MonikerData, error)
	hooks       []func(context.Context, string, int, int) ([][]client.MonikerData, error)
	history     []BundleClientMonikersByPositionFuncCall
	mutex       sync.Mutex
}

// MonikersByPosition delegates to the next hook function in the queue and
// stores the parameter and result values of this invocation.
func (m *MockBundleClient) MonikersByPosition(v0 context.Context, v1 string, v2 int, v3 int) ([][]client.MonikerData, error) {
	r0, r1 := m.MonikersByPositionFunc.nextHook()(v0, v1, v2, v3)
	m.MonikersByPositionFunc.appendCall(BundleClientMonikersByPositionFuncCall{v0, v1, v2, v3, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the MonikersByPosition
// method of the parent MockBundleClient instance is invoked and the hook
// queue is empty.
func (f *BundleClientMonikersByPositionFunc) SetDefaultHook(hook func(context.Context, string, int, int) ([][]client.MonikerData, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// MonikersByPosition method of the parent MockBundleClient instance inovkes
// the hook at the front of the queue and discards it. After the queue is
// empty, the default hook function is invoked for any future action.
func (f *BundleClientMonikersByPositionFunc) PushHook(hook func(context.Context, string, int, int) ([][]client.MonikerData, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *BundleClientMonikersByPositionFunc) SetDefaultReturn(r0 [][]client.MonikerData, r1 error) {
	f.SetDefaultHook(func(context.Context, string, int, int) ([][]client.MonikerData, error) {
		return r0, r1
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *BundleClientMonikersByPositionFunc) PushReturn(r0 [][]client.MonikerData, r1 error) {
	f.PushHook(func(context.Context, string, int, int) ([][]client.MonikerData, error) {
		return r0, r1
	})
}

func (f *BundleClientMonikersByPositionFunc) nextHook() func(context.Context, string, int, int) ([][]client.MonikerData, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *BundleClientMonikersByPositionFunc) appendCall(r0 BundleClientMonikersByPositionFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of BundleClientMonikersByPositionFuncCall
// objects describing the invocations of this function.
func (f *BundleClientMonikersByPositionFunc) History() []BundleClientMonikersByPositionFuncCall {
	f.mutex.Lock()
	history := make([]BundleClientMonikersByPositionFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// BundleClientMonikersByPositionFuncCall is an object that describes an
// invocation of method MonikersByPosition on an instance of
// MockBundleClient.
type BundleClientMonikersByPositionFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 string
	// Arg2 is the value of the 3rd argument passed to this method
	// invocation.
	Arg2 int
	// Arg3 is the value of the 4th argument passed to this method
	// invocation.
	Arg3 int
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 [][]client.MonikerData
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c BundleClientMonikersByPositionFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1, c.Arg2, c.Arg3}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c BundleClientMonikersByPositionFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}

// BundleClientPackageInformationFunc describes the behavior when the
// PackageInformation method of the parent MockBundleClient instance is
// invoked.
type BundleClientPackageInformationFunc struct {
	defaultHook func(context.Context, string, string) (client.PackageInformationData, error)
	hooks       []func(context.Context, string, string) (client.PackageInformationData, error)
	history     []BundleClientPackageInformationFuncCall
	mutex       sync.Mutex
}

// PackageInformation delegates to the next hook function in the queue and
// stores the parameter and result values of this invocation.
func (m *MockBundleClient) PackageInformation(v0 context.Context, v1 string, v2 string) (client.PackageInformationData, error) {
	r0, r1 := m.PackageInformationFunc.nextHook()(v0, v1, v2)
	m.PackageInformationFunc.appendCall(BundleClientPackageInformationFuncCall{v0, v1, v2, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the PackageInformation
// method of the parent MockBundleClient instance is invoked and the hook
// queue is empty.
func (f *BundleClientPackageInformationFunc) SetDefaultHook(hook func(context.Context, string, string) (client.PackageInformationData, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// PackageInformation method of the parent MockBundleClient instance inovkes
// the hook at the front of the queue and discards it. After the queue is
// empty, the default hook function is invoked for any future action.
func (f *BundleClientPackageInformationFunc) PushHook(hook func(context.Context, string, string) (client.PackageInformationData, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *BundleClientPackageInformationFunc) SetDefaultReturn(r0 client.PackageInformationData, r1 error) {
	f.SetDefaultHook(func(context.Context, string, string) (client.PackageInformationData, error) {
		return r0, r1
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *BundleClientPackageInformationFunc) PushReturn(r0 client.PackageInformationData, r1 error) {
	f.PushHook(func(context.Context, string, string) (client.PackageInformationData, error) {
		return r0, r1
	})
}

func (f *BundleClientPackageInformationFunc) nextHook() func(context.Context, string, string) (client.PackageInformationData, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *BundleClientPackageInformationFunc) appendCall(r0 BundleClientPackageInformationFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of BundleClientPackageInformationFuncCall
// objects describing the invocations of this function.
func (f *BundleClientPackageInformationFunc) History() []BundleClientPackageInformationFuncCall {
	f.mutex.Lock()
	history := make([]BundleClientPackageInformationFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// BundleClientPackageInformationFuncCall is an object that describes an
// invocation of method PackageInformation on an instance of
// MockBundleClient.
type BundleClientPackageInformationFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 string
	// Arg2 is the value of the 3rd argument passed to this method
	// invocation.
	Arg2 string
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 client.PackageInformationData
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c BundleClientPackageInformationFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1, c.Arg2}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c BundleClientPackageInformationFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}

// BundleClientReferencesFunc describes the behavior when the References
// method of the parent MockBundleClient instance is invoked.
type BundleClientReferencesFunc struct {
	defaultHook func(context.Context, string, int, int) ([]client.Location, error)
	hooks       []func(context.Context, string, int, int) ([]client.Location, error)
	history     []BundleClientReferencesFuncCall
	mutex       sync.Mutex
}

// References delegates to the next hook function in the queue and stores
// the parameter and result values of this invocation.
func (m *MockBundleClient) References(v0 context.Context, v1 string, v2 int, v3 int) ([]client.Location, error) {
	r0, r1 := m.ReferencesFunc.nextHook()(v0, v1, v2, v3)
	m.ReferencesFunc.appendCall(BundleClientReferencesFuncCall{v0, v1, v2, v3, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the References method of
// the parent MockBundleClient instance is invoked and the hook queue is
// empty.
func (f *BundleClientReferencesFunc) SetDefaultHook(hook func(context.Context, string, int, int) ([]client.Location, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// References method of the parent MockBundleClient instance inovkes the
// hook at the front of the queue and discards it. After the queue is empty,
// the default hook function is invoked for any future action.
func (f *BundleClientReferencesFunc) PushHook(hook func(context.Context, string, int, int) ([]client.Location, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *BundleClientReferencesFunc) SetDefaultReturn(r0 []client.Location, r1 error) {
	f.SetDefaultHook(func(context.Context, string, int, int) ([]client.Location, error) {
		return r0, r1
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *BundleClientReferencesFunc) PushReturn(r0 []client.Location, r1 error) {
	f.PushHook(func(context.Context, string, int, int) ([]client.Location, error) {
		return r0, r1
	})
}

func (f *BundleClientReferencesFunc) nextHook() func(context.Context, string, int, int) ([]client.Location, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *BundleClientReferencesFunc) appendCall(r0 BundleClientReferencesFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of BundleClientReferencesFuncCall objects
// describing the invocations of this function.
func (f *BundleClientReferencesFunc) History() []BundleClientReferencesFuncCall {
	f.mutex.Lock()
	history := make([]BundleClientReferencesFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// BundleClientReferencesFuncCall is an object that describes an invocation
// of method References on an instance of MockBundleClient.
type BundleClientReferencesFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 string
	// Arg2 is the value of the 3rd argument passed to this method
	// invocation.
	Arg2 int
	// Arg3 is the value of the 4th argument passed to this method
	// invocation.
	Arg3 int
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 []client.Location
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c BundleClientReferencesFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1, c.Arg2, c.Arg3}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c BundleClientReferencesFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}
