package a8n

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/a8n"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
	log15 "gopkg.in/inconshreveable/log15.v2"
)

// NewService returns a Service.
func NewService(store *Store, git GitserverClient, repoResolveRevision repoResolveRevision, cf *httpcli.Factory) *Service {
	return NewServiceWithClock(store, git, repoResolveRevision, cf, func() time.Time {
		return time.Now().UTC().Truncate(time.Microsecond)
	})
}

// NewServiceWithClock returns a Service the given clock used
// to generate timestamps.
func NewServiceWithClock(store *Store, git GitserverClient, repoResolveRevision repoResolveRevision, cf *httpcli.Factory, clock func() time.Time) *Service {
	svc := &Service{
		store:               store,
		git:                 git,
		repoResolveRevision: repoResolveRevision,
		cf:                  cf,
		clock:               clock,
	}
	if svc.repoResolveRevision == nil {
		svc.repoResolveRevision = defaultRepoResolveRevision
	}

	return svc
}

type GitserverClient interface {
	CreateCommitFromPatch(ctx context.Context, req protocol.CreateCommitFromPatchRequest) (string, error)
}

type Service struct {
	store               *Store
	git                 GitserverClient
	repoResolveRevision repoResolveRevision
	cf                  *httpcli.Factory

	clock func() time.Time
}

// repoResolveRevision resolves a Git revspec in a repository and returns the resolved commit ID.
type repoResolveRevision func(context.Context, *repos.Repo, string) (api.CommitID, error)

// defaultRepoResolveRevision is an implementation of repoResolveRevision that talks to gitserver to
// resolve a Git revspec.
var defaultRepoResolveRevision = func(ctx context.Context, repo *repos.Repo, revspec string) (api.CommitID, error) {
	return backend.Repos.ResolveRev(ctx,
		&types.Repo{Name: api.RepoName(repo.Name), ExternalRepo: repo.ExternalRepo},
		revspec,
	)
}

// CreateCampaignPlanFromPatches creates a CampaignPlan and its associated CampaignJobs from patches
// computed by the caller. There is no diff execution or computation performed during creation of
// the CampaignJobs in this case (unlike when using Runner to create a CampaignPlan from a
// specification).
//
// If resolveRevision is nil, a default implementation is used.
func (s *Service) CreateCampaignPlanFromPatches(ctx context.Context, patches []a8n.CampaignPlanPatch) (*a8n.CampaignPlan, error) {
	// Look up all repositories.
	reposStore := repos.NewDBStore(s.store.DB(), sql.TxOptions{})
	repoIDs := make([]uint32, len(patches))
	for i, patch := range patches {
		repoIDs[i] = uint32(patch.Repo)
	}
	allRepos, err := reposStore.ListRepos(ctx, repos.StoreListReposArgs{IDs: repoIDs})
	if err != nil {
		return nil, err
	}
	reposByID := make(map[uint32]*repos.Repo, len(patches))
	for _, repo := range allRepos {
		reposByID[repo.ID] = repo
	}

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Done(&err)

	plan := &a8n.CampaignPlan{
		CampaignType: patchCampaignType,
		Arguments:    "", // intentionally empty to avoid needless duplication with CampaignJob diffs
	}

	err = tx.CreateCampaignPlan(ctx, plan)
	if err != nil {
		return nil, err
	}

	for _, patch := range patches {
		repo := reposByID[uint32(patch.Repo)]
		if repo == nil {
			return nil, fmt.Errorf("repository ID %d not found", patch.Repo)
		}
		if !a8n.IsRepoSupported(&repo.ExternalRepo) {
			continue
		}

		commit, err := s.repoResolveRevision(ctx, repo, patch.BaseRevision)
		if err != nil {
			return nil, errors.Wrapf(err, "repository %q", repo.Name)
		}

		job := &a8n.CampaignJob{
			CampaignPlanID: plan.ID,
			RepoID:         int32(patch.Repo),
			BaseRef:        patch.BaseRevision,
			Rev:            commit,
			Diff:           patch.Patch,
			StartedAt:      s.clock(),
			FinishedAt:     s.clock(),
		}
		if err := tx.CreateCampaignJob(ctx, job); err != nil {
			return nil, err
		}
	}

	return plan, nil
}

// CreateCampaign creates the Campaign. When a CampaignPlanID is set, it also
// creates one ChangesetJob for each CampaignJob belonging to the respective
// CampaignPlan, together with the Campaign in a transaction.
func (s *Service) CreateCampaign(ctx context.Context, c *a8n.Campaign) error {
	var err error
	tr, ctx := trace.New(ctx, "Service.CreateCampaign", fmt.Sprintf("Name: %q", c.Name))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()
	tx, err := s.store.Transact(ctx)
	if err != nil {
		return err
	}
	defer tx.Done(&err)

	if err := tx.CreateCampaign(ctx, c); err != nil {
		return err
	}

	if c.CampaignPlanID == 0 {
		return nil
	}

	jobs, _, err := tx.ListCampaignJobs(ctx, ListCampaignJobsOpts{
		CampaignPlanID: c.CampaignPlanID,
		Limit:          -1,
		OnlyFinished:   true,
		OnlyWithDiff:   true,
	})
	if err != nil {
		return err
	}

	for _, job := range jobs {
		changesetJob := &a8n.ChangesetJob{
			CampaignID:    c.ID,
			CampaignJobID: job.ID,
		}
		err = tx.CreateChangesetJob(ctx, changesetJob)
		if err != nil {
			return err
		}
	}

	return nil
}

// RunChangesetJobs will run all the changeset jobs for the supplied campaign.
// It is idempotent and jobs that have already completed will not be rerun.
func (s *Service) RunChangesetJobs(ctx context.Context, c *a8n.Campaign) error {
	var err error
	tr, ctx := trace.New(ctx, "Service.RunChangesetJobs", fmt.Sprintf("Campaign: %q", c.Name))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()
	jobs, _, err := s.store.ListChangesetJobs(ctx, ListChangesetJobsOpts{
		CampaignID: c.ID,
		Limit:      -1,
	})
	if err != nil {
		return err
	}

	errs := &multierror.Error{}
	for _, job := range jobs {
		err := s.runChangesetJob(ctx, c, job)
		if err != nil {
			err = errors.Wrapf(err, "ChangesetJob %d", job.ID)
			errs = multierror.Append(errs, err)
		}
	}

	return errs.ErrorOrNil()
}

func (s *Service) runChangesetJob(
	ctx context.Context,
	c *a8n.Campaign,
	job *a8n.ChangesetJob,
) (err error) {
	tr, ctx := trace.New(ctx, "service.runChangeSetJob", fmt.Sprintf("job_id: %d", job.ID))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()
	tr.LogFields(log.Bool("completed", job.SuccessfullyCompleted()), log.Int64("job_id", job.ID), log.Int64("campaign_id", c.ID))

	if job.SuccessfullyCompleted() {
		log15.Info("ChangesetJob already completed", "id", job.ID)
		return nil
	}

	defer func() {
		if err != nil {
			job.Error = err.Error()
		}
		job.FinishedAt = s.clock()

		if e := s.store.UpdateChangesetJob(ctx, job); e != nil {
			if err == nil {
				err = e
			} else {
				err = multierror.Append(err, e)
			}
		}
	}()

	job.StartedAt = s.clock()

	campaignJob, err := s.store.GetCampaignJob(ctx, GetCampaignJobOpts{ID: job.CampaignJobID})
	if err != nil {
		return err
	}

	reposStore := repos.NewDBStore(s.store.DB(), sql.TxOptions{})
	rs, err := reposStore.ListRepos(ctx, repos.StoreListReposArgs{IDs: []uint32{uint32(campaignJob.RepoID)}})
	if err != nil {
		return err
	}
	if len(rs) != 1 {
		return errors.Errorf("repo not found: %d", campaignJob.RepoID)
	}
	repo := rs[0]

	headRefName := fmt.Sprintf("sourcegraph/%s-%d", git.HumanReadableBranchName(c.Name), c.CreatedAt.Unix())

	_, err = s.git.CreateCommitFromPatch(ctx, protocol.CreateCommitFromPatchRequest{
		Repo:       api.RepoName(repo.Name),
		BaseCommit: campaignJob.Rev,
		// IMPORTANT: We add a trailing newline here, otherwise `git apply`
		// will fail with "corrupt patch at line <N>" where N is the last line.
		Patch:     campaignJob.Diff + "\n",
		TargetRef: headRefName,
		CommitInfo: protocol.PatchCommitInfo{
			Message:     c.Name,
			AuthorName:  "Sourcegraph Bot",
			AuthorEmail: "automation@sourcegraph.com",
			Date:        job.StartedAt,
		},
		// We use unified diffs, not git diffs, which means they're missing the
		// `a/` and `/b` filename prefixes. `-p0` tells `git apply` to not
		// expect and strip prefixes.
		// Since we also produce diffs manually, we might not have context lines,
		// so we need to disable that check with `--unidiff-zero`.
		GitApplyArgs: []string{"-p0", "--unidiff-zero"},
		Push:         true,
	})

	if err != nil {
		if diffErr, ok := err.(*protocol.CreateCommitFromPatchError); ok {
			return errors.Errorf("creating commit from patch for repo %q: %v (command: %q)", diffErr.RepositoryName, diffErr.Err, diffErr.Command)
		}
		return err
	}

	var codeHost *repos.CodeHost
	{
		args := repos.StoreListCodeHostsArgs{IDs: repo.CodeHostIDs()}

		es, err := reposStore.ListCodeHosts(ctx, args)
		if err != nil {
			return err
		}

		for _, e := range es {
			cfg, err := e.Configuration()
			if err != nil {
				return err
			}

			switch cfg := cfg.(type) {
			case *schema.GitHubConnection:
				if cfg.Token != "" {
					codeHost = e
				}
			case *schema.BitbucketServerConnection:
				if cfg.Token != "" {
					codeHost = e
				}
			}
			if codeHost != nil {
				break
			}
		}
	}

	if codeHost == nil {
		return errors.Errorf("no external services found for repo %q", repo.Name)
	}

	src, err := repos.NewSource(codeHost, s.cf)
	if err != nil {
		return err
	}

	baseRef := "refs/heads/master"
	if campaignJob.BaseRef != "" {
		baseRef = campaignJob.BaseRef
	}

	body := c.Description
	if campaignJob.Description != "" {
		body += "\n\n---\n\n" + campaignJob.Description
	}

	cs := repos.Changeset{
		Title:   c.Name,
		Body:    body,
		BaseRef: baseRef,
		HeadRef: git.EnsureRefPrefix(headRefName),
		Repo:    repo,
		Changeset: &a8n.Changeset{
			RepoID:      int32(repo.ID),
			CampaignIDs: []int64{job.CampaignID},
		},
	}

	ccs, ok := src.(repos.ChangesetSource)
	if !ok {
		return errors.Errorf("creating changesets on code host of repo %q is not implemented", repo.Name)
	}

	err = ccs.CreateChangeset(ctx, &cs)
	if err != nil {
		return errors.Wrap(err, "creating changeset")
	}

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return err
	}
	defer tx.Done(&err)

	if err = tx.CreateChangesets(ctx, cs.Changeset); err != nil {
		if _, ok := err.(AlreadyExistError); !ok {
			return err
		}
		// Changeset already exists so continue
	}

	c.ChangesetIDs = append(c.ChangesetIDs, cs.Changeset.ID)
	if err = tx.UpdateCampaign(ctx, c); err != nil {
		return err
	}

	job.ChangesetID = cs.Changeset.ID
	return tx.UpdateChangesetJob(ctx, job)
}

// CloseCampaign closes the Campaign with the given ID if it has not been closed yet.
func (s *Service) CloseCampaign(ctx context.Context, id int64, closeChangesets bool) (campaign *a8n.Campaign, err error) {
	traceTitle := fmt.Sprintf("campaign: %d, closeChangesets: %t", id, closeChangesets)
	tr, ctx := trace.New(ctx, "service.CloseCampaign", traceTitle)
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Done(&err)

	campaign, err = tx.GetCampaign(ctx, GetCampaignOpts{ID: id})
	if err != nil {
		return nil, errors.Wrap(err, "getting campaign")
	}

	if !campaign.ClosedAt.IsZero() {
		return campaign, nil
	}

	campaign.ClosedAt = time.Now().UTC()

	if err = tx.UpdateCampaign(ctx, campaign); err != nil {
		return nil, err
	}

	if closeChangesets {
		go func() {
			ctx := trace.ContextWithTrace(context.Background(), tr)

			cs, _, err := s.store.ListChangesets(ctx, ListChangesetsOpts{
				CampaignID: campaign.ID,
				Limit:      -1,
			})
			if err != nil {
				log15.Error("ListChangesets", "err", err)
				return
			}

			// Close only the changesets that are open
			err = s.CloseOpenChangesets(ctx, cs)
			if err != nil {
				log15.Error("CloseCampaignChangesets", "err", err)
			}
		}()
	}

	return campaign, nil
}

// DeleteCampaign deletes the Campaign with the given ID if it hasn't been
// deleted yet. If closeChangesets is true, the changesets associated with the
// Campaign will be closed on the codehosts.
func (s *Service) DeleteCampaign(ctx context.Context, id int64, closeChangesets bool) (err error) {
	traceTitle := fmt.Sprintf("campaign: %d, closeChangesets: %t", id, closeChangesets)
	tr, ctx := trace.New(ctx, "service.DeleteCampaign", traceTitle)
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	// If we don't have to close the changesets, we can simply delete the
	// Campaign and return. The triggers in the database will remove the
	// campaign's ID from the changesets' CampaignIDs.
	if !closeChangesets {
		return s.store.DeleteCampaign(ctx, id)
	}

	// First load the Changesets with the given campaignID, before deleting
	// the campaign would remove the association.
	cs, _, err := s.store.ListChangesets(ctx, ListChangesetsOpts{
		CampaignID: id,
		Limit:      -1,
	})
	if err != nil {
		return err
	}

	// Remove the association manually, since we'll update the Changesets in
	// the database, after closing them and we can't update them with an
	// invalid CampaignID.
	for _, c := range cs {
		c.RemoveCampaignID(id)
	}

	err = s.store.DeleteCampaign(ctx, id)
	if err != nil {
		return err
	}

	go func() {
		ctx := trace.ContextWithTrace(context.Background(), tr)
		err := s.CloseOpenChangesets(ctx, cs)
		if err != nil {
			log15.Error("CloseCampaignChangesets", "err", err)
		}
	}()

	return nil
}

// CloseOpenChangesets closes the given Changesets on their respective codehosts and syncs them.
func (s *Service) CloseOpenChangesets(ctx context.Context, cs []*a8n.Changeset) (err error) {
	cs = selectChangesets(cs, func(c *a8n.Changeset) bool {
		s, err := c.State()
		if err != nil {
			log15.Warn("could not determine changeset state", "err", err)
			return false
		}
		return s == a8n.ChangesetStateOpen
	})

	if len(cs) == 0 {
		return nil
	}

	reposStore := repos.NewDBStore(s.store.DB(), sql.TxOptions{})
	syncer := ChangesetSyncer{
		ReposStore:  reposStore,
		Store:       s.store,
		HTTPFactory: s.cf,
	}

	bySource, err := syncer.GroupChangesetsBySource(ctx, cs...)
	if err != nil {
		return err
	}

	errs := &multierror.Error{}
	for _, s := range bySource {
		for _, c := range s.Changesets {
			if err := s.CloseChangeset(ctx, c); err != nil {
				errs = multierror.Append(errs, err)
			}
		}
	}

	if len(errs.Errors) != 0 {
		return errs
	}

	// Here we need to sync the just-closed changesets (even though
	// CloseChangesets updates the given Changesets too), because closing a
	// Changeset often produces a ChangesetEvent on the codehost and if we were
	// to close the Changesets and not update the events (which is what
	// SyncChangesetsWithSources does) our burndown chart will be outdated
	// until the next run of a8n.Syncer.
	return syncer.SyncChangesetsWithSources(ctx, bySource)
}

func selectChangesets(cs []*a8n.Changeset, predicate func(*a8n.Changeset) bool) []*a8n.Changeset {
	i := 0
	for _, c := range cs {
		if predicate(c) {
			cs[i] = c
			i++
		}
	}

	return cs[:i]
}
