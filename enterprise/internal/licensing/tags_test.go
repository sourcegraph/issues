package licensing

import (
	"fmt"
	"testing"
)

func TestProductNameWithBrand(t *testing.T) {
	tests := []struct {
		hasLicense  bool
		licenseTags []string
		want        string
	}{
		{hasLicense: false, want: "Sourcegraph Free"},
		{hasLicense: true, licenseTags: nil, want: "Sourcegraph Enterprise"},
		{hasLicense: true, licenseTags: []string{}, want: "Sourcegraph Enterprise"},
		{hasLicense: true, licenseTags: []string{"x"}, want: "Sourcegraph Enterprise"}, // unrecognized tag "x" is ignored
		{hasLicense: true, licenseTags: []string{"starter"}, want: "Sourcegraph Enterprise Starter"},
		{hasLicense: true, licenseTags: []string{"trial"}, want: "Sourcegraph Enterprise (trial)"},
		{hasLicense: true, licenseTags: []string{"dev"}, want: "Sourcegraph Enterprise (dev use only)"},
		{hasLicense: true, licenseTags: []string{"starter", "trial"}, want: "Sourcegraph Enterprise Starter (trial)"},
		{hasLicense: true, licenseTags: []string{"starter", "dev"}, want: "Sourcegraph Enterprise Starter (dev use only)"},
		{hasLicense: true, licenseTags: []string{"starter", "trial", "dev"}, want: "Sourcegraph Enterprise Starter (trial, dev use only)"},
		{hasLicense: true, licenseTags: []string{"trial", "dev"}, want: "Sourcegraph Enterprise (trial, dev use only)"},

		{hasLicense: true, licenseTags: []string{"team"}, want: "Sourcegraph Team"},
		{hasLicense: true, licenseTags: []string{"starter", "team"}, want: "Sourcegraph Team"}, // Team should overrule the old Starter plan
		{hasLicense: true, licenseTags: []string{"team", "trial"}, want: "Sourcegraph Team (trial)"},
		{hasLicense: true, licenseTags: []string{"team", "dev"}, want: "Sourcegraph Team (dev use only)"},
		{hasLicense: true, licenseTags: []string{"team", "dev", "trial"}, want: "Sourcegraph Team (trial, dev use only)"},

		{hasLicense: true, licenseTags: []string{"plan:team-0"}, want: "Sourcegraph Team"},
		{hasLicense: true, licenseTags: []string{"plan:team-0", "starter"}, want: "Sourcegraph Team"}, // Team should overrule the old Starter plan
		{hasLicense: true, licenseTags: []string{"plan:team-0", "trial"}, want: "Sourcegraph Team (trial)"},
		{hasLicense: true, licenseTags: []string{"plan:team-0", "dev"}, want: "Sourcegraph Team (dev use only)"},
		{hasLicense: true, licenseTags: []string{"plan:team-0", "dev", "trial"}, want: "Sourcegraph Team (trial, dev use only)"},

		{hasLicense: true, licenseTags: []string{"plan:enterprise-0"}, want: "Sourcegraph Enterprise"},
		{hasLicense: true, licenseTags: []string{"plan:enterprise-0", "starter"}, want: "Sourcegraph Enterprise"}, // Enterprise should overrule the old Starter plan
		{hasLicense: true, licenseTags: []string{"plan:enterprise-0", "trial"}, want: "Sourcegraph Enterprise (trial)"},
		{hasLicense: true, licenseTags: []string{"plan:enterprise-0", "dev"}, want: "Sourcegraph Enterprise (dev use only)"},
		{hasLicense: true, licenseTags: []string{"plan:enterprise-0", "dev", "trial"}, want: "Sourcegraph Enterprise (trial, dev use only)"},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("hasLicense=%v licenseTags=%v", test.hasLicense, test.licenseTags), func(t *testing.T) {
			if got := ProductNameWithBrand(test.hasLicense, test.licenseTags); got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}
