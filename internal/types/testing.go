package types

import (
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/awscodecommit"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitolite"
	"github.com/sourcegraph/sourcegraph/internal/timeutil"
)

func MakeRepo(name, serviceID, serviceType string, services ...*ExternalService) *Repo {
	clock := timeutil.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	repo := Repo{
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "1234",
			ServiceType: serviceType,
			ServiceID:   serviceID,
		},
		Name:        api.RepoName(name),
		URI:         name,
		Description: "The description",
		CreatedAt:   now,
		Sources:     make(map[string]*SourceInfo),
	}

	for _, svc := range services {
		repo.Sources[svc.URN()] = &SourceInfo{
			ID: svc.URN(),
		}
	}

	return &repo
}

// MakeGithubRepo returns a configured Github repository.
func MakeGithubRepo(services ...*ExternalService) *Repo {
	repo := MakeRepo("github.com/foo/bar", "http://github.com", extsvc.TypeGitHub, services...)
	repo.Metadata = new(github.Repository)
	return repo
}

// MakeGitlabRepo returns a configured Gitlab repository.
func MakeGitlabRepo(services ...*ExternalService) *Repo {
	repo := MakeRepo("gitlab.com/foo/bar", "http://gitlab.com", extsvc.TypeGitLab, services...)
	repo.Metadata = new(gitlab.Project)
	return repo
}

// MakeBitbucketServerRepo returns a configured Bitbucket Server repository.
func MakeBitbucketServerRepo(services ...*ExternalService) *Repo {
	repo := MakeRepo("bitbucketserver.mycorp.com/foo/bar", "http://bitbucketserver.mycorp.com", extsvc.TypeBitbucketServer, services...)
	repo.Metadata = new(bitbucketserver.Repo)
	return repo
}

// MakeAWSCodeCommitRepo returns a configured AWS Code Commit repository.
func MakeAWSCodeCommitRepo(services ...*ExternalService) *Repo {
	repo := MakeRepo("git-codecommit.us-west-1.amazonaws.com/stripe-go", "arn:aws:codecommit:us-west-1:999999999999:", extsvc.KindAWSCodeCommit, services...)
	repo.Metadata = new(awscodecommit.Repository)
	return repo
}

// MakeOtherRepo returns a configured repository from a custom host.
func MakeOtherRepo(services ...*ExternalService) *Repo {
	repo := MakeRepo("git-host.com/org/foo", "https://git-host.com/", extsvc.KindOther, services...)
	repo.Metadata = new(extsvc.OtherRepoMetadata)
	return repo
}

// MakeGitoliteRepo returns a configured Gitolite repository.
func MakeGitoliteRepo(services ...*ExternalService) *Repo {
	repo := MakeRepo("gitolite.mycorp.com/bar", "git@gitolite.mycorp.com", extsvc.KindGitolite, services...)
	repo.Metadata = new(gitolite.Repo)
	return repo
}

// GenerateRepos takes a list of base repos and generates n ones with different names.
func GenerateRepos(n int, base ...*Repo) Repos {
	if len(base) == 0 {
		return nil
	}

	rs := make(Repos, 0, n)
	for i := 0; i < n; i++ {
		id := strconv.Itoa(i)
		r := base[i%len(base)].Clone()
		r.Name += api.RepoName(id)
		r.ExternalRepo.ID += id
		rs = append(rs, r)
	}
	return rs
}

// MakeExternalServices creates one configured external service per kind and returns the list.
func MakeExternalServices() ExternalServices {
	clock := timeutil.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	githubSvc := ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "Github - Test",
		Config:      `{"url": "https://github.com", "token": "abc", "repositoryQuery": ["none"]}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	gitlabSvc := ExternalService{
		Kind:        extsvc.KindGitLab,
		DisplayName: "GitLab - Test",
		Config:      `{"url": "https://gitlab.com", "token": "abc", "projectQuery": ["projects?membership=true&archived=no"]}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	bitbucketServerSvc := ExternalService{
		Kind:        extsvc.KindBitbucketServer,
		DisplayName: "Bitbucket Server - Test",
		Config:      `{"url": "https://bitbucket.com", "username": "foo", "token": "abc", "repositoryQuery": ["none"]}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	bitbucketCloudSvc := ExternalService{
		Kind:        extsvc.KindBitbucketCloud,
		DisplayName: "Bitbucket Cloud - Test",
		Config:      `{"url": "https://bitbucket.com", "username": "foo", "appPassword": "abc"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	awsSvc := ExternalService{
		Kind:        extsvc.KindAWSCodeCommit,
		DisplayName: "AWS Code - Test",
		Config:      `{"region": "eu-west-1", "accessKeyID": "key", "secretAccessKey": "secret", "gitCredentials": {"username": "foo", "password": "bar"}}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	otherSvc := ExternalService{
		Kind:        extsvc.KindOther,
		DisplayName: "Other - Test",
		Config:      `{"url": "https://other.com", "repos": ["none"]}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	gitoliteSvc := ExternalService{
		Kind:        extsvc.KindGitolite,
		DisplayName: "Gitolite - Test",
		Config:      `{"prefix": "foo", "host": "bar"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return []*ExternalService{
		&githubSvc,
		&gitlabSvc,
		&bitbucketServerSvc,
		&bitbucketCloudSvc,
		&awsSvc,
		&otherSvc,
		&gitoliteSvc,
	}
}

// GenerateExternalServices takes a list of base external services and generates n ones with different names.
func GenerateExternalServices(n int, base ...*ExternalService) ExternalServices {
	if len(base) == 0 {
		return nil
	}
	es := make(ExternalServices, 0, n)
	for i := 0; i < n; i++ {
		id := strconv.Itoa(i)
		r := base[i%len(base)].Clone()
		r.DisplayName += id
		es = append(es, r)
	}
	return es
}

// ExternalServicesToMap is a helper function that returns a map whose key is the external service kind.
// If two external services have the same kind, only the last one will be stored in the map.
func ExternalServicesToMap(es ExternalServices) map[string]*ExternalService {
	m := make(map[string]*ExternalService)

	for _, svc := range es {
		m[svc.Kind] = svc
	}

	return m
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
	RepoName                  func(api.RepoName) func(*Repo)
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
	RepoName: func(name api.RepoName) func(*Repo) {
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
				r.Sources[src] = &SourceInfo{ID: src, CloneURL: "clone-url"}
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
			if diff := cmp.Diff(want, have, cmpopts.IgnoreFields(Repo{}, "CreatedAt", "UpdatedAt")); diff != "" {
				t.Errorf("repos (-want +got): %s", diff)
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
			if diff := cmp.Diff(want, have); diff != "" {
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
			if diff := cmp.Diff(want, have); diff != "" {
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
			if diff := cmp.Diff(want, have); diff != "" {
				t.Errorf("external services (-want +got): %s", cmp.Diff(want, have))
			}
		}
	},
}
