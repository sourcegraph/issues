package campaigns

import (
	"github.com/sourcegraph/sourcegraph/internal/conf"
)

func init() {
	conf.ContributeValidator(func(c conf.Unified) (problems conf.Problems) {
		if exp := c.ExperimentalFeatures; exp != nil && exp.Automation != "" {
			problems = append(problems, conf.NewSiteProblem("The `{ experimentalFeatures: { automation: \"enabled\" } }` feature flag was deprecated. Campaigns are now enabled by default. Set `campaigns.enabled` to `false` to disable it."))
		}

		if c.AutomationReadAccessEnabled != nil {
			problems = append(problems, conf.NewSiteProblem("The `automation.readAccess.enabled` property is deprecated and will be removed in a future release. Use `campaigns.enabled` to enable or disable the campaigns feature instead."))
		}

		if c.CampaignsReadAccessEnabled != nil {
			problems = append(problems, conf.NewSiteProblem("The `campaigns.readAccess.enabled` property is deprecated and will be removed in a future release. Use `campaigns.enabled` to enable or disable the campaigns feature instead."))
		}

		return
	})
}
