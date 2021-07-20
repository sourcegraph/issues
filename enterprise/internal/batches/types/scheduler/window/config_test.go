package window

import (
	"math"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-multierror"

	"github.com/sourcegraph/sourcegraph/schema"
)

// We have a bunch of tests in here that rely on unexported fields in the window
// structs. Since we control all of this, we're going to provide a common set of
// options that will allow that.
var (
	cmpAllowUnexported = cmp.AllowUnexported(Window{}, rate{})
	cmpOptions         = cmp.Options{cmpAllowUnexported}
)

func timeOfDayPtr(hour, minute int8) *timeOfDay {
	t := timeOfDayFromParts(hour, minute)
	return &t
}

func TestConfiguration_Estimate(t *testing.T) {
	t.Run("no windows", func(t *testing.T) {
		cfg := &Configuration{}
		now := time.Now()

		if have := cfg.Estimate(now, 1000); have == nil {
			t.Error("unexpected nil estimate")
		} else if *have != now {
			t.Errorf("unexpected estimate: have=%v want=%v", *have, now)
		}
	})

	t.Run("multiple windows", func(t *testing.T) {
		// Let's set up a configuration that looks roughly like this:
		//
		// |  Mon  |  Tue  |  Wed  |  Thu  |  Fri  |  Sat  |  Sun  |
		// |-------|-------|-------|-------|-------|-------|-------|
		// | 10/hr | 20/hr | 10/hr | 0     | 10/hr | 0     | ∞     |
		makeWindow := func(day time.Weekday, n int) Window {
			return Window{
				days: newWeekdaySet(day),
				rate: rate{n: n, unit: ratePerHour},
			}
		}
		cfg := &Configuration{
			windows: []Window{
				makeWindow(time.Monday, 10),
				makeWindow(time.Tuesday, 20),
				makeWindow(time.Wednesday, 10),
				makeWindow(time.Thursday, 0),
				makeWindow(time.Friday, 10),
				// Saturday intentionally omitted.
				makeWindow(time.Sunday, -1),
			},
		}

		// For convenience, let's also set up a time at 12:00 each day.
		var (
			monday    = time.Date(2021, 4, 5, 12, 0, 0, 0, time.UTC)
			tuesday   = time.Date(2021, 4, 6, 12, 0, 0, 0, time.UTC)
			wednesday = time.Date(2021, 4, 7, 12, 0, 0, 0, time.UTC)
			thursday  = time.Date(2021, 4, 8, 12, 0, 0, 0, time.UTC)
			friday    = time.Date(2021, 4, 9, 12, 0, 0, 0, time.UTC)
			saturday  = time.Date(2021, 4, 10, 12, 0, 0, 0, time.UTC)
			sunday    = time.Date(2021, 4, 11, 12, 0, 0, 0, time.UTC)
		)

		for name, tc := range map[string]struct {
			now  time.Time
			n    int
			want time.Time
		}{
			"right now because the window is unlimited": {
				now:  sunday,
				n:    1000,
				want: sunday,
			},
			"right now because n is 0 and a window is open": {
				now:  monday,
				n:    0,
				want: monday,
			},
			"not right now, even though n is 0, because nothing is done until tomorrow": {
				now:  saturday,
				n:    0,
				want: sunday.Truncate(24 * time.Hour),
			},
			"in an hour": {
				now:  tuesday,
				n:    20,
				want: tuesday.Add(1 * time.Hour),
			},
			"at the very end of the day's schedule": {
				now:  wednesday,
				n:    120,
				want: thursday.Truncate(24 * time.Hour),
			},
			"the next time a window is open, plus an hour, since we're asking for the 10th item with a 10/hr limit": {
				now:  thursday,
				n:    10,
				want: friday.Truncate(24 * time.Hour).Add(1 * time.Hour),
			},
		} {
			t.Run(name, func(t *testing.T) {
				have := cfg.Estimate(tc.now, tc.n)
				if have == nil {
					t.Error("unexpected nil estimate")
				} else if diff := time.Duration(math.Abs(float64(have.Sub(tc.want)))); diff > 1*time.Millisecond {
					// There's some floating point maths involved in the
					// estimation process, so we'll be happy if this is within a
					// millisecond (which is still _wildly_ more accurate than
					// any reasonable expectation).
					t.Errorf("unexpected estimate: have=%v want=%v", *have, tc.want)
				}
			})
		}
	})

	t.Run("nil estimates", func(t *testing.T) {
		for name, tc := range map[string]struct {
			cfg *Configuration
			now time.Time
			n   int
		}{
			"all zeroes": {
				cfg: &Configuration{
					windows: []Window{
						{days: newWeekdaySet(), rate: rate{n: 0}},
					},
				},
				now: time.Now(),
				n:   0,
			},
			"more than a week in the future": {
				cfg: &Configuration{
					windows: []Window{
						{days: newWeekdaySet(), rate: rate{n: 1, unit: ratePerHour}},
					},
				},
				now: time.Now(),
				n:   24*7 + 1,
			},
		} {
			t.Run(name, func(t *testing.T) {
				if have := tc.cfg.Estimate(tc.now, tc.n); have != nil {
					t.Errorf("unexpected non-nil estimate: %v", *have)
				}
			})
		}
	})
}

func TestConfiguration_Schedule(t *testing.T) {
	// We have other tests to test the actual implementation of scheduleAt();
	// this is purely to ensure that we do the special case handling of not
	// having rollout windows correctly.
	//
	// Since we do _not_ control the current time here, any configurations below
	// must have the same windows across the entire week.
	for name, tc := range map[string]struct {
		cfg          *Configuration
		wantDuration time.Duration
		wantRate     rate
	}{
		"no rollout windows": {
			cfg: &Configuration{
				windows: []Window{},
			},
			wantDuration: 10 * time.Minute,
			wantRate:     rate{n: -1},
		},
		"rollout windows": {
			cfg: &Configuration{
				windows: []Window{
					{days: newWeekdaySet(), rate: rate{n: 40, unit: ratePerHour}},
				},
			},
			wantDuration: 24 * time.Hour,
			wantRate:     rate{n: 40, unit: ratePerHour},
		},
	} {
		t.Run(name, func(t *testing.T) {
			have := tc.cfg.Schedule()
			if have.duration != tc.wantDuration {
				t.Errorf("unexpected schedule duration: have=%v want=%v", have.duration, tc.wantDuration)
			}
			if have.rate != tc.wantRate {
				t.Errorf("unexpected schedule rate: have=%v want=%v", have.rate, tc.wantRate)
			}
		})
	}
}

func TestConfiguration_currentFor(t *testing.T) {
	// Let's set up some common windows to simplify defining the test cases.

	// The window is always unlimited at zombo.com.
	zombo := Window{
		days: newWeekdaySet(),
		rate: makeUnlimitedRate(),
	}

	// Restrict to a crawl on Friday afternoons because the ops team is drunk.
	friday := Window{
		days:  newWeekdaySet(time.Friday),
		start: timeOfDayPtr(15, 0),
		end:   timeOfDayPtr(23, 0),
		rate:  rate{n: 1, unit: ratePerHour},
	}

	// Every day we shut down for breakfast. It's the most important meal of the
	// day!
	breakfast := Window{
		days:  newWeekdaySet(),
		start: timeOfDayPtr(8, 0),
		end:   timeOfDayPtr(9, 0),
		rate:  rate{n: 0},
	}

	// But we might also use coffee to go super fast.
	coffee := Window{
		days:  newWeekdaySet(),
		start: timeOfDayPtr(8, 30),
		end:   timeOfDayPtr(9, 0),
		rate:  rate{n: 100, unit: ratePerSecond},
	}

	// Finally, we have a day of rest, where we have no start or end times, but
	// a weekday restriction.
	sunday := Window{
		days: newWeekdaySet(time.Sunday),
		rate: rate{n: 0},
	}

	// And some useful times.
	thursday0815 := time.Date(2021, 4, 1, 8, 15, 0, 0, time.UTC)
	friday1900 := time.Date(2021, 4, 2, 19, 0, 0, 0, time.UTC)
	sunday0600 := time.Date(2021, 4, 4, 6, 0, 0, 0, time.UTC)

	newDuration := func(d time.Duration) *time.Duration { return &d }

	for name, tc := range map[string]struct {
		cfg          *Configuration
		when         time.Time
		wantWindow   *Window
		wantDuration *time.Duration
	}{
		"no windows": {
			cfg:          &Configuration{},
			when:         time.Now(),
			wantWindow:   nil,
			wantDuration: nil,
		},
		"single, unlimited window": {
			cfg: &Configuration{
				windows: []Window{zombo},
			},
			when:         time.Now(),
			wantWindow:   &zombo,
			wantDuration: nil,
		},
		"multiple windows, but zombo always wins": {
			cfg: &Configuration{
				windows: []Window{friday, zombo},
			},
			when:         friday1900,
			wantWindow:   &zombo,
			wantDuration: nil,
		},
		"multiple windows, but Friday wins": {
			cfg: &Configuration{
				windows: []Window{zombo, friday},
			},
			when:         friday1900,
			wantWindow:   &friday,
			wantDuration: newDuration(4 * time.Hour),
		},
		"multiple overlapping windows causing the current window to end early": {
			cfg: &Configuration{
				windows: []Window{zombo, breakfast, coffee},
			},
			when:         thursday0815,
			wantWindow:   &breakfast,
			wantDuration: newDuration(15 * time.Minute),
		},
		"duration calculated without an end time in the window": {
			cfg: &Configuration{
				windows: []Window{sunday},
			},
			when:         sunday0600,
			wantWindow:   &sunday,
			wantDuration: newDuration(18 * time.Hour),
		},
		"duration calculated without an end time in the window, but with an overlap": {
			cfg: &Configuration{
				windows: []Window{zombo, breakfast},
			},
			when:         sunday0600,
			wantWindow:   &zombo,
			wantDuration: newDuration(2 * time.Hour),
		},
		"no current window": {
			cfg: &Configuration{
				windows: []Window{breakfast, coffee},
			},
			when:       friday1900,
			wantWindow: nil,
			// 13 hours because it's 19:00, and the next window is at 08:00 the
			// next day.
			wantDuration: newDuration(13 * time.Hour),
		},
	} {
		t.Run(name, func(t *testing.T) {
			haveWindow, haveDuration := tc.cfg.windowFor(tc.when)

			if tc.wantWindow == nil {
				if haveWindow != nil {
					t.Errorf("unexpected non-nil window: have=%v", *haveWindow)
				}
			} else if haveWindow == nil {
				t.Errorf("unexpected nil window: want=%v", *tc.wantWindow)
			} else if diff := cmp.Diff(*haveWindow, *tc.wantWindow, cmpOptions); diff != "" {
				t.Errorf("unexpected window (-have +want):\n%s", diff)
			}

			if tc.wantDuration == nil {
				if haveDuration != nil {
					t.Errorf("unexpected non-nil duration: have=%v", *haveDuration)
				}
			} else if haveDuration == nil {
				t.Errorf("unexpected nil duration: want=%v", *tc.wantDuration)
			} else if *haveDuration != *tc.wantDuration {
				t.Errorf("unexpected duration: have=%v want=%v", *haveDuration, *tc.wantDuration)
			}
		})
	}
}

func TestParseConfiguration(t *testing.T) {
	t.Run("errors", func(t *testing.T) {
		for name, tc := range map[string]struct {
			in   *[]*schema.BatchChangeRolloutWindow
			want int
		}{
			"one bad window": {
				in: &[]*schema.BatchChangeRolloutWindow{
					{Rate: "xx"},
					{Rate: 0},
				},
				want: 1,
			},
			"two bad windows, ha ha ha": {
				in: &[]*schema.BatchChangeRolloutWindow{
					{Rate: "xx"},
					{Rate: "yy"},
				},
				want: 2,
			},
		} {
			t.Run(name, func(t *testing.T) {
				_, err := parseConfiguration(tc.in)

				var e *multierror.Error
				if !errors.As(err, &e) || len(e.Errors) != tc.want {
					t.Errorf("unexpected number of errors: have=%d want=%d", len(e.Errors), tc.want)
				}
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		for name, tc := range map[string]struct {
			in   *[]*schema.BatchChangeRolloutWindow
			want *Configuration
		}{
			"nil": {
				in:   nil,
				want: &Configuration{windows: []Window{}},
			},
			"valid windows": {
				in: &[]*schema.BatchChangeRolloutWindow{
					{
						Rate:  "20/hr",
						Days:  []string{"monday"},
						Start: "01:15",
						End:   "02:30",
					},
					{
						Rate: "2/hr",
					},
				},
				want: &Configuration{
					windows: []Window{
						{
							rate:  rate{n: 20, unit: ratePerHour},
							days:  newWeekdaySet(time.Monday),
							start: timeOfDayPtr(1, 15),
							end:   timeOfDayPtr(2, 30),
						},
						{
							rate: rate{n: 2, unit: ratePerHour},
							days: newWeekdaySet(),
						},
					},
				},
			},
		} {
			t.Run(name, func(t *testing.T) {
				if have, err := parseConfiguration(tc.in); err != nil {
					t.Errorf("unexpected error: %v", err)
				} else if diff := cmp.Diff(have, tc.want.windows, cmpOptions); diff != "" {
					t.Errorf("unexpected configuration (-have +want):\n%s", diff)
				}
			})
		}
	})
}
