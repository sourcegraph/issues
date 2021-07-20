package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/cockroachdb/errors"
	gh "github.com/google/go-github/v28/github"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

var dsn = flag.String("dsn", "", "Database connection string to use in integration tests")

func TestGithubWebhookDispatchSuccess(t *testing.T) {
	h := GitHubWebhook{}
	var called bool
	h.Register(func(ctx context.Context, svc *types.ExternalService, payload interface{}) error {
		called = true
		return nil
	}, "test-event-1")

	ctx := context.Background()
	if err := h.Dispatch(ctx, "test-event-1", nil, nil); err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
	if !called {
		t.Errorf("Expected called to be true, was false")
	}
}

func TestGithubWebhookDispatchNoHandler(t *testing.T) {
	h := GitHubWebhook{}
	ctx := context.Background()
	// no op
	if err := h.Dispatch(ctx, "test-event-1", nil, nil); err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
}

func TestGithubWebhookDispatchSuccessMultiple(t *testing.T) {
	var (
		h      = GitHubWebhook{}
		called = make(chan struct{}, 2)
	)
	h.Register(func(ctx context.Context, svc *types.ExternalService, payload interface{}) error {
		called <- struct{}{}
		return nil
	}, "test-event-1")
	h.Register(func(ctx context.Context, svc *types.ExternalService, payload interface{}) error {
		called <- struct{}{}
		return nil
	}, "test-event-1")

	ctx := context.Background()
	if err := h.Dispatch(ctx, "test-event-1", nil, nil); err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
	if len(called) != 2 {
		t.Errorf("Expected called to be 2, got %v", called)
	}
}

func TestGithubWebhookDispatchError(t *testing.T) {
	var (
		h      = GitHubWebhook{}
		called = make(chan struct{}, 2)
	)
	h.Register(func(ctx context.Context, svc *types.ExternalService, payload interface{}) error {
		called <- struct{}{}
		return errors.Errorf("oh no")
	}, "test-event-1")
	h.Register(func(ctx context.Context, svc *types.ExternalService, payload interface{}) error {
		called <- struct{}{}
		return nil
	}, "test-event-1")

	ctx := context.Background()
	if have, want := h.Dispatch(ctx, "test-event-1", nil, nil), "oh no"; errString(have) != want {
		t.Errorf("Expected %q, got %q", want, have)
	}
	if len(called) != 2 {
		t.Errorf("Expected called to be 2, got %v", called)
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestGithubWebhookExternalServices(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	db := dbtest.NewDB(t, *dsn)

	ctx := context.Background()

	secret := "secret"
	esStore := database.ExternalServices(db)
	extSvc := &types.ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "GitHub",
		Config: marshalJSON(t, &schema.GitHubConnection{
			Url:      "https://github.com",
			Token:    os.Getenv("GITHUB_TOKEN"),
			Repos:    []string{"sourcegraph/sourcegraph"},
			Webhooks: []*schema.GitHubWebhook{{Org: "sourcegraph", Secret: secret}},
		}),
	}

	err := esStore.Upsert(ctx, extSvc)
	if err != nil {
		t.Fatal(t)
	}

	hook := GitHubWebhook{
		ExternalServices: esStore,
	}

	var called bool
	hook.Register(func(ctx context.Context, extSvc *types.ExternalService, payload interface{}) error {
		evt, ok := payload.(*gh.PublicEvent)
		if !ok {
			t.Errorf("Expected *gh.PublicEvent event, got %T", payload)
		}
		if evt.GetRepo().GetFullName() != "sourcegraph/sourcegraph" {
			t.Errorf("Expected 'sourcegraph/sourcegraph', got %s", evt.GetRepo().GetFullName())
		}
		called = true
		return nil
	}, "public")

	urls := []string{
		// current webhook URLs, uses fast path for finding external service
		extsvc.WebhookURL(extsvc.TypeGitHub, extSvc.ID, "https://example.com/"),
		// old webhook URLs, finds external service by searching all configured external services
		"https://example.com/.api/github-webhook",
	}

	for _, u := range urls {
		called = false

		req, err := http.NewRequest("POST", u, bytes.NewReader(eventPayload))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Github-Event", "public")
		req.Header.Set("X-Hub-Signature", sign(t, eventPayload, []byte(secret)))

		rec := httptest.NewRecorder()
		hook.ServeHTTP(rec, req)
		resp := rec.Result()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Non 200 code: %v", resp.StatusCode)
		}

		if !called {
			t.Fatalf("Expected called to be true, got false (webhook handler was not called)")
		}
	}
}

func marshalJSON(t testing.TB, v interface{}) string {
	t.Helper()

	bs, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	return string(bs)
}

func sign(t *testing.T, message, secret []byte) string {
	t.Helper()

	mac := hmac.New(sha256.New, secret)

	_, err := mac.Write(message)
	if err != nil {
		t.Fatalf("writing hmac message failed: %s", err)
	}

	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

var eventPayload = []byte(`{
  "repository": {
    "id": 310572870,
    "node_id": "MDEwOlJlcG9zaXRvcnkzMTA1NzI4NzA=",
    "name": "sourcegraph",
    "full_name": "sourcegraph/sourcegraph",
    "private": false,
    "owner": {
      "login": "sourcegraph",
      "id": 74051180,
      "node_id": "MDEyOk9yZ2FuaXphdGlvbjc0MDUxMTgw",
      "type": "Organization",
      "site_admin": false
    },
    "html_url": "https://github.com/sourcegraph",
    "created_at": "2020-11-06T11:02:56Z",
    "updated_at": "2020-11-09T15:06:34Z",
    "pushed_at": "2020-11-06T11:02:58Z",
    "default_branch": "main"
  },
  "organization": {
    "login": "sourcegraph",
    "id": 74051180,
    "node_id": "MDEyOk9yZ2FuaXphdGlvbjc0MDUxMTgw",
    "description": null
  },
  "sender": {
    "login": "sourcegraph",
    "id": 5236823,
    "node_id": "MDQ6VXNlcjUyMzY4MjM=",
    "type": "User",
    "site_admin": false
  }
}`)
