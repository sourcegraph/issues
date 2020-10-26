package licensing

import "strings"

// A Plan is a pricing plan, with an associated set of features that it offers.
type Plan string

// HasFeature reports whether the plan has the given feature.
func (p Plan) HasFeature(feature Feature) bool {
	for _, f := range planFeatures[p] {
		if feature == f {
			return true
		}
	}
	return false
}

const planTagPrefix = "plan:"

// tag is the representation of the plan as a tag in a license key.
func (p Plan) tag() string { return planTagPrefix + string(p) }

// isKnown reports whether the plan is a known plan.
func (p Plan) isKnown() bool {
	for _, plan := range allPlans {
		if p == plan {
			return true
		}
	}
	return false
}

// Plan is the pricing plan of the license.
func (info *Info) Plan() Plan {
	for _, tag := range info.Tags {
		// A tag that begins with "plan:" indicates the license's plan.
		if strings.HasPrefix(tag, planTagPrefix) {
			plan := Plan(tag[len(planTagPrefix):])
			if plan.isKnown() {
				return plan
			}
		}

		// Backcompat: support the old "starter" tag (which mapped to "Enterprise Starter").
		if tag == "starter" {
			return oldEnterpriseStarter
		}
	}

	// Backcompat: no tags means it is the old "Enterprise" plan.
	return oldEnterprise
}
