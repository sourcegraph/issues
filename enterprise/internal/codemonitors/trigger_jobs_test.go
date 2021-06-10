package codemonitors

import (
	"testing"

	"github.com/keegancsmith/sqlf"
)

const setToCompletedFmtStr = `
UPDATE cm_trigger_jobs
SET state = 'completed',
    started_at = %s,
    finished_at = %s
WHERE id = %s;
`

const getJobIDs = `
SELECT id
FROM cm_trigger_jobs;
`

func TestDeleteOldJobLogs(t *testing.T) {
	t.Parallel()

	retentionInDays := 7
	ctx, s := newTestStore(t)
	_, _, _, userCTX := newTestUser(ctx, t, s.Handle().DB())
	_, err := s.insertTestMonitor(userCTX, t)
	if err != nil {
		t.Fatal(err)
	}

	// Add 1 job and date it back to a long time ago.
	err = s.EnqueueTriggerQueries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	longTimeAgo := s.Now().AddDate(0, 0, -(retentionInDays + 1))
	err = s.Exec(ctx, sqlf.Sprintf(setToCompletedFmtStr, longTimeAgo, longTimeAgo, 1))
	if err != nil {
		t.Fatal(err)
	}

	// Add second job.
	err = s.EnqueueTriggerQueries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = s.Exec(ctx, sqlf.Sprintf(setToCompletedFmtStr, s.Now(), s.Now(), 2))
	if err != nil {
		t.Fatal(err)
	}

	err = s.DeleteOldJobLogs(ctx, retentionInDays)
	if err != nil {
		t.Fatal(err)
	}

	rows, err := s.Query(ctx, sqlf.Sprintf(getJobIDs))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var (
		rowCount int
		id       int
	)
	for rows.Next() {
		rowCount++
		if rowCount > 1 {
			t.Fatalf("got more than 1 row, expected exactly 1 row")
		}
		err = rows.Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
	}
	wantID := 2
	if id != wantID {
		t.Fatalf("got %d, expected %d", id, wantID)
	}
}
