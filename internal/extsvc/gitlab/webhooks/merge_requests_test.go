package webhooks

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
)

func TestMergeRequestDowncast(t *testing.T) {
	t.Run("invalid actions", func(t *testing.T) {
		for _, action := range []string{
			"",
			"not a valid action",
		} {
			t.Run(action, func(t *testing.T) {
				mre := &mergeRequestEvent{
					ObjectAttributes: mergeRequestEventObjectAttributes{
						Action: action,
					},
				}
				dc, err := mre.downcast()
				if !errors.Is(err, ErrObjectKindUnknown) {
					t.Errorf("unexpected error: %+v", err)
				}
				if dc != nil {
					t.Errorf("unexpected non-nil value: %+v", dc)
				}
			})
		}
	})

	t.Run("valid actions", func(t *testing.T) {
		for _, tc := range []struct {
			action string
			want   string
		}{
			{action: "approved", want: "*webhooks.MergeRequestApprovedEvent"},
			{action: "close", want: "*webhooks.MergeRequestCloseEvent"},
			{action: "merge", want: "*webhooks.MergeRequestMergeEvent"},
			{action: "reopen", want: "*webhooks.MergeRequestReopenEvent"},
			{action: "unapproved", want: "*webhooks.MergeRequestUnapprovedEvent"},
			{action: "update", want: "*webhooks.MergeRequestUpdateEvent"},
		} {
			t.Run(tc.want, func(t *testing.T) {
				mre := &mergeRequestEvent{
					EventCommon: EventCommon{
						ObjectKind: "merge_request",
					},
					User:   &gitlab.User{},
					Labels: new([]gitlab.Label),
					ObjectAttributes: mergeRequestEventObjectAttributes{
						MergeRequest: &gitlab.MergeRequest{},
						Action:       tc.action,
					},
				}
				dc, err := mre.downcast()
				if err != nil {
					t.Errorf("unexpected non-nil error: %+v", err)
				}
				if have := reflect.TypeOf(dc).String(); have != tc.want {
					t.Errorf("unexpected downcasted type: have %s; want %s", have, tc.want)
				}

				mr := dc.(MergeRequestEventContainer).ToEvent()
				if diff := cmp.Diff(mre.EventCommon, mr.EventCommon); diff != "" {
					t.Errorf("mismatched EventCommon: %s", diff)
				}
				if mr.User != mre.User {
					t.Errorf("mismatched User: have %p; want %p", mr.User, mre.User)
				}
				if mr.Labels != mre.Labels {
					t.Errorf("mismatched Labels: have %p; want %p", mr.Labels, mre.Labels)
				}
				if mr.MergeRequest != mre.ObjectAttributes.MergeRequest {
					t.Errorf("mismatched User: have %p; want %p", mr.MergeRequest, mre.ObjectAttributes.MergeRequest)
				}
			})
		}
	})
}
