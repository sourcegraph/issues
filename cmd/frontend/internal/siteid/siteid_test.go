package siteid

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/globalstatedb"
)

func TestNotInited(t *testing.T) {
	if inited {
		t.Fatal("one of this test package's imports called Init, but these tests require that it has not yet been called")
	}
}

func TestGet(t *testing.T) {
	reset := func() {
		inited = false
		siteID = ""
		db.Mocks = db.MockStores{}
		conf.Mock(nil)
	}

	{
		origFatalln := fatalln
		fatalln = func(v ...interface{}) { panic(v) }
		defer func() { fatalln = origFatalln }()
	}

	tryInit := func() (err error) {
		defer func() {
			if e := recover(); e != nil {
				err = fmt.Errorf("panic: %v", e)
			}
		}()
		Init()
		return nil
	}

	t.Run("from DB", func(t *testing.T) {
		defer reset()
		globalstatedb.Mock.Get = func(ctx context.Context) (*globalstatedb.State, error) {
			return &globalstatedb.State{SiteID: "a"}, nil
		}

		if err := tryInit(); err != nil {
			t.Fatal(err)
		}
		if !inited {
			t.Error("!inited")
		}
		if got, want := Get(), "a"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("panics if DB unavailable", func(t *testing.T) {
		defer reset()
		globalstatedb.Mock.Get = func(ctx context.Context) (*globalstatedb.State, error) {
			return nil, errors.New("x")
		}

		want := fmt.Errorf("panic: [Error initializing global state: x]")
		if err := tryInit(); fmt.Sprint(err) != fmt.Sprint(want) {
			t.Errorf("got error %q, want %q", err, want)
		}
		if inited {
			t.Error("inited")
		}
		if siteID != "" {
			t.Error("siteID is set")
		}
	})

	t.Run("from env var", func(t *testing.T) {
		defer reset()
		os.Setenv("TRACKING_APP_ID", "a")
		defer os.Unsetenv("TRACKING_APP_ID")

		if err := tryInit(); err != nil {
			t.Fatal(err)
		}
		if !inited {
			t.Error("!inited")
		}
		if got, want := Get(), "a"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("env var takes precedence over DB", func(t *testing.T) {
		defer reset()
		os.Setenv("TRACKING_APP_ID", "a")
		defer os.Unsetenv("TRACKING_APP_ID")
		globalstatedb.Mock.Get = func(ctx context.Context) (*globalstatedb.State, error) {
			return &globalstatedb.State{SiteID: "b"}, nil
		}

		if err := tryInit(); err != nil {
			t.Fatal(err)
		}
		if !inited {
			t.Error("!inited")
		}
		if got, want := Get(), "a"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
