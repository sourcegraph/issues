package testutil

import (
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/dnaeon/go-vcr/cassette"
	"github.com/dnaeon/go-vcr/recorder"

	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/httptestutil"
)

type VCR struct {
	cassetteBase string
	goldenBase   string
	update       *regexp.Regexp
}

// NewVCR creates a new VCR that can be used to create client factories that
// record cassettes. The base parameter represents the base path to the cassette
// storage (most likely something like testdata/sources); updateRegex is the
// regex given by the user denoting which cassettes should be updated.
func NewVCR(cassetteBase, goldenBase string, updateRegex *string) *VCR {
	var re *regexp.Regexp
	if updateRegex != nil && *updateRegex != "" {
		re = regexp.MustCompile(*updateRegex)
	}

	return &VCR{
		cassetteBase: cassetteBase,
		goldenBase:   goldenBase,
		update:       re,
	}
}

type ClientFactoryOpts struct {
	Name        string
	Middlewares []httpcli.Middleware
	PreUpdate   func(*http.Request) error
	PostUpdate  func(*http.Request, *http.Response) error
}

type ClientFactory struct {
	*httpcli.Factory

	name string
	vcr  *VCR
}

func (vcr *VCR) NewClientFactory(t testing.TB, opts ClientFactoryOpts) *ClientFactory {
	t.Helper()

	rec, err := vcr.newRecorder(opts.Name)
	if err != nil {
		t.Fatalf("error creating recorder: %v", err)
	}

	t.Cleanup(func() {
		if err := rec.Stop(); err != nil {
			t.Errorf("failed to update test data: %v", err)
		}
	})

	mws := opts.Middlewares
	update := vcr.isRecording(opts.Name)
	mws = append(mws, func(cli httpcli.Doer) httpcli.Doer {
		return httpcli.DoerFunc(func(req *http.Request) (*http.Response, error) {
			if update && opts.PreUpdate != nil {
				if err := opts.PreUpdate(req); err != nil {
					return nil, err
				}
			}

			resp, err := cli.Do(req)
			if err != nil {
				return nil, err
			}

			if update && opts.PostUpdate != nil {
				if err := opts.PostUpdate(req, resp); err != nil {
					return nil, err
				}
			}

			return resp, err
		})
	})

	mw := httpcli.NewMiddleware(mws...)
	return &ClientFactory{
		Factory: httpcli.NewFactory(mw, httptestutil.NewRecorderOpt(rec)),
		name:    opts.Name,
		vcr:     vcr,
	}
}

func (vcr *VCR) isRecording(name string) bool {
	return vcr.update != nil && vcr.update.MatchString(name)
}

func (vcr *VCR) newRecorder(name string) (*recorder.Recorder, error) {
	cas := filepath.Join(vcr.cassetteBase, sanitiseTestName(name))

	// TODO: figure out if this is generally appropriate, or too
	// internal/repos-specific.
	return httptestutil.NewRecorder(cas, vcr.isRecording(name), func(i *cassette.Interaction) error {
		// The ratelimit.Monitor type resets its internal timestamp if it's
		// updated with a timestamp in the past. This makes tests ran with
		// recorded interations just wait for a very long time. Removing
		// these headers from the casseste effectively disables rate-limiting
		// in tests which replay HTTP interactions, which is desired behaviour.
		for _, name := range [...]string{
			"RateLimit-Limit",
			"RateLimit-Observed",
			"RateLimit-Remaining",
			"RateLimit-Reset",
			"RateLimit-Resettime",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		} {
			i.Response.Headers.Del(name)
		}

		// Phabricator requests include a token in the form and body.
		ua := i.Request.Headers.Get("User-Agent")
		if strings.Contains(strings.ToLower(ua), extsvc.TypePhabricator) {
			i.Request.Body = ""
			i.Request.Form = nil
		}

		return nil
	})
}

func (cf *ClientFactory) AssertGolden(t testing.TB, want interface{}) {
	t.Helper()
	AssertGolden(t, filepath.Join(cf.vcr.goldenBase, sanitiseTestName(cf.name)), cf.vcr.isRecording(cf.name), want)
}

func sanitiseTestName(name string) string {
	return strings.Replace(name, " ", "-", -1)
}
