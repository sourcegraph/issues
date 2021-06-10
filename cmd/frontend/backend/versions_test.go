package backend

import (
	"context"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
)

func TestGetFirstServiceVersion(t *testing.T) {
	db := dbtest.NewDB(t, "")

	ctx := context.Background()

	if err := UpdateServiceVersion(ctx, db, "service", "1.2.3"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := UpdateServiceVersion(ctx, db, "service", "1.2.4"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := UpdateServiceVersion(ctx, db, "service", "1.3.0"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	firstVersion, err := GetFirstServiceVersion(ctx, db, "service")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if firstVersion != "1.2.3" {
		t.Errorf("unexpected first version. want=%s have=%s", "1.2.3", firstVersion)
	}
}

func TestUpdateServiceVersion(t *testing.T) {
	db := dbtest.NewDB(t, "")

	ctx := context.Background()
	for _, tc := range []struct {
		version string
		err     error
	}{
		{"0.0.0", nil},
		{"0.0.1", nil},
		{"0.1.0", nil},
		{"0.2.0", nil},
		{"1.0.0", nil},
		{"1.2.0", &UpgradeError{
			Service:  "service",
			Previous: semver.MustParse("1.0.0"),
			Latest:   semver.MustParse("1.2.0"),
		}},
		{"2.1.0", &UpgradeError{
			Service:  "service",
			Previous: semver.MustParse("1.0.0"),
			Latest:   semver.MustParse("2.1.0"),
		}},
		{"0.3.0", nil}, // rollback
		{"non-semantic-version-is-always-valid", nil},
		{"1.0.0", nil}, // back to semantic version is allowed
		{"2.1.0", &UpgradeError{
			Service:  "service",
			Previous: semver.MustParse("1.0.0"),
			Latest:   semver.MustParse("2.1.0"),
		}}, // upgrade policy violation returns
	} {
		have := UpdateServiceVersion(ctx, db, "service", tc.version)
		want := tc.err

		if diff := cmp.Diff(have, want); diff != "" {
			t.Fatal(diff)
		}

		t.Logf("version = %q", tc.version)
	}
}

func TestIsValidUpgrade(t *testing.T) {
	for _, tc := range []struct {
		name     string
		previous string
		latest   string
		want     bool
	}{{
		name:     "no versions",
		previous: "",
		latest:   "",
		want:     true,
	}, {
		name:     "no previous version",
		previous: "",
		latest:   "v3.13.0",
		want:     true,
	}, {
		name:     "no latest version",
		previous: "v3.13.0",
		latest:   "",
		want:     true,
	}, {
		name:     "same version",
		previous: "v3.13.0",
		latest:   "v3.13.0",
		want:     true,
	}, {
		name:     "one minor version up",
		previous: "v3.12.4",
		latest:   "v3.13.1",
		want:     true,
	}, {
		name:     "one patch version up",
		previous: "v3.12.4",
		latest:   "v3.12.5",
		want:     true,
	}, {
		name:     "two patch versions up",
		previous: "v3.12.4",
		latest:   "v3.12.6",
		want:     true,
	}, {
		name:     "one major version up",
		previous: "v3.13.1",
		latest:   "v4.0.0",
		want:     true,
	}, {
		name:     "more than one minor version up",
		previous: "v3.9.4",
		latest:   "v3.11.0",
		want:     false,
	}, {
		name:     "major jump",
		previous: "v3.9.4",
		latest:   "v4.1.0",
		want:     false,
	}, {
		name:     "major rollback",
		previous: "v4.1.0",
		latest:   "v3.9.4",
		want:     true,
	}, {
		name:     "minor rollback",
		previous: "v4.1.0",
		latest:   "v4.0.4",
		want:     true,
	}, {
		name:     "patch rollback",
		previous: "v4.1.4",
		latest:   "v4.1.3",
		want:     true,
	},
	} {
		t.Run(tc.name, func(t *testing.T) {
			previous, _ := semver.NewVersion(tc.previous)
			latest, _ := semver.NewVersion(tc.latest)

			if got := IsValidUpgrade(previous, latest); got != tc.want {
				t.Errorf(
					"IsValidUpgrade(previous: %s, latest: %s) = %t, want %t",
					tc.previous,
					tc.latest,
					got,
					tc.want,
				)
			}
		})
	}
}
