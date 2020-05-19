// Code generated by github.com/efritz/go-mockgen 0.1.0; DO NOT EDIT.

package mocks

import (
	"context"
	"io"
	"sync"

	client "github.com/sourcegraph/sourcegraph/internal/codeintel/bundles/client"
)

// MockBundleManagerClient is a mock impelementation of the
// BundleManagerClient interface (from the package
// github.com/sourcegraph/sourcegraph/internal/codeintel/bundles/client)
// used for unit testing.
type MockBundleManagerClient struct {
	// BundleClientFunc is an instance of a mock function object controlling
	// the behavior of the method BundleClient.
	BundleClientFunc *BundleManagerClientBundleClientFunc
	// GetUploadFunc is an instance of a mock function object controlling
	// the behavior of the method GetUpload.
	GetUploadFunc *BundleManagerClientGetUploadFunc
	// SendDBFunc is an instance of a mock function object controlling the
	// behavior of the method SendDB.
	SendDBFunc *BundleManagerClientSendDBFunc
	// SendUploadFunc is an instance of a mock function object controlling
	// the behavior of the method SendUpload.
	SendUploadFunc *BundleManagerClientSendUploadFunc
}

// NewMockBundleManagerClient creates a new mock of the BundleManagerClient
// interface. All methods return zero values for all results, unless
// overwritten.
func NewMockBundleManagerClient() *MockBundleManagerClient {
	return &MockBundleManagerClient{
		BundleClientFunc: &BundleManagerClientBundleClientFunc{
			defaultHook: func(int) client.BundleClient {
				return nil
			},
		},
		GetUploadFunc: &BundleManagerClientGetUploadFunc{
			defaultHook: func(context.Context, int, string) (string, error) {
				return "", nil
			},
		},
		SendDBFunc: &BundleManagerClientSendDBFunc{
			defaultHook: func(context.Context, int, io.Reader) error {
				return nil
			},
		},
		SendUploadFunc: &BundleManagerClientSendUploadFunc{
			defaultHook: func(context.Context, int, io.Reader) error {
				return nil
			},
		},
	}
}

// NewMockBundleManagerClientFrom creates a new mock of the
// MockBundleManagerClient interface. All methods delegate to the given
// implementation, unless overwritten.
func NewMockBundleManagerClientFrom(i client.BundleManagerClient) *MockBundleManagerClient {
	return &MockBundleManagerClient{
		BundleClientFunc: &BundleManagerClientBundleClientFunc{
			defaultHook: i.BundleClient,
		},
		GetUploadFunc: &BundleManagerClientGetUploadFunc{
			defaultHook: i.GetUpload,
		},
		SendDBFunc: &BundleManagerClientSendDBFunc{
			defaultHook: i.SendDB,
		},
		SendUploadFunc: &BundleManagerClientSendUploadFunc{
			defaultHook: i.SendUpload,
		},
	}
}

// BundleManagerClientBundleClientFunc describes the behavior when the
// BundleClient method of the parent MockBundleManagerClient instance is
// invoked.
type BundleManagerClientBundleClientFunc struct {
	defaultHook func(int) client.BundleClient
	hooks       []func(int) client.BundleClient
	history     []BundleManagerClientBundleClientFuncCall
	mutex       sync.Mutex
}

// BundleClient delegates to the next hook function in the queue and stores
// the parameter and result values of this invocation.
func (m *MockBundleManagerClient) BundleClient(v0 int) client.BundleClient {
	r0 := m.BundleClientFunc.nextHook()(v0)
	m.BundleClientFunc.appendCall(BundleManagerClientBundleClientFuncCall{v0, r0})
	return r0
}

// SetDefaultHook sets function that is called when the BundleClient method
// of the parent MockBundleManagerClient instance is invoked and the hook
// queue is empty.
func (f *BundleManagerClientBundleClientFunc) SetDefaultHook(hook func(int) client.BundleClient) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// BundleClient method of the parent MockBundleManagerClient instance
// inovkes the hook at the front of the queue and discards it. After the
// queue is empty, the default hook function is invoked for any future
// action.
func (f *BundleManagerClientBundleClientFunc) PushHook(hook func(int) client.BundleClient) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *BundleManagerClientBundleClientFunc) SetDefaultReturn(r0 client.BundleClient) {
	f.SetDefaultHook(func(int) client.BundleClient {
		return r0
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *BundleManagerClientBundleClientFunc) PushReturn(r0 client.BundleClient) {
	f.PushHook(func(int) client.BundleClient {
		return r0
	})
}

func (f *BundleManagerClientBundleClientFunc) nextHook() func(int) client.BundleClient {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *BundleManagerClientBundleClientFunc) appendCall(r0 BundleManagerClientBundleClientFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of BundleManagerClientBundleClientFuncCall
// objects describing the invocations of this function.
func (f *BundleManagerClientBundleClientFunc) History() []BundleManagerClientBundleClientFuncCall {
	f.mutex.Lock()
	history := make([]BundleManagerClientBundleClientFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// BundleManagerClientBundleClientFuncCall is an object that describes an
// invocation of method BundleClient on an instance of
// MockBundleManagerClient.
type BundleManagerClientBundleClientFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 int
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 client.BundleClient
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c BundleManagerClientBundleClientFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c BundleManagerClientBundleClientFuncCall) Results() []interface{} {
	return []interface{}{c.Result0}
}

// BundleManagerClientGetUploadFunc describes the behavior when the
// GetUpload method of the parent MockBundleManagerClient instance is
// invoked.
type BundleManagerClientGetUploadFunc struct {
	defaultHook func(context.Context, int, string) (string, error)
	hooks       []func(context.Context, int, string) (string, error)
	history     []BundleManagerClientGetUploadFuncCall
	mutex       sync.Mutex
}

// GetUpload delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockBundleManagerClient) GetUpload(v0 context.Context, v1 int, v2 string) (string, error) {
	r0, r1 := m.GetUploadFunc.nextHook()(v0, v1, v2)
	m.GetUploadFunc.appendCall(BundleManagerClientGetUploadFuncCall{v0, v1, v2, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the GetUpload method of
// the parent MockBundleManagerClient instance is invoked and the hook queue
// is empty.
func (f *BundleManagerClientGetUploadFunc) SetDefaultHook(hook func(context.Context, int, string) (string, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// GetUpload method of the parent MockBundleManagerClient instance inovkes
// the hook at the front of the queue and discards it. After the queue is
// empty, the default hook function is invoked for any future action.
func (f *BundleManagerClientGetUploadFunc) PushHook(hook func(context.Context, int, string) (string, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *BundleManagerClientGetUploadFunc) SetDefaultReturn(r0 string, r1 error) {
	f.SetDefaultHook(func(context.Context, int, string) (string, error) {
		return r0, r1
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *BundleManagerClientGetUploadFunc) PushReturn(r0 string, r1 error) {
	f.PushHook(func(context.Context, int, string) (string, error) {
		return r0, r1
	})
}

func (f *BundleManagerClientGetUploadFunc) nextHook() func(context.Context, int, string) (string, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *BundleManagerClientGetUploadFunc) appendCall(r0 BundleManagerClientGetUploadFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of BundleManagerClientGetUploadFuncCall
// objects describing the invocations of this function.
func (f *BundleManagerClientGetUploadFunc) History() []BundleManagerClientGetUploadFuncCall {
	f.mutex.Lock()
	history := make([]BundleManagerClientGetUploadFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// BundleManagerClientGetUploadFuncCall is an object that describes an
// invocation of method GetUpload on an instance of MockBundleManagerClient.
type BundleManagerClientGetUploadFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 int
	// Arg2 is the value of the 3rd argument passed to this method
	// invocation.
	Arg2 string
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 string
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c BundleManagerClientGetUploadFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1, c.Arg2}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c BundleManagerClientGetUploadFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}

// BundleManagerClientSendDBFunc describes the behavior when the SendDB
// method of the parent MockBundleManagerClient instance is invoked.
type BundleManagerClientSendDBFunc struct {
	defaultHook func(context.Context, int, io.Reader) error
	hooks       []func(context.Context, int, io.Reader) error
	history     []BundleManagerClientSendDBFuncCall
	mutex       sync.Mutex
}

// SendDB delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockBundleManagerClient) SendDB(v0 context.Context, v1 int, v2 io.Reader) error {
	r0 := m.SendDBFunc.nextHook()(v0, v1, v2)
	m.SendDBFunc.appendCall(BundleManagerClientSendDBFuncCall{v0, v1, v2, r0})
	return r0
}

// SetDefaultHook sets function that is called when the SendDB method of the
// parent MockBundleManagerClient instance is invoked and the hook queue is
// empty.
func (f *BundleManagerClientSendDBFunc) SetDefaultHook(hook func(context.Context, int, io.Reader) error) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// SendDB method of the parent MockBundleManagerClient instance inovkes the
// hook at the front of the queue and discards it. After the queue is empty,
// the default hook function is invoked for any future action.
func (f *BundleManagerClientSendDBFunc) PushHook(hook func(context.Context, int, io.Reader) error) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *BundleManagerClientSendDBFunc) SetDefaultReturn(r0 error) {
	f.SetDefaultHook(func(context.Context, int, io.Reader) error {
		return r0
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *BundleManagerClientSendDBFunc) PushReturn(r0 error) {
	f.PushHook(func(context.Context, int, io.Reader) error {
		return r0
	})
}

func (f *BundleManagerClientSendDBFunc) nextHook() func(context.Context, int, io.Reader) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *BundleManagerClientSendDBFunc) appendCall(r0 BundleManagerClientSendDBFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of BundleManagerClientSendDBFuncCall objects
// describing the invocations of this function.
func (f *BundleManagerClientSendDBFunc) History() []BundleManagerClientSendDBFuncCall {
	f.mutex.Lock()
	history := make([]BundleManagerClientSendDBFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// BundleManagerClientSendDBFuncCall is an object that describes an
// invocation of method SendDB on an instance of MockBundleManagerClient.
type BundleManagerClientSendDBFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 int
	// Arg2 is the value of the 3rd argument passed to this method
	// invocation.
	Arg2 io.Reader
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c BundleManagerClientSendDBFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1, c.Arg2}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c BundleManagerClientSendDBFuncCall) Results() []interface{} {
	return []interface{}{c.Result0}
}

// BundleManagerClientSendUploadFunc describes the behavior when the
// SendUpload method of the parent MockBundleManagerClient instance is
// invoked.
type BundleManagerClientSendUploadFunc struct {
	defaultHook func(context.Context, int, io.Reader) error
	hooks       []func(context.Context, int, io.Reader) error
	history     []BundleManagerClientSendUploadFuncCall
	mutex       sync.Mutex
}

// SendUpload delegates to the next hook function in the queue and stores
// the parameter and result values of this invocation.
func (m *MockBundleManagerClient) SendUpload(v0 context.Context, v1 int, v2 io.Reader) error {
	r0 := m.SendUploadFunc.nextHook()(v0, v1, v2)
	m.SendUploadFunc.appendCall(BundleManagerClientSendUploadFuncCall{v0, v1, v2, r0})
	return r0
}

// SetDefaultHook sets function that is called when the SendUpload method of
// the parent MockBundleManagerClient instance is invoked and the hook queue
// is empty.
func (f *BundleManagerClientSendUploadFunc) SetDefaultHook(hook func(context.Context, int, io.Reader) error) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// SendUpload method of the parent MockBundleManagerClient instance inovkes
// the hook at the front of the queue and discards it. After the queue is
// empty, the default hook function is invoked for any future action.
func (f *BundleManagerClientSendUploadFunc) PushHook(hook func(context.Context, int, io.Reader) error) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *BundleManagerClientSendUploadFunc) SetDefaultReturn(r0 error) {
	f.SetDefaultHook(func(context.Context, int, io.Reader) error {
		return r0
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *BundleManagerClientSendUploadFunc) PushReturn(r0 error) {
	f.PushHook(func(context.Context, int, io.Reader) error {
		return r0
	})
}

func (f *BundleManagerClientSendUploadFunc) nextHook() func(context.Context, int, io.Reader) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *BundleManagerClientSendUploadFunc) appendCall(r0 BundleManagerClientSendUploadFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of BundleManagerClientSendUploadFuncCall
// objects describing the invocations of this function.
func (f *BundleManagerClientSendUploadFunc) History() []BundleManagerClientSendUploadFuncCall {
	f.mutex.Lock()
	history := make([]BundleManagerClientSendUploadFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// BundleManagerClientSendUploadFuncCall is an object that describes an
// invocation of method SendUpload on an instance of
// MockBundleManagerClient.
type BundleManagerClientSendUploadFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 int
	// Arg2 is the value of the 3rd argument passed to this method
	// invocation.
	Arg2 io.Reader
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c BundleManagerClientSendUploadFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1, c.Arg2}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c BundleManagerClientSendUploadFuncCall) Results() []interface{} {
	return []interface{}{c.Result0}
}
