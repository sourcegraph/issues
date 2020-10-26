package resolvers

import (
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	ee "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
)

type changesetEventResolver struct {
	store             *ee.Store
	httpFactory       *httpcli.Factory
	changesetResolver *changesetResolver
	*campaigns.ChangesetEvent
}

const changesetEventIDKind = "ChangesetEvent"

func marshalChangesetEventID(id int64) graphql.ID {
	return relay.MarshalID(changesetEventIDKind, id)
}

func (r *changesetEventResolver) ID() graphql.ID {
	return marshalChangesetEventID(r.ChangesetEvent.ID)
}

func (r *changesetEventResolver) CreatedAt() graphqlbackend.DateTime {
	return graphqlbackend.DateTime{Time: r.ChangesetEvent.CreatedAt}
}

func (r *changesetEventResolver) Changeset() graphqlbackend.ExternalChangesetResolver {
	return r.changesetResolver
}
