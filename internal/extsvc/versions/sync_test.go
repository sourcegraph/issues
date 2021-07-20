package versions

import (
	"context"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestGetAndStoreVersions(t *testing.T) {
	oldHandler := log15.Root().GetHandler()
	log15.Root().SetHandler(log15.DiscardHandler())
	t.Cleanup(func() { log15.Root().SetHandler(oldHandler) })

	es := []*types.ExternalService{
		{Kind: extsvc.KindGitHub, DisplayName: "github.com 1", Config: `{"url": "https://github.com"}`},
		{Kind: extsvc.KindGitHub, DisplayName: "github.com 2", Config: `{"url": "https://github.com"}`},
		{Kind: extsvc.KindGitHub, DisplayName: "github enterprise", Config: `{"url": "https://github.example.com"}`},
		{Kind: extsvc.KindGitHub, DisplayName: "gitlab", Config: `{"url": "https://gitlab.example.com"}`},
		{Kind: extsvc.KindGitHub, DisplayName: "gitlab.com", Config: `{"url": "https://gitlab.com"}`},
		{Kind: extsvc.KindGitHub, DisplayName: "bitbucket server", Config: `{"url": "https://bitbucket.sgdev.org"}`},
		{Kind: extsvc.KindGitHub, DisplayName: "another bitbucket server", Config: `{"url": "https://bitbucket2.sgdev.org"}`},
	}

	mockExternalServices := func(t *testing.T, es []*types.ExternalService) {
		database.Mocks.ExternalServices.List = func(opt database.ExternalServicesListOptions) ([]*types.ExternalService, error) {
			return es, nil
		}
		t.Cleanup(func() { database.Mocks.ExternalServices.List = nil })
	}

	t.Run("success", func(t *testing.T) {
		mockExternalServices(t, es)

		src := &fakeVersionSource{version: "1.2.3.4", err: nil, es: es}

		have, err := loadVersions(context.Background(), nil, newFakeSourcer(src))
		if err != nil {
			t.Fatal(err)
		}

		if len(have) != 6 {
			t.Errorf("wrong number of versions returned. want=%d, have=%d", 2, len(have))
		}
	})

	t.Run("error fetching version", func(t *testing.T) {
		mockExternalServices(t, es)

		testErr := errors.Errorf("what is up")
		src := &fakeVersionSource{version: "1.2.3.4", err: testErr, es: es}

		_, err := loadVersions(context.Background(), nil, newFakeSourcer(src))
		if err != nil {
			t.Fatal("error returned even though it should be logged and skipped")
		}
	})

	t.Run("error parsing external service config", func(t *testing.T) {
		invalidEs := []*types.ExternalService{
			{Kind: extsvc.KindGitHub, DisplayName: "github.com 1", Config: `invalid bogus`},
		}
		mockExternalServices(t, invalidEs)

		src := &fakeVersionSource{version: "1.2.3.4", err: nil, es: invalidEs}

		_, err := loadVersions(context.Background(), nil, newFakeSourcer(src))
		if err == nil {
			t.Fatal("no error, but was expected")
		}
	})
}

type fakeVersionSource struct {
	version string
	err     error

	es types.ExternalServices
}

func (f *fakeVersionSource) ListRepos(ctx context.Context, res chan repos.SourceResult) {}
func (f *fakeVersionSource) ExternalServices() types.ExternalServices {
	return f.es
}
func (f *fakeVersionSource) Version(context.Context) (string, error) {
	return f.version, f.err
}

func newFakeSourcer(fakeSource *fakeVersionSource) repos.Sourcer {
	return func(es ...*types.ExternalService) (repos.Sources, error) {
		return repos.Sources([]repos.Source{fakeSource}), nil
	}
}
