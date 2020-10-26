package licensing

import (
	"github.com/sourcegraph/sourcegraph/internal/conf"
)

func init() {
	conf.ContributeValidator(func(cfg conf.Unified) conf.Problems {
		if cfg.SiteConfiguration.LicenseKey != "" {
			if _, _, err := ParseProductLicenseKeyWithBuiltinOrGenerationKey(cfg.SiteConfiguration.LicenseKey); err != nil {
				return conf.NewSiteProblems("should provide a valid license key")
			}
		}
		return nil
	})
}
