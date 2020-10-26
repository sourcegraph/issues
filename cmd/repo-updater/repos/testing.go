package repos

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-multierror"
	"github.com/sourcegraph/sourcegraph/internal/api"
)

// NewFakeSourcer returns a Sourcer which always returns the given error and sources,
// ignoring the given external services.
func NewFakeSourcer(err error, srcs ...Source) Sourcer {
	return func(svcs ...*ExternalService) (Sources, error) {
		var errs *multierror.Error

		if err != nil {
			for _, svc := range svcs {
				errs = multierror.Append(errs, &SourceError{Err: err, ExtSvc: svc})
			}
			if len(svcs) == 0 {
				errs = multierror.Append(errs, &SourceError{Err: err, ExtSvc: nil})
			}
		}

		return srcs, errs.ErrorOrNil()
	}
}

// FakeSource is a fake implementation of Source to be used in tests.
type FakeSource struct {
	svc   *ExternalService
	repos []*Repo
	err   error
}

// NewFakeSource returns an instance of FakeSource with the given urn, error
// and repos.
func NewFakeSource(svc *ExternalService, err error, rs ...*Repo) *FakeSource {
	return &FakeSource{svc: svc, err: err, repos: rs}
}

// ListRepos returns the Repos that FakeSource was instantiated with
// as well as the error, if any.
func (s FakeSource) ListRepos(ctx context.Context, results chan SourceResult) {
	if s.err != nil {
		results <- SourceResult{Source: s, Err: s.err}
		return
	}

	for _, r := range s.repos {
		results <- SourceResult{Source: s, Repo: r.With(Opt.RepoSources(s.svc.URN()))}
	}
}

// ExternalServices returns a singleton slice containing the external service.
func (s FakeSource) ExternalServices() ExternalServices {
	return ExternalServices{s.svc}
}

//
// Assertions
//

// A ReposAssertion performs an assertion on the given Repos.
type ReposAssertion func(testing.TB, Repos)

// An ExternalServicesAssertion performs an assertion on the given
// ExternalServices.
type ExternalServicesAssertion func(testing.TB, ExternalServices)

// Assert contains assertion functions to be used in tests.
var Assert = struct {
	ReposEqual                func(...*Repo) ReposAssertion
	ReposOrderedBy            func(func(a, b *Repo) bool) ReposAssertion
	ExternalServicesEqual     func(...*ExternalService) ExternalServicesAssertion
	ExternalServicesOrderedBy func(func(a, b *ExternalService) bool) ExternalServicesAssertion
}{
	ReposEqual: func(rs ...*Repo) ReposAssertion {
		want := append(Repos{}, rs...).With(Opt.RepoID(0))
		return func(t testing.TB, have Repos) {
			t.Helper()
			// Exclude auto-generated IDs from equality tests
			have = append(Repos{}, have...).With(Opt.RepoID(0))
			if !reflect.DeepEqual(have, want) {
				t.Errorf("repos (-want +got): %s", cmp.Diff(want, have))
			}
		}
	},
	ReposOrderedBy: func(ord func(a, b *Repo) bool) ReposAssertion {
		return func(t testing.TB, have Repos) {
			t.Helper()
			want := have.Clone()
			sort.Slice(want, func(i, j int) bool {
				return ord(want[i], want[j])
			})
			if !reflect.DeepEqual(have, want) {
				t.Errorf("repos (-want +got): %s", cmp.Diff(want, have))
			}
		}
	},
	ExternalServicesEqual: func(es ...*ExternalService) ExternalServicesAssertion {
		want := append(ExternalServices{}, es...).With(Opt.ExternalServiceID(0))
		return func(t testing.TB, have ExternalServices) {
			t.Helper()
			// Exclude auto-generated IDs from equality tests
			have = append(ExternalServices{}, have...).With(Opt.ExternalServiceID(0))
			if !reflect.DeepEqual(have, want) {
				t.Errorf("external services (-want +got): %s", cmp.Diff(want, have))
			}
		}
	},
	ExternalServicesOrderedBy: func(ord func(a, b *ExternalService) bool) ExternalServicesAssertion {
		return func(t testing.TB, have ExternalServices) {
			t.Helper()
			want := have.Clone()
			sort.Slice(want, func(i, j int) bool {
				return ord(want[i], want[j])
			})
			if !reflect.DeepEqual(have, want) {
				t.Errorf("external services (-want +got): %s", cmp.Diff(want, have))
			}
		}
	},
}

//
// Functional options
//

// Opt contains functional options to be used in tests.
var Opt = struct {
	ExternalServiceID         func(int64) func(*ExternalService)
	ExternalServiceModifiedAt func(time.Time) func(*ExternalService)
	ExternalServiceDeletedAt  func(time.Time) func(*ExternalService)
	RepoID                    func(api.RepoID) func(*Repo)
	RepoName                  func(string) func(*Repo)
	RepoCreatedAt             func(time.Time) func(*Repo)
	RepoModifiedAt            func(time.Time) func(*Repo)
	RepoDeletedAt             func(time.Time) func(*Repo)
	RepoSources               func(...string) func(*Repo)
	RepoMetadata              func(interface{}) func(*Repo)
	RepoExternalID            func(string) func(*Repo)
}{
	ExternalServiceID: func(n int64) func(*ExternalService) {
		return func(e *ExternalService) {
			e.ID = n
		}
	},
	ExternalServiceModifiedAt: func(ts time.Time) func(*ExternalService) {
		return func(e *ExternalService) {
			e.UpdatedAt = ts
			e.DeletedAt = time.Time{}
		}
	},
	ExternalServiceDeletedAt: func(ts time.Time) func(*ExternalService) {
		return func(e *ExternalService) {
			e.UpdatedAt = ts
			e.DeletedAt = ts
		}
	},
	RepoID: func(n api.RepoID) func(*Repo) {
		return func(r *Repo) {
			r.ID = n
		}
	},
	RepoName: func(name string) func(*Repo) {
		return func(r *Repo) {
			r.Name = name
		}
	},
	RepoCreatedAt: func(ts time.Time) func(*Repo) {
		return func(r *Repo) {
			r.CreatedAt = ts
			r.UpdatedAt = ts
			r.DeletedAt = time.Time{}
		}
	},
	RepoModifiedAt: func(ts time.Time) func(*Repo) {
		return func(r *Repo) {
			r.UpdatedAt = ts
			r.DeletedAt = time.Time{}
		}
	},
	RepoDeletedAt: func(ts time.Time) func(*Repo) {
		return func(r *Repo) {
			r.UpdatedAt = ts
			r.DeletedAt = ts
			r.Sources = map[string]*SourceInfo{}
		}
	},
	RepoSources: func(srcs ...string) func(*Repo) {
		return func(r *Repo) {
			r.Sources = map[string]*SourceInfo{}
			for _, src := range srcs {
				r.Sources[src] = &SourceInfo{ID: src}
			}
		}
	},
	RepoMetadata: func(md interface{}) func(*Repo) {
		return func(r *Repo) {
			r.Metadata = md
		}
	},
	RepoExternalID: func(id string) func(*Repo) {
		return func(r *Repo) {
			r.ExternalRepo.ID = id
		}
	},
}

// FakeClock provides a controllable clock for use in tests.
type FakeClock struct {
	epoch time.Time
	step  time.Duration
	steps int
}

// NewFakeClock returns a FakeClock instance that starts telling time at the given epoch.
// Every invocation of Now adds step amount of time to the clock.
func NewFakeClock(epoch time.Time, step time.Duration) FakeClock {
	return FakeClock{epoch: epoch, step: step}
}

// Now returns the current fake time and advances the clock "step" amount of time.
func (c *FakeClock) Now() time.Time {
	c.steps++
	return c.Time(c.steps)
}

// Time tells the time at the given step from the provided epoch.
func (c FakeClock) Time(step int) time.Time {
	// We truncate to microsecond precision because Postgres' timestamptz type
	// doesn't handle nanoseconds.
	return c.epoch.Add(time.Duration(step) * c.step).UTC().Truncate(time.Microsecond)
}
