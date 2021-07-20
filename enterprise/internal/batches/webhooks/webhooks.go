package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/state"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

type Webhook struct {
	Store *store.Store

	// ServiceType corresponds to api.ExternalRepoSpec.ServiceType
	// Example values: extsvc.TypeBitbucketServer, extsvc.TypeGitHub
	ServiceType string
}

type PR struct {
	ID             int64
	RepoExternalID string
}

func (h Webhook) getRepoForPR(
	ctx context.Context,
	tx *store.Store,
	pr PR,
	externalServiceID string,
) (*types.Repo, error) {
	rs, err := tx.Repos().List(ctx, database.ReposListOptions{
		ExternalRepos: []api.ExternalRepoSpec{
			{
				ID:          pr.RepoExternalID,
				ServiceType: h.ServiceType,
				ServiceID:   externalServiceID,
			},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to load repository")
	}

	if len(rs) != 1 {
		return nil, errors.Errorf("fetched repositories have wrong length: %d", len(rs))
	}

	return rs[0], nil
}

func extractExternalServiceID(extSvc *types.ExternalService) (string, error) {
	c, err := extSvc.Configuration()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get external service config")
	}

	var serviceID string
	switch c := c.(type) {
	case *schema.GitHubConnection:
		serviceID = c.Url
	case *schema.BitbucketServerConnection:
		serviceID = c.Url
	case *schema.GitLabConnection:
		serviceID = c.Url
	}
	if serviceID == "" {
		return "", errors.New("could not determine service id")
	}

	u, err := url.Parse(serviceID)
	if err != nil {
		return "", errors.Wrap(err, "Failed to parse service ID")
	}

	return extsvc.NormalizeBaseURL(u).String(), nil
}

type keyer interface {
	Key() string
}

func (h Webhook) upsertChangesetEvent(
	ctx context.Context,
	externalServiceID string,
	pr PR,
	ev keyer,
) (err error) {
	var tx *store.Store
	if tx, err = h.Store.Transact(ctx); err != nil {
		return err
	}
	defer func() { err = tx.Done(err) }()

	r, err := h.getRepoForPR(ctx, tx, pr, externalServiceID)
	if err != nil {
		log15.Debug("Webhook event could not be matched to repo", "err", err)
		return nil
	}

	var kind btypes.ChangesetEventKind
	if kind, err = btypes.ChangesetEventKindFor(ev); err != nil {
		return err
	}

	cs, err := tx.GetChangeset(ctx, store.GetChangesetOpts{
		RepoID:              r.ID,
		ExternalID:          strconv.FormatInt(pr.ID, 10),
		ExternalServiceType: h.ServiceType,
	})
	if err != nil {
		if err == store.ErrNoResults {
			err = nil // Nothing to do
		}
		return err
	}

	now := h.Store.Clock()()
	event := &btypes.ChangesetEvent{
		ChangesetID: cs.ID,
		Kind:        kind,
		Key:         ev.Key(),
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata:    ev,
	}

	existing, err := tx.GetChangesetEvent(ctx, store.GetChangesetEventOpts{
		ChangesetID: cs.ID,
		Kind:        event.Kind,
		Key:         event.Key,
	})

	if err != nil && err != store.ErrNoResults {
		return err
	}

	if existing != nil {
		// Update is used to create or update the record in the database,
		// but we're actually "patching" the record with specific merge semantics
		// encoded in Update. This is because some webhooks payloads don't contain
		// all the information that we can get from the API, so we only update the
		// bits that we know are more up to date and leave the others as they were.
		if err := existing.Update(event); err != nil {
			return err
		}
		event = existing
	}

	// Add new event
	if err := tx.UpsertChangesetEvents(ctx, event); err != nil {
		return err
	}

	// The webhook may have caused the external state of the changeset to change
	// so we need to update it. We need all events as we may have received more than just the
	// event we are currently handling
	events, _, err := tx.ListChangesetEvents(ctx, store.ListChangesetEventsOpts{
		ChangesetIDs: []int64{cs.ID},
	})
	state.SetDerivedState(ctx, tx.Repos(), cs, events)
	if err := tx.UpdateChangesetCodeHostState(ctx, cs); err != nil {
		return err
	}

	return nil
}

type httpError struct {
	code int
	err  error
}

func (e httpError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("HTTP %d: %v", e.code, e.err)
	}
	return fmt.Sprintf("HTTP %d: %s", e.code, http.StatusText(e.code))
}

func respond(w http.ResponseWriter, code int, v interface{}) {
	switch val := v.(type) {
	case nil:
		w.WriteHeader(code)
	case error:
		if val != nil {
			log15.Error(val.Error())
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(code)
			fmt.Fprintf(w, "%v", val)
		}
	default:
		w.Header().Set("Content-Type", "application/json")
		bs, err := json.Marshal(v)
		if err != nil {
			respond(w, http.StatusInternalServerError, err)
			return
		}

		w.WriteHeader(code)
		if _, err = w.Write(bs); err != nil {
			log15.Error("failed to write response", "error", err)
		}
	}
}
