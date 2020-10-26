package repos

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
)

// A Sourcer converts the given ExternalServices to Sources
// whose yielded Repos should be synced.
type Sourcer func(...*ExternalService) (Sources, error)

// NewSourcer returns a Sourcer that converts the given ExternalServices
// into Sources that use the provided httpcli.Factory to create the
// http.Clients needed to contact the respective upstream code host APIs.
//
// Deleted external services are ignored.
//
// The provided decorator functions will be applied to each Source.
func NewSourcer(cf *httpcli.Factory, decs ...func(Source) Source) Sourcer {
	return func(svcs ...*ExternalService) (Sources, error) {
		srcs := make([]Source, 0, len(svcs))
		var errs *multierror.Error

		for _, svc := range svcs {
			if svc.IsDeleted() {
				continue
			}

			src, err := NewSource(svc, cf)
			if err != nil {
				errs = multierror.Append(errs, &SourceError{Err: err, ExtSvc: svc})
				continue
			}

			for _, dec := range decs {
				src = dec(src)
			}

			srcs = append(srcs, src)
		}

		return srcs, errs.ErrorOrNil()
	}
}

// NewSource returns a repository yielding Source from the given ExternalService configuration.
func NewSource(svc *ExternalService, cf *httpcli.Factory) (Source, error) {
	switch strings.ToUpper(svc.Kind) {
	case extsvc.KindGitHub:
		return NewGithubSource(svc, cf)
	case extsvc.KindGitLab:
		return NewGitLabSource(svc, cf)
	case extsvc.KindBitbucketServer:
		return NewBitbucketServerSource(svc, cf)
	case extsvc.KindBitbucketCloud:
		return NewBitbucketCloudSource(svc, cf)
	case extsvc.KindGitolite:
		return NewGitoliteSource(svc, cf)
	case extsvc.KindPhabricator:
		return NewPhabricatorSource(svc, cf)
	case extsvc.KindAWSCodeCommit:
		return NewAWSCodeCommitSource(svc, cf)
	case extsvc.KindOther:
		return NewOtherSource(svc, cf)
	default:
		panic(fmt.Sprintf("source not implemented for external service kind %q", svc.Kind))
	}
}

// A Source yields repositories to be stored and analysed by Sourcegraph.
// Successive calls to its ListRepos method may yield different results.
type Source interface {
	// ListRepos sends all the repos a source yields over the passed in channel
	// as SourceResults
	ListRepos(context.Context, chan SourceResult)
	// ExternalServices returns the ExternalServices for the Source.
	ExternalServices() ExternalServices
}

// A DraftChangesetSource can create draft changesets and undraft them.
type DraftChangesetSource interface {
	// CreateDraftChangeset will create the Changeset on the source. If it already
	// exists, *Changeset will be populated and the return value will be
	// true.
	CreateDraftChangeset(context.Context, *Changeset) (bool, error)
	// UndraftChangeset will update the Changeset on the source to be not in draft mode anymore.
	UndraftChangeset(context.Context, *Changeset) error
}

// A ChangesetSource can load the latest state of a list of Changesets.
type ChangesetSource interface {
	// LoadChangeset loads the given Changeset from the source and updates it.
	// If the Changeset could not be found on the source, a ChangesetNotFoundError is returned.
	LoadChangeset(context.Context, *Changeset) error
	// CreateChangeset will create the Changeset on the source. If it already
	// exists, *Changeset will be populated and the return value will be
	// true.
	CreateChangeset(context.Context, *Changeset) (bool, error)
	// CloseChangeset will close the Changeset on the source, where "close"
	// means the appropriate final state on the codehost (e.g. "declined" on
	// Bitbucket Server).
	CloseChangeset(context.Context, *Changeset) error
	// UpdateChangeset can update Changesets.
	UpdateChangeset(context.Context, *Changeset) error
	// ReopenChangeset will reopen the Changeset on the source, if it's closed.
	// If not, it's a noop.
	ReopenChangeset(context.Context, *Changeset) error
}

// ChangesetNotFoundError is returned by LoadChangeset if the changeset
// could not be found on the codehost.
type ChangesetNotFoundError struct {
	Changeset *Changeset
}

func (e ChangesetNotFoundError) Error() string {
	return fmt.Sprintf("Changeset with external ID %s not found", e.Changeset.Changeset.ExternalID)
}

// A SourceResult is sent by a Source over a channel for each repository it
// yields when listing repositories
type SourceResult struct {
	// Source points to the Source that produced this result
	Source Source
	// Repo is the repository that was listed by the Source
	Repo *Repo
	// Err is only set in case the Source ran into an error when listing repositories
	Err error
}

type SourceError struct {
	Err    error
	ExtSvc *ExternalService
}

func (s *SourceError) Error() string {
	if multiErr, ok := s.Err.(*multierror.Error); ok {
		// Create new Error with custom formatter. Do not mutate otherwise can
		// race with other callers of Error.
		return (&multierror.Error{
			Errors:      multiErr.Errors,
			ErrorFormat: sourceErrorFormatFunc,
		}).Error()
	}
	return s.Err.Error()
}

func sourceErrorFormatFunc(es []error) string {
	if len(es) == 1 {
		return es[0].Error()
	}

	points := make([]string, len(es))
	for i, err := range es {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d errors occurred:\n\t%s\n\n",
		len(es), strings.Join(points, "\n\t"))
}

// Sources is a list of Sources that implements the Source interface.
type Sources []Source

// ListRepos lists all the repos of all the sources and returns the
// aggregate result.
func (srcs Sources) ListRepos(ctx context.Context, results chan SourceResult) {
	if len(srcs) == 0 {
		return
	}

	// Group sources by external service kind so that we execute requests
	// serially to each code host. This is to comply with abuse rate limits of GitHub,
	// but we do it for any source to be conservative.
	// See https://developer.github.com/v3/guides/best-practices-for-integrators/#dealing-with-abuse-rate-limits)

	var wg sync.WaitGroup
	for _, sources := range group(srcs) {
		wg.Add(1)
		go func(sources Sources) {
			defer wg.Done()
			for _, src := range sources {
				src.ListRepos(ctx, results)
			}
		}(sources)
	}

	wg.Wait()
}

// ExternalServices returns the ExternalServices from the given Sources.
func (srcs Sources) ExternalServices() ExternalServices {
	es := make(ExternalServices, 0, len(srcs))
	for _, src := range srcs {
		es = append(es, src.ExternalServices()...)
	}
	return es
}

type multiSource interface {
	Sources() []Source
}

// Sources returns the underlying Sources.
func (srcs Sources) Sources() []Source { return srcs }

func group(srcs []Source) map[string]Sources {
	groups := make(map[string]Sources)

	for _, src := range srcs {
		if ms, ok := src.(multiSource); ok {
			for kind, ss := range group(ms.Sources()) {
				groups[kind] = append(groups[kind], ss...)
			}
		} else if es := src.ExternalServices(); len(es) > 1 {
			err := errors.Errorf("Source %#v has many external services and isn't a multiSource", src)
			panic(err)
		} else {
			kind := es[0].Kind
			groups[kind] = append(groups[kind], src)
		}
	}

	return groups
}

// listAll calls ListRepos on the given Source and collects the SourceResults
// the Source sends over a channel into a slice of *Repo and a single error
func listAll(ctx context.Context, src Source, onSourced ...func(*Repo) error) ([]*Repo, error) {
	results := make(chan SourceResult)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		src.ListRepos(ctx, results)
		close(results)
	}()

	var (
		repos []*Repo
		errs  *multierror.Error
	)

	for res := range results {
		if res.Err != nil {
			for _, extSvc := range res.Source.ExternalServices() {
				errs = multierror.Append(errs, &SourceError{Err: res.Err, ExtSvc: extSvc})
			}
			continue
		}
		for _, o := range onSourced {
			err := o(res.Repo)
			if err != nil {
				// onSourced has returned an error indicating we should stop sourcing.
				// We're being defensive here in case one of the Source implementations doesn't handle
				// cancellation correctly. We'll continue to drain the results to ensure we don't
				// have a goroutine leak.
				cancel()
				errs = multierror.Append(errs, err)
			}
		}
		repos = append(repos, res.Repo)
	}

	return repos, errs.ErrorOrNil()
}
