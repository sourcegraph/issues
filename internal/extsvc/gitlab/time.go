package gitlab

import (
	"encoding/json"
	"time"
)

// Time is a type that can unmarshal the multiple date/time formats that GitLab
// webhooks may include. While GitLab's normal API uses RFC 3339 dates, some
// objects in webhook payloads include a more legacy format, even though they
// generally adhere to the REST API otherwise. We need to be able to handle
// both to be able to handle those types in a unified way.
//
// The underlying GitLab issue is
// https://gitlab.com/gitlab-org/gitlab/-/issues/19567
type Time struct{ time.Time }

func (t *Time) UnmarshalJSON(data []byte) error {
	// First, try the normal RFC 3339 decoding.
	if err := t.Time.UnmarshalJSON(data); err == nil {
		return nil
	}

	// Now let's try their other format, which looks like this:
	// 2020-06-26 23:11:17 UTC
	//
	// First, we need to get the string itself.
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	dec, err := time.Parse("2006-01-02 15:04:05 MST", s)
	if err == nil {
		t.Time = dec
	}
	return err
}
