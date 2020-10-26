package db

import (
	"context"
	"database/sql"
	"math"
	"time"

	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
)

// SurveyResponseListOptions specifies the options for listing survey responses.
type SurveyResponseListOptions struct {
	*LimitOffset
}

type surveyResponses struct{}

// Create creates a survey response.
func (s *surveyResponses) Create(ctx context.Context, userID *int32, email *string, score int, reason *string, better *string) (id int64, err error) {
	err = dbconn.Global.QueryRowContext(ctx,
		"INSERT INTO survey_responses(user_id, email, score, reason, better) VALUES($1, $2, $3, $4, $5) RETURNING id",
		userID, email, score, reason, better,
	).Scan(&id)
	return id, err
}

func (*surveyResponses) getBySQL(ctx context.Context, query string, args ...interface{}) ([]*types.SurveyResponse, error) {
	rows, err := dbconn.Global.QueryContext(ctx, "SELECT id, user_id, email, score, reason, better, created_at FROM survey_responses "+query, args...)
	if err != nil {
		return nil, err
	}
	responses := []*types.SurveyResponse{}
	defer rows.Close()
	for rows.Next() {
		r := types.SurveyResponse{}
		err := rows.Scan(&r.ID, &r.UserID, &r.Email, &r.Score, &r.Reason, &r.Better, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		responses = append(responses, &r)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return responses, nil
}

// GetAll gets all survey responses.
func (s *surveyResponses) GetAll(ctx context.Context) ([]*types.SurveyResponse, error) {
	return s.getBySQL(ctx, "ORDER BY created_at DESC")
}

// GetByUserID gets all survey responses by a given user.
func (s *surveyResponses) GetByUserID(ctx context.Context, userID int32) ([]*types.SurveyResponse, error) {
	return s.getBySQL(ctx, "WHERE user_id=$1 ORDER BY created_at DESC", userID)
}

// Count returns the count of all survey responses.
func (s *surveyResponses) Count(ctx context.Context) (int, error) {
	q := sqlf.Sprintf("SELECT COUNT(*) FROM survey_responses")

	var count int
	err := dbconn.Global.QueryRowContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...).Scan(&count)
	return count, err
}

// Last30DaysAverageScore returns the average score for all surveys submitted in the last 30 days.
func (s *surveyResponses) Last30DaysAverageScore(ctx context.Context) (float64, error) {
	q := sqlf.Sprintf("SELECT AVG(score) FROM survey_responses WHERE created_at>%s", thirtyDaysAgo())

	var avg sql.NullFloat64
	err := dbconn.Global.QueryRowContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...).Scan(&avg)
	return avg.Float64, err
}

// Last30DaysNPS returns the net promoter score for all surveys submitted in the last 30 days.
// This is calculated as 100*((% of responses that are >= 9) - (% of responses that are <= 6))
func (s *surveyResponses) Last30DaysNetPromoterScore(ctx context.Context) (int, error) {
	since := thirtyDaysAgo()
	promotersQ := sqlf.Sprintf("SELECT COUNT(*) FROM survey_responses WHERE created_at>%s AND score>8", since)
	detractorsQ := sqlf.Sprintf("SELECT COUNT(*) FROM survey_responses WHERE created_at>%s AND score<7", since)

	count, err := s.Last30DaysCount(ctx)
	// If no survey responses have been recorded, return 0.
	if err != nil || count == 0 {
		return 0, err
	}

	var promoters, detractors int
	err = dbconn.Global.QueryRowContext(ctx, promotersQ.Query(sqlf.PostgresBindVar), promotersQ.Args()...).Scan(&promoters)
	if err != nil {
		return 0, err
	}
	err = dbconn.Global.QueryRowContext(ctx, detractorsQ.Query(sqlf.PostgresBindVar), detractorsQ.Args()...).Scan(&detractors)
	promoterPercent := math.Round(float64(promoters) / float64(count) * 100.0)
	detractorPercent := math.Round(float64(detractors) / float64(count) * 100.0)

	return int(promoterPercent - detractorPercent), err
}

// Last30Count returns the count of surveys submitted in the last 30 days.
func (s *surveyResponses) Last30DaysCount(ctx context.Context) (int, error) {
	q := sqlf.Sprintf("SELECT COUNT(*) FROM survey_responses WHERE created_at>%s", thirtyDaysAgo())

	var count int
	err := dbconn.Global.QueryRowContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...).Scan(&count)
	return count, err
}

func thirtyDaysAgo() string {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -30).Format("2006-01-02 15:04:05 UTC")
}
