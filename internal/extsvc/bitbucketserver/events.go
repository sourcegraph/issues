package bitbucketserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
)

const (
	eventTypeHeader = "X-Event-Key"
)

func WebhookEventType(r *http.Request) string {
	return r.Header.Get(eventTypeHeader)
}

func ParseWebhookEvent(eventType string, payload []byte) (e interface{}, err error) {
	switch eventType {
	case "ping":
		return PingEvent{}, nil
	case "repo:build_status":
		e = &BuildStatusEvent{}
		return e, json.Unmarshal(payload, e)
	case "pr:activity:status", "pr:activity:event", "pr:activity:rescope", "pr:activity:merge", "pr:activity:comment", "pr:activity:reviewers":
		e = &PullRequestActivityEvent{}
		return e, json.Unmarshal(payload, e)
	case "pr:participant:status":
		e = &PullRequestParticipantStatusEvent{}
		return e, json.Unmarshal(payload, e)
	default:
		return nil, errors.Errorf("unknown webhook event type: %q", eventType)
	}
}

type PingEvent struct{}

type PullRequestActivityEvent struct {
	Date        time.Time      `json:"date"`
	Actor       User           `json:"actor"`
	PullRequest PullRequest    `json:"pullRequest"`
	Action      ActivityAction `json:"action"`
	Activity    *Activity      `json:"activity"`
}

type PullRequestParticipantStatusEvent struct {
	*ParticipantStatusEvent
	PullRequest PullRequest `json:"pullRequest"`
}

type ParticipantStatusEvent struct {
	CreatedDate int            `json:"createdDate"`
	User        User           `json:"user"`
	Action      ActivityAction `json:"action"`
}

func (a *ParticipantStatusEvent) Key() string {
	return fmt.Sprintf("%s:%d:%d", a.Action, a.User.ID, a.CreatedDate)
}

type BuildStatusEvent struct {
	Commit       string        `json:"commit"`
	Status       BuildStatus   `json:"status"`
	PullRequests []PullRequest `json:"pullRequests"`
}
