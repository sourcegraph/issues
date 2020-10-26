package campaigns

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab/webhooks"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/schema"
)

func testGitLabWebhook(db *sql.DB, userID int32) func(*testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		t.Run("ServeHTTP", func(t *testing.T) {
			t.Run("missing external service", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)

				u := extsvc.WebhookURL(extsvc.TypeGitLab, 12345, "https://example.com/")
				req, err := http.NewRequest("POST", u, nil)
				if err != nil {
					t.Fatal(err)
				}

				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				if have, want := rec.Result().StatusCode, http.StatusUnauthorized; have != want {
					t.Errorf("unexpected status code: have %d; want %d", have, want)
				}
			})

			t.Run("invalid external service", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)

				u := strings.ReplaceAll(extsvc.WebhookURL(extsvc.TypeGitLab, 12345, "https://example.com/"), "12345", "foo")
				req, err := http.NewRequest("POST", u, nil)
				if err != nil {
					t.Fatal(err)
				}

				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				resp := rec.Result()
				if have, want := resp.StatusCode, http.StatusInternalServerError; have != want {
					t.Errorf("unexpected status code: have %d; want %d", have, want)
				}
				assertBodyIncludes(t, resp.Body, "getting external service")
			})

			t.Run("malformed external service", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)
				es := createGitLabExternalService(t, ctx, rstore)

				es.Config = "invalid JSON"
				if err := rstore.UpsertExternalServices(ctx, es); err != nil {
					t.Fatal(err)
				}

				u := extsvc.WebhookURL(extsvc.TypeGitLab, es.ID, "https://example.com/")
				req, err := http.NewRequest("POST", u, nil)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Add(webhooks.TokenHeaderName, "not a valid secret")

				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				resp := rec.Result()
				if have, want := resp.StatusCode, http.StatusInternalServerError; have != want {
					t.Errorf("unexpected status code: have %d; want %d", have, want)
				}
				assertBodyIncludes(t, resp.Body, "validating the shared secret")
			})

			t.Run("missing secret", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)
				es := createGitLabExternalService(t, ctx, rstore)

				u := extsvc.WebhookURL(extsvc.TypeGitLab, es.ID, "https://example.com/")
				req, err := http.NewRequest("POST", u, nil)
				if err != nil {
					t.Fatal(err)
				}

				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				resp := rec.Result()
				if have, want := resp.StatusCode, http.StatusUnauthorized; have != want {
					t.Errorf("unexpected status code: have %d; want %d", have, want)
				}
				assertBodyIncludes(t, resp.Body, "shared secret is incorrect")
			})

			t.Run("incorrect secret", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)
				es := createGitLabExternalService(t, ctx, rstore)

				u := extsvc.WebhookURL(extsvc.TypeGitLab, es.ID, "https://example.com/")
				req, err := http.NewRequest("POST", u, nil)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Add(webhooks.TokenHeaderName, "not a valid secret")

				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				resp := rec.Result()
				if have, want := resp.StatusCode, http.StatusUnauthorized; have != want {
					t.Errorf("unexpected status code: have %d; want %d", have, want)
				}
				assertBodyIncludes(t, resp.Body, "shared secret is incorrect")
			})

			t.Run("missing body", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)
				es := createGitLabExternalService(t, ctx, rstore)

				u := extsvc.WebhookURL(extsvc.TypeGitLab, es.ID, "https://example.com/")
				req, err := http.NewRequest("POST", u, nil)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Add(webhooks.TokenHeaderName, "secret")

				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				resp := rec.Result()
				if have, want := resp.StatusCode, http.StatusBadRequest; have != want {
					t.Errorf("unexpected status code: have %d; want %d", have, want)
				}
				assertBodyIncludes(t, resp.Body, "missing request body")
			})

			t.Run("unreadable body", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)
				es := createGitLabExternalService(t, ctx, rstore)

				u := extsvc.WebhookURL(extsvc.TypeGitLab, es.ID, "https://example.com/")
				req, err := http.NewRequest("POST", u, nil)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Add(webhooks.TokenHeaderName, "secret")
				req.Body = &brokenReader{errors.New("foo")}

				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				resp := rec.Result()
				if have, want := resp.StatusCode, http.StatusInternalServerError; have != want {
					t.Errorf("unexpected status code: have %d; want %d", have, want)
				}
				assertBodyIncludes(t, resp.Body, "reading payload")
			})

			t.Run("malformed body", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)
				es := createGitLabExternalService(t, ctx, rstore)

				u := extsvc.WebhookURL(extsvc.TypeGitLab, es.ID, "https://example.com/")
				req, err := http.NewRequest("POST", u, bytes.NewBufferString("invalid JSON"))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Add(webhooks.TokenHeaderName, "secret")

				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				resp := rec.Result()
				if have, want := resp.StatusCode, http.StatusInternalServerError; have != want {
					t.Errorf("unexpected status code: have %d; want %d", have, want)
				}
				assertBodyIncludes(t, resp.Body, "unmarshalling payload")
			})

			t.Run("invalid body", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)
				es := createGitLabExternalService(t, ctx, rstore)

				u := extsvc.WebhookURL(extsvc.TypeGitLab, es.ID, "https://example.com/")
				body := marshalJSON(t, &webhooks.EventCommon{
					ObjectKind: "unknown",
				})
				req, err := http.NewRequest("POST", u, bytes.NewBufferString(body))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Add(webhooks.TokenHeaderName, "secret")

				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				resp := rec.Result()
				if have, want := resp.StatusCode, http.StatusNoContent; have != want {
					t.Errorf("unexpected status code: have %d; want %d", have, want)
				}
			})

			t.Run("error from handleEvent", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)
				es := createGitLabExternalService(t, ctx, rstore)
				repo := createGitLabRepo(t, ctx, rstore, es)
				changeset := createGitLabChangeset(t, ctx, store, repo)
				body := createMergeRequestPayload(t, repo, changeset, "close")

				// Remove the URL from the GitLab configuration.
				cfg, err := es.Configuration()
				if err != nil {
					t.Fatal(err)
				}
				conn := cfg.(*schema.GitLabConnection)
				conn.Url = ""
				es.Config = marshalJSON(t, conn)
				if err := rstore.UpsertExternalServices(ctx, es); err != nil {
					t.Fatal(err)
				}

				u := extsvc.WebhookURL(extsvc.TypeGitLab, es.ID, "https://example.com/")
				req, err := http.NewRequest("POST", u, bytes.NewBufferString(body))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Add(webhooks.TokenHeaderName, "secret")

				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				resp := rec.Result()
				if have, want := resp.StatusCode, http.StatusInternalServerError; have != want {
					t.Errorf("unexpected status code: have %d; want %d", have, want)
				}
				assertBodyIncludes(t, resp.Body, "could not determine service id")
			})

			// The valid tests below are pretty "happy path": specific unit
			// tests for the utility methods on GitLabWebhook are below. We're
			// mostly just testing the routing here, since these are ServeHTTP
			// tests; however, these also act as integration tests. (Which,
			// considering they're ultimately invoked from TestIntegration,
			// seems fair.)

			t.Run("valid merge request approval events", func(t *testing.T) {
				for _, action := range []string{"approved", "unapproved"} {
					t.Run(action, func(t *testing.T) {
						store, rstore, clock := gitLabTestSetup(t, db)
						h := NewGitLabWebhook(store, rstore, clock.now)
						es := createGitLabExternalService(t, ctx, rstore)
						repo := createGitLabRepo(t, ctx, rstore, es)
						changeset := createGitLabChangeset(t, ctx, store, repo)
						body := createMergeRequestPayload(t, repo, changeset, "approved")

						u := extsvc.WebhookURL(extsvc.TypeGitLab, es.ID, "https://example.com/")
						req, err := http.NewRequest("POST", u, bytes.NewBufferString(body))
						if err != nil {
							t.Fatal(err)
						}
						req.Header.Add(webhooks.TokenHeaderName, "secret")

						changesetEnqueued := false
						repoupdater.MockEnqueueChangesetSync = func(ctx context.Context, ids []int64) error {
							changesetEnqueued = true
							if diff := cmp.Diff(ids, []int64{changeset.ID}); diff != "" {
								t.Errorf("unexpected changeset ID: %s", diff)
							}
							return nil
						}
						defer func() { repoupdater.MockEnqueueChangesetSync = nil }()

						rec := httptest.NewRecorder()
						h.ServeHTTP(rec, req)

						resp := rec.Result()
						if have, want := resp.StatusCode, http.StatusNoContent; have != want {
							t.Errorf("unexpected status code: have %d; want %d", have, want)
						}
						if !changesetEnqueued {
							t.Error("changeset was not enqueued")
						}
					})
				}
			})

			t.Run("valid merge request state change events", func(t *testing.T) {
				for action, want := range map[string]campaigns.ChangesetEventKind{
					"close":  campaigns.ChangesetEventKindGitLabClosed,
					"merge":  campaigns.ChangesetEventKindGitLabMerged,
					"reopen": campaigns.ChangesetEventKindGitLabReopened,
				} {
					t.Run(action, func(t *testing.T) {
						store, rstore, clock := gitLabTestSetup(t, db)
						h := NewGitLabWebhook(store, rstore, clock.now)
						es := createGitLabExternalService(t, ctx, rstore)
						repo := createGitLabRepo(t, ctx, rstore, es)
						changeset := createGitLabChangeset(t, ctx, store, repo)
						body := createMergeRequestPayload(t, repo, changeset, action)

						u := extsvc.WebhookURL(extsvc.TypeGitLab, es.ID, "https://example.com/")
						req, err := http.NewRequest("POST", u, bytes.NewBufferString(body))
						if err != nil {
							t.Fatal(err)
						}
						req.Header.Add(webhooks.TokenHeaderName, "secret")

						rec := httptest.NewRecorder()
						h.ServeHTTP(rec, req)

						resp := rec.Result()
						if have, want := resp.StatusCode, http.StatusNoContent; have != want {
							t.Errorf("unexpected status code: have %d; want %d", have, want)
						}

						// Verify that the changeset event was upserted.
						assertChangesetEventForChangeset(t, ctx, store, changeset, want)
					})
				}
			})

			t.Run("valid pipeline events", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)
				es := createGitLabExternalService(t, ctx, rstore)
				repo := createGitLabRepo(t, ctx, rstore, es)
				changeset := createGitLabChangeset(t, ctx, store, repo)
				body := createPipelinePayload(t, repo, changeset, gitlab.Pipeline{
					ID:     123,
					Status: gitlab.PipelineStatusSuccess,
				})

				u := extsvc.WebhookURL(extsvc.TypeGitLab, es.ID, "https://example.com/")
				req, err := http.NewRequest("POST", u, bytes.NewBufferString(body))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Add(webhooks.TokenHeaderName, "secret")

				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				resp := rec.Result()
				if have, want := resp.StatusCode, http.StatusNoContent; have != want {
					t.Errorf("unexpected status code: have %d; want %d", have, want)
				}

				assertChangesetEventForChangeset(t, ctx, store, changeset, campaigns.ChangesetEventKindGitLabPipeline)
			})
		})

		t.Run("getExternalServiceFromRawID", func(t *testing.T) {
			// Since these tests don't write to the database, we can just share
			// the same database setup.
			store, rstore, clock := gitLabTestSetup(t, db)
			h := NewGitLabWebhook(store, rstore, clock.now)

			// Set up two GitLab external services.
			a := createGitLabExternalService(t, ctx, rstore)
			b := createGitLabExternalService(t, ctx, rstore)

			// Set up a GitHub external service.
			github := createGitLabExternalService(t, ctx, rstore)
			github.Kind = extsvc.KindGitHub
			if err := rstore.UpsertExternalServices(ctx, github); err != nil {
				t.Fatal(err)
			}

			t.Run("invalid ID", func(t *testing.T) {
				for _, id := range []string{"", "foo"} {
					t.Run(id, func(t *testing.T) {
						es, err := h.getExternalServiceFromRawID(ctx, "foo")
						if es != nil {
							t.Errorf("unexpected non-nil external service: %+v", es)
						}
						if err == nil {
							t.Error("unexpected nil error")
						}
					})
				}
			})

			t.Run("missing ID", func(t *testing.T) {
				for name, id := range map[string]string{
					"not found":  "12345",
					"wrong kind": strconv.FormatInt(github.ID, 10),
				} {
					t.Run(name, func(t *testing.T) {
						es, err := h.getExternalServiceFromRawID(ctx, id)
						if es != nil {
							t.Errorf("unexpected non-nil external service: %+v", es)
						}
						if want := errExternalServiceNotFound; err != want {
							t.Errorf("unexpected error: have %+v; want %+v", err, want)
						}
					})
				}
			})

			t.Run("valid ID", func(t *testing.T) {
				for id, want := range map[int64]*repos.ExternalService{
					a.ID: a,
					b.ID: b,
				} {
					sid := strconv.FormatInt(id, 10)
					t.Run(sid, func(t *testing.T) {
						have, err := h.getExternalServiceFromRawID(ctx, sid)
						if err != nil {
							t.Errorf("unexpected non-nil error: %+v", err)
						}
						if diff := cmp.Diff(have, want); diff != "" {
							t.Errorf("unexpected external service: %s", diff)
						}
					})
				}
			})
		})

		t.Run("broken repo store", func(t *testing.T) {
			// This test is separate from the other unit tests for this
			// function above because it needs to set up a bad database
			// connection on the repo store.
			store, _, clock := gitLabTestSetup(t, db)
			rstore := repos.NewDBStore(&brokenDB{errors.New("foo")}, sql.TxOptions{})
			h := NewGitLabWebhook(store, rstore, clock.now)

			_, err := h.getExternalServiceFromRawID(ctx, "12345")
			if err == nil {
				t.Error("unexpected nil error")
			}
		})

		t.Run("broken campaign store", func(t *testing.T) {
			// We can induce an error with a broken database connection.
			store, rstore, clock := gitLabTestSetup(t, db)
			h := NewGitLabWebhook(store, rstore, clock.now)
			h.Store = NewStoreWithClock(&brokenDB{errors.New("foo")}, clock.now)

			es, err := h.getExternalServiceFromRawID(ctx, "12345")
			if es != nil {
				t.Errorf("unexpected non-nil external service: %+v", es)
			}
			if err == nil {
				t.Error("unexpected nil error")
			}
		})

		t.Run("handleEvent", func(t *testing.T) {
			// There aren't a lot of these tests, as most of the viable error
			// paths are covered by the ServeHTTP tests above, but these fill
			// in the gaps as best we can.

			t.Run("unknown event type", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)
				es := createGitLabExternalService(t, ctx, rstore)

				err := h.handleEvent(ctx, es, nil)
				if err != nil {
					t.Errorf("unexpected non-nil error: %+v", err)
				}
			})

			t.Run("error from enqueueChangesetSyncFromEvent", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)
				es := createGitLabExternalService(t, ctx, rstore)

				// We can induce an error with an incomplete merge request
				// event that's missing a project.
				event := &webhooks.MergeRequestApprovedEvent{
					MergeRequestEventCommon: webhooks.MergeRequestEventCommon{
						MergeRequest: &gitlab.MergeRequest{IID: 42},
					},
				}

				err := h.handleEvent(ctx, es, event)
				if err == nil {
					t.Error("unexpected nil error")
				} else if want := http.StatusInternalServerError; err.code != want {
					t.Errorf("unexpected status code: have %d; want %d", err.code, want)
				}
			})

			t.Run("error from handleMergeRequestStateEvent", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)
				es := createGitLabExternalService(t, ctx, rstore)

				event := &webhooks.MergeRequestCloseEvent{
					MergeRequestEventCommon: webhooks.MergeRequestEventCommon{
						MergeRequest: &gitlab.MergeRequest{IID: 42},
					},
				}

				// We can induce an error with a broken database connection.
				h.Store = NewStoreWithClock(&brokenDB{errors.New("foo")}, clock.now)

				err := h.handleEvent(ctx, es, event)
				if err == nil {
					t.Error("unexpected nil error")
				} else if want := http.StatusInternalServerError; err.code != want {
					t.Errorf("unexpected status code: have %d; want %d", err.code, want)
				}
			})

			t.Run("error from handlePipelineEvent", func(t *testing.T) {
				store, rstore, clock := gitLabTestSetup(t, db)
				h := NewGitLabWebhook(store, rstore, clock.now)
				es := createGitLabExternalService(t, ctx, rstore)

				event := &webhooks.PipelineEvent{
					MergeRequest: &gitlab.MergeRequest{IID: 42},
				}

				// We can induce an error with a broken database connection.
				h.Store = NewStoreWithClock(&brokenDB{errors.New("foo")}, clock.now)

				err := h.handleEvent(ctx, es, event)
				if err == nil {
					t.Error("unexpected nil error")
				} else if want := http.StatusInternalServerError; err.code != want {
					t.Errorf("unexpected status code: have %d; want %d", err.code, want)
				}
			})
		})

		t.Run("enqueueChangesetSyncFromEvent", func(t *testing.T) {
			// Since these tests don't write to the database, we can just share
			// the same database setup.
			store, rstore, clock := gitLabTestSetup(t, db)
			h := NewGitLabWebhook(store, rstore, clock.now)
			es := createGitLabExternalService(t, ctx, rstore)
			repo := createGitLabRepo(t, ctx, rstore, es)
			changeset := createGitLabChangeset(t, ctx, store, repo)

			// Extract IDs we'll need to build events.
			cid, err := strconv.Atoi(changeset.ExternalID)
			if err != nil {
				t.Fatal(err)
			}

			pid, err := strconv.Atoi(repo.ExternalRepo.ID)
			if err != nil {
				t.Fatal(err)
			}

			esid, err := extractExternalServiceID(es)
			if err != nil {
				t.Fatal(err)
			}

			t.Run("missing repo", func(t *testing.T) {
				event := &webhooks.MergeRequestEventCommon{
					EventCommon: webhooks.EventCommon{
						Project: gitlab.ProjectCommon{ID: 12345},
					},
					MergeRequest: &gitlab.MergeRequest{IID: gitlab.ID(cid)},
				}

				if err := h.enqueueChangesetSyncFromEvent(ctx, esid, event); err == nil {
					t.Error("unexpected nil error")
				}
			})

			t.Run("missing changeset", func(t *testing.T) {
				event := &webhooks.MergeRequestEventCommon{
					EventCommon: webhooks.EventCommon{
						Project: gitlab.ProjectCommon{ID: pid},
					},
					MergeRequest: &gitlab.MergeRequest{IID: 12345},
				}

				if err := h.enqueueChangesetSyncFromEvent(ctx, esid, event); err == nil {
					t.Error("unexpected nil error")
				}
			})

			t.Run("repo updater error", func(t *testing.T) {
				event := &webhooks.MergeRequestEventCommon{
					EventCommon: webhooks.EventCommon{
						Project: gitlab.ProjectCommon{ID: pid},
					},
					MergeRequest: &gitlab.MergeRequest{IID: gitlab.ID(cid)},
				}

				want := errors.New("foo")
				repoupdater.MockEnqueueChangesetSync = func(ctx context.Context, ids []int64) error {
					return want
				}
				defer func() { repoupdater.MockEnqueueChangesetSync = nil }()

				if have := h.enqueueChangesetSyncFromEvent(ctx, esid, event); !errors.Is(have, want) {
					t.Errorf("unexpected error: have %+v; want %+v", have, want)
				}
			})

			t.Run("success", func(t *testing.T) {
				event := &webhooks.MergeRequestEventCommon{
					EventCommon: webhooks.EventCommon{
						Project: gitlab.ProjectCommon{ID: pid},
					},
					MergeRequest: &gitlab.MergeRequest{IID: gitlab.ID(cid)},
				}

				repoupdater.MockEnqueueChangesetSync = func(ctx context.Context, ids []int64) error {
					return nil
				}
				defer func() { repoupdater.MockEnqueueChangesetSync = nil }()

				if err := h.enqueueChangesetSyncFromEvent(ctx, esid, event); err != nil {
					t.Errorf("unexpected non-nil error: %+v", err)
				}
			})
		})

		t.Run("handleMergeRequestStateEvent changeset upsert error", func(t *testing.T) {
			// The success path is well tested in the ServeHTTP tests above, so
			// here we're just going to exercise the upsert error path.
			//
			// It's actually fairly difficult to induce
			// Webhook.upsertChangesetEvent to return an error: if it can't
			// find the repo or changeset, it returns nil and swallows the
			// error so that the code host doesn't see an error. However, we
			// can return an error when it attempts to begin a transaction and
			// that will generate a real error that we can use to exercise the
			// error path.
			store, rstore, clock := gitLabTestSetup(t, db)
			store = NewStoreWithClock(&noNestingTx{store.DB()}, clock.now)
			h := NewGitLabWebhook(store, rstore, clock.now)

			event := &webhooks.MergeRequestCloseEvent{
				MergeRequestEventCommon: webhooks.MergeRequestEventCommon{
					EventCommon: webhooks.EventCommon{
						Project: gitlab.ProjectCommon{ID: 12345},
					},
					MergeRequest: &gitlab.MergeRequest{IID: gitlab.ID(12345)},
				},
			}

			if err := h.handleMergeRequestStateEvent(ctx, "ignored", event); err == nil {
				t.Error("unexpected nil error")
			}
		})

		t.Run("handlePipelineEvent", func(t *testing.T) {
			// As with the handleMergeRequestStateEvent test above, we don't
			// really need to test the success path here. However, there's one
			// extra error path, so we'll use two sub-tests to ensure we hit
			// them both.
			//
			// Again, we're going to set up a poisoned store database that will
			// error if a transaction is started.
			store, rstore, clock := gitLabTestSetup(t, db)
			store = NewStoreWithClock(&noNestingTx{store.DB()}, clock.now)
			h := NewGitLabWebhook(store, rstore, clock.now)

			t.Run("missing merge request", func(t *testing.T) {
				event := &webhooks.PipelineEvent{}

				if have := h.handlePipelineEvent(ctx, "ignored", event); have != errPipelineMissingMergeRequest {
					t.Errorf("unexpected error: have %+v; want %+v", have, errPipelineMissingMergeRequest)
				}
			})

			t.Run("changeset upsert error", func(t *testing.T) {
				event := &webhooks.PipelineEvent{
					MergeRequest: &gitlab.MergeRequest{},
				}

				if err := h.handlePipelineEvent(ctx, "ignored", event); err == nil || err == errPipelineMissingMergeRequest {
					t.Errorf("unexpected error: %+v", err)
				}
			})
		})
	}
}

func TestValidateGitLabSecret(t *testing.T) {
	t.Run("empty secret", func(t *testing.T) {
		ok, err := validateGitLabSecret(nil, "")
		if ok {
			t.Errorf("unexpected ok: %v", ok)
		}
		if err != nil {
			t.Errorf("unexpected non-nil error: %+v", err)
		}
	})

	t.Run("invalid configuration", func(t *testing.T) {
		es := &repos.ExternalService{}
		ok, err := validateGitLabSecret(es, "secret")
		if ok {
			t.Errorf("unexpected ok: %v", ok)
		}
		if err == nil {
			t.Error("unexpected nil error")
		}
	})

	t.Run("not a GitLab connection", func(t *testing.T) {
		es := &repos.ExternalService{Kind: extsvc.KindGitHub}
		ok, err := validateGitLabSecret(es, "secret")
		if ok {
			t.Errorf("unexpected ok: %v", ok)
		}
		if err != errExternalServiceWrongKind {
			t.Errorf("unexpected error: have %+v; want %+v", err, errExternalServiceWrongKind)
		}
	})

	t.Run("no webhooks", func(t *testing.T) {
		es := &repos.ExternalService{
			Kind: extsvc.KindGitLab,
			Config: marshalJSON(t, &schema.GitLabConnection{
				Webhooks: []*schema.GitLabWebhook{},
			}),
		}

		ok, err := validateGitLabSecret(es, "secret")
		if ok {
			t.Errorf("unexpected ok: %v", ok)
		}
		if err != nil {
			t.Errorf("unexpected non-nil error: %+v", err)
		}
	})

	t.Run("valid webhooks", func(t *testing.T) {
		for secret, want := range map[string]bool{
			"not secret": false,
			"secret":     true,
			"super":      true,
		} {
			t.Run(secret, func(t *testing.T) {
				es := &repos.ExternalService{
					Kind: extsvc.KindGitLab,
					Config: marshalJSON(t, &schema.GitLabConnection{
						Webhooks: []*schema.GitLabWebhook{
							{Secret: "super"},
							{Secret: "secret"},
						},
					}),
				}

				ok, err := validateGitLabSecret(es, secret)
				if ok != want {
					t.Errorf("unexpected ok: have %v; want %v", ok, want)
				}
				if err != nil {
					t.Errorf("unexpected non-nil error: %+v", err)
				}
			})
		}
	})
}

// brokenDB provides a dbutil.DB that always fails: for methods that return an
// error, the err field will be returned; otherwise nil will be returned.
type brokenDB struct{ err error }

func (db *brokenDB) QueryContext(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
	return nil, db.err
}

func (db *brokenDB) ExecContext(ctx context.Context, q string, args ...interface{}) (sql.Result, error) {
	return nil, db.err
}

func (db *brokenDB) QueryRowContext(ctx context.Context, q string, args ...interface{}) *sql.Row {
	return nil
}

// brokenReader implements an io.ReadCloser that always returns an error when
// read.
type brokenReader struct{ err error }

func (br *brokenReader) Close() error { return nil }

func (br *brokenReader) Read(p []byte) (int, error) {
	return 0, br.err
}

// nestedTx wraps an existing transaction and overrides its transaction methods
// to be no-ops. This allows us to have a master transaction used in tests that
// test functions that attempt to create and commit transactions: since
// PostgreSQL doesn't support nested transactions, we can still use the master
// transaction to manage the test database state without rollback/commit
// already performed errors.
//
// It would be theoretically possible to use savepoints to implement something
// resembling the semantics of a true nested transaction, but that's
// unnecessary for these tests.
type nestedTx struct{ *sql.Tx }

func (ntx *nestedTx) Rollback() error                                        { return nil }
func (ntx *nestedTx) Commit() error                                          { return nil }
func (ntx *nestedTx) BeginTx(ctx context.Context, opts *sql.TxOptions) error { return nil }

// noNestingTx is another transaction wrapper that always returns an error when
// a transaction is attempted.
type noNestingTx struct{ dbutil.DB }

func (nntx *noNestingTx) BeginTx(ctx context.Context, opts *sql.TxOptions) error {
	return errors.New("foo")
}

// gitLabTestSetup instantiates the stores and a clock for use within tests.
// Any changes made to the stores will be rolled back after the test is
// complete.
func gitLabTestSetup(t *testing.T, db *sql.DB) (*Store, repos.Store, clock) {
	c := &testClock{t: time.Now().UTC().Truncate(time.Microsecond)}
	tx := dbtest.NewTx(t, db)

	// Note that tx is wrapped in nestedTx to effectively neuter further use of
	// transactions within the test.
	return NewStoreWithClock(&nestedTx{tx}, c.now), repos.NewDBStore(tx, sql.TxOptions{}), c
}

// assertBodyIncludes checks for a specific substring within the given response
// body, and generates a test error if the substring is not found. This is
// mostly useful to look for wrapped errors in the output.
func assertBodyIncludes(t *testing.T, r io.Reader, want string) {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte(want)) {
		t.Errorf("cannot find expected string in output: want: %s; have:\n%s", want, string(body))
	}
}

// assertChangesetEventForChangeset checks that one (and only one) changeset
// event has been created on the given changeset, and that it is of the given
// kind.
func assertChangesetEventForChangeset(t *testing.T, ctx context.Context, store *Store, changeset *campaigns.Changeset, want campaigns.ChangesetEventKind) {
	ces, _, err := store.ListChangesetEvents(ctx, ListChangesetEventsOpts{
		ChangesetIDs: []int64{changeset.ID},
		LimitOpts:    LimitOpts{Limit: 100},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ces) == 1 {
		ce := ces[0]

		if ce.ChangesetID != changeset.ID {
			t.Errorf("unexpected changeset ID: have %d; want %d", ce.ChangesetID, changeset.ID)
		}
		if ce.Kind != want {
			t.Errorf(
				"unexpected changeset event kind: have %v; want %v", ce.Kind, want)
		}
	} else {
		t.Errorf("unexpected number of changeset events; got %+v", ces)
	}
}

// createGitLabExternalService creates a mock GitLab service with a valid
// configuration, including the secrets "super" and "secret".
func createGitLabExternalService(t *testing.T, ctx context.Context, rstore repos.Store) *repos.ExternalService {
	es := &repos.ExternalService{
		Kind:        extsvc.KindGitLab,
		DisplayName: "gitlab",
		Config: marshalJSON(t, &schema.GitLabConnection{
			Url: "https://gitlab.com/",
			Webhooks: []*schema.GitLabWebhook{
				{Secret: "super"},
				{Secret: "secret"},
			},
		}),
	}
	if err := rstore.UpsertExternalServices(ctx, es); err != nil {
		t.Fatal(err)
	}

	return es
}

// createGitLabRepo creates a mock GitLab repo attached to the given external
// service.
func createGitLabRepo(t *testing.T, ctx context.Context, rstore repos.Store, es *repos.ExternalService) *repos.Repo {
	repo := (&repos.Repo{
		Name: "gitlab.com/sourcegraph/test",
		URI:  "gitlab.com/sourcegraph/test",
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "123",
			ServiceType: extsvc.TypeGitLab,
			ServiceID:   "https://gitlab.com/",
		},
	}).With(repos.Opt.RepoSources(es.URN()))
	if err := rstore.InsertRepos(ctx, repo); err != nil {
		t.Fatal(err)
	}

	return repo
}

// createGitLabChangeset creates a mock GitLab changeset.
func createGitLabChangeset(t *testing.T, ctx context.Context, store *Store, repo *repos.Repo) *campaigns.Changeset {
	c := &campaigns.Changeset{
		RepoID:              repo.ID,
		ExternalID:          "1",
		ExternalServiceType: extsvc.TypeGitLab,
	}
	if err := store.CreateChangeset(ctx, c); err != nil {
		t.Fatal(err)
	}

	return c
}

// createMergeRequestPayload creates a mock GitLab webhook payload of the merge
// request object kind.
func createMergeRequestPayload(t *testing.T, repo *repos.Repo, changeset *campaigns.Changeset, action string) string {
	cid, err := strconv.Atoi(changeset.ExternalID)
	if err != nil {
		t.Fatal(err)
	}

	pid, err := strconv.Atoi(repo.ExternalRepo.ID)
	if err != nil {
		t.Fatal(err)
	}

	// We use an untyped set of maps here because the webhooks package doesn't
	// export its internal mergeRequestEvent type that is used for
	// unmarshalling. (Which is fine; it's an implementation detail.)
	return marshalJSON(t, map[string]interface{}{
		"object_kind": "merge_request",
		"project": map[string]interface{}{
			"id": pid,
		},
		"object_attributes": map[string]interface{}{
			"iid":    cid,
			"action": action,
		},
	})
}

// createPipelinePayload creates a mock GitLab webhook payload of the pipeline
// object kind.
func createPipelinePayload(t *testing.T, repo *repos.Repo, changeset *campaigns.Changeset, pipeline gitlab.Pipeline) string {
	pid, err := strconv.Atoi(repo.ExternalRepo.ID)
	if err != nil {
		t.Fatal(err)
	}

	payload := &webhooks.PipelineEvent{
		EventCommon: webhooks.EventCommon{
			ObjectKind: "pipeline",
			Project: gitlab.ProjectCommon{
				ID: pid,
			},
		},
		Pipeline: pipeline,
	}

	if changeset != nil {
		cid, err := strconv.Atoi(changeset.ExternalID)
		if err != nil {
			t.Fatal(err)
		}

		payload.MergeRequest = &gitlab.MergeRequest{
			IID: gitlab.ID(cid),
		}
	}

	return marshalJSON(t, payload)
}
