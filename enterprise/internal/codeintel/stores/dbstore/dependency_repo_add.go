package dbstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/keegancsmith/sqlf"
	"github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/batch"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

type DependencyRepoAddingJob struct {
	ID             int        `json:"id"`
	State          string     `json:"state"`
	FailureMessage *string    `json:"failureMessage"`
	StartedAt      *time.Time `json:"startedAt"`
	FinishedAt     *time.Time `json:"finishedAt"`
	ProcessAfter   *time.Time `json:"processAfter"`
	NumResets      int        `json:"numResets"`
	NumFailures    int        `json:"numFailures"`
	UploadID       int        `json:"uploadId"`
}

func (u DependencyRepoAddingJob) RecordID() int {
	return u.ID
}

func scanDependencyRepoAddingJobs(rows *sql.Rows, queryErr error) (_ []DependencyRepoAddingJob, err error) {
	if queryErr != nil {
		return nil, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	var jobs []DependencyRepoAddingJob
	for rows.Next() {
		var job DependencyRepoAddingJob
		if err := rows.Scan(
			&job.ID,
			&job.State,
			&job.FailureMessage,
			&job.StartedAt,
			&job.FinishedAt,
			&job.ProcessAfter,
			&job.NumResets,
			&job.NumFailures,
			&job.UploadID,
		); err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

var dependencyRepoAddingJobColumns = []*sqlf.Query{
	sqlf.Sprintf("j.id"),
	sqlf.Sprintf("j.state"),
	sqlf.Sprintf("j.failure_message"),
	sqlf.Sprintf("j.started_at"),
	sqlf.Sprintf("j.finished_at"),
	sqlf.Sprintf("j.process_after"),
	sqlf.Sprintf("j.num_resets"),
	sqlf.Sprintf("j.num_failures"),
	sqlf.Sprintf("j.upload_id"),
}

// scanFirstDependencyRepoAddingJob scans a slice of dependency RepoAdding jobs from the return
// value of `*Store.query` and returns the first.
func scanFirstDependencyRepoAddingJob(rows *sql.Rows, err error) (DependencyRepoAddingJob, bool, error) {
	jobs, err := scanDependencyRepoAddingJobs(rows, err)
	if err != nil || len(jobs) == 0 {
		return DependencyRepoAddingJob{}, false, err
	}
	return jobs[0], true, nil
}

// scanFirstDependencyRepoAddingJobRecord scans a slice of dependency RepoAdding jobs from the
// return value of `*Store.query` and returns the first.
func scanFirstDependencyRepoAddingJobRecord(rows *sql.Rows, err error) (workerutil.Record, bool, error) {
	return scanFirstDependencyRepoAddingJob(rows, err)
}

func (s *Store) InsertDependencyRepoAddingJob(ctx context.Context, uploadID int) (id int, err error) {
	ctx, endObservation := s.operations.insertDependencyRepoAddingJob.With(ctx, &err, observation.Args{})
	defer func() {
		endObservation(1, observation.Args{LogFields: []log.Field{
			log.Int("id", id),
		}})
	}()
	id, _, err = basestore.ScanFirstInt(s.Store.Query(ctx, sqlf.Sprintf(insertDependencyRepoAddingJobQuery, uploadID)))
	return id, err
}

const insertDependencyRepoAddingJobQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/dependency_repo_add.go:InsertDependencyRepoInsertionJob
INSERT INTO codeintel_dependency_repo_adding_jobs (upload_id) VALUES (%s)
RETURNING id
`

func (s *Store) InsertCloneableDependencyRepos(ctx context.Context, dependencies []DependencyRepoInfo) (err error) {
	inserter := batch.NewInserterWithOnConflict(
		ctx,
		s.Handle().DB(),
		"codeintel_dependency_repos",
		[]string{"identifier", "version", "scheme"},
		"DO NOTHING")
	defer func() {
		if flushErr := inserter.Flush(ctx); flushErr != nil {
			err = errors.Wrap(flushErr, "batch.Inserter.Flush")
		}
	}()
	fmt.Printf("inserting %d elements\n", len(dependencies))
	for _, dep := range dependencies {
		if insertErr := inserter.Insert(ctx, dep.Identifier, dep.Version, dep.Scheme); insertErr != nil {
			return errors.Wrap(err, "batch.Inserter")
		}
		fmt.Printf("inserted %+v\n", dep)
	}
	return nil
}

type DependencyRepoInfo struct {
	Identifier, Version, Scheme string
}
