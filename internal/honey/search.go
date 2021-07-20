package honey

import (
	"context"

	"github.com/honeycombio/libhoney-go"

	"github.com/sourcegraph/sourcegraph/internal/actor"
)

type SearchEventArgs struct {
	OriginalQuery string
	Typ           string
	Source        string
	Status        string
	AlertType     string
	DurationMs    int64
	ResultSize    int
}

// SearchEvent returns a honey event for the dataset "search".
func SearchEvent(ctx context.Context, args SearchEventArgs) *libhoney.Event {
	var act actor.Actor
	if a := actor.FromContext(ctx); a != nil {
		act = *a
	}
	ev := Event("search")
	ev.AddField("query", args.OriginalQuery)
	ev.AddField("actor_uid", act.UID)
	ev.AddField("actor_internal", act.Internal)
	ev.AddField("type", args.Typ)
	ev.AddField("source", args.Source)
	ev.AddField("status", args.Status)
	ev.AddField("alert_type", args.AlertType)
	ev.AddField("duration_ms", args.DurationMs)
	ev.AddField("result_size", args.ResultSize)
	return ev
}
