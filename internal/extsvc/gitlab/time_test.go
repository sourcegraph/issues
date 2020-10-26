package gitlab

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTimeUnmarshal(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		for _, tc := range []struct {
			input string
			want  time.Time
		}{
			{
				input: "2020-06-26T20:54:05+00:00",
				want:  time.Date(2020, 6, 26, 20, 54, 05, 0, time.UTC),
			},
			{
				input: "2019-06-05T14:32:20.211Z",
				want:  time.Date(2019, 6, 5, 14, 32, 20, 211000000, time.UTC),
			},
			{
				input: "2020-06-24 00:05:18 UTC",
				want:  time.Date(2020, 6, 24, 0, 5, 18, 0, time.UTC),
			},
		} {
			t.Run(tc.input, func(t *testing.T) {
				js, err := json.Marshal(tc.input)
				if err != nil {
					t.Fatal(err)
				}

				var have Time
				if err := json.Unmarshal(js, &have); err != nil {
					t.Errorf("unexpected non-nil error: %+v", err)
				}
				if !have.Time.Equal(tc.want) {
					t.Errorf("incorrect time: have %s; want %s", have, tc.want)
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		for _, input := range []string{
			``,
			`42`,
			`false`,
			`[]`,
			`{}`,
			`""`,
			`"not a valid date"`,
			`"2020-06-24"`,
		} {
			t.Run(input, func(t *testing.T) {
				var out Time
				if err := json.Unmarshal([]byte(input), &out); err == nil {
					t.Error("unexpected nil error")
				}
			})
		}
	})
}
