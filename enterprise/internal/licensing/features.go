package licensing

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

// EnforceTiers is a temporary flag to indicate whether to enforce new license tier constraints defined in
// RFC 167 to incrementally merge changes into main branch, we'll remove it once fully implemented the RFC.
var EnforceTiers, _ = strconv.ParseBool(env.Get("SRC_ENFORCE_TIERS", "false", "Enforce license tier constraints defined in RFC 167"))

// Feature is a product feature that is selectively activated based on the current license key.
type Feature string

// Check checks whether the feature is activated based on the current license. If it is
// disabled, it returns a non-nil error.
//
// The returned error may implement errcode.PresentationError to indicate that it can be displayed
// directly to the user. Use IsFeatureNotActivated to distinguish between the error reasons.
func Check(feature Feature) error {
	if MockCheckFeature != nil {
		return MockCheckFeature(feature)
	}

	info, err := GetConfiguredProductLicenseInfo()
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("checking feature %q activation", feature))
	}
	return checkFeature(info, feature)
}

func checkFeature(info *Info, feature Feature) error {
	if info == nil {
		return newFeatureNotActivatedError(fmt.Sprintf("The feature %q is not activated because it requires a valid Sourcegraph license. Purchase a Sourcegraph subscription to activate this feature.", feature))
	}

	// Check if the feature is explicitly allowed via license tag.
	hasFeature := func(want Feature) bool {
		for _, t := range info.Tags {
			if Feature(t) == want {
				return true
			}
		}
		return false
	}
	if !info.Plan().HasFeature(feature) && !hasFeature(feature) {
		return newFeatureNotActivatedError(fmt.Sprintf("The feature %q is not activated in your Sourcegraph license. Upgrade your Sourcegraph subscription to use this feature.", feature))
	}
	return nil // feature is activated for current license
}

// MockCheckFeature is for mocking Check in tests.
var MockCheckFeature func(feature Feature) error

// TestingSkipFeatureChecks is for tests that want to mock Check to always return nil (i.e.,
// behave as though the current license enables all features).
//
// It returns a cleanup func so callers can use `defer TestingSkipFeatureChecks()()` in a test body.
func TestingSkipFeatureChecks() func() {
	MockCheckFeature = func(Feature) error { return nil }
	return func() { MockCheckFeature = nil }
}

func newFeatureNotActivatedError(message string) featureNotActivatedError {
	e := errcode.NewPresentationError(message).(errcode.PresentationError)
	return featureNotActivatedError{e}
}

type featureNotActivatedError struct{ errcode.PresentationError }

// IsFeatureNotActivated reports whether err indicates that the license is valid but does not
// activate the feature.
//
// It is used to distinguish between the multiple reasons for errors from Check: either
// failed license verification, or a valid license that does not activate a feature (e.g.,
// Enterprise Starter not including an Enterprise-only feature).
func IsFeatureNotActivated(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(featureNotActivatedError)
	if !ok {
		// Also check for the pointer type to guard against stupid mistakes.
		_, ok = err.(*featureNotActivatedError)
	}
	return ok
}

// IsFeatureEnabledLenient reports whether the current license enables the given feature. If there
// is an error reading the license, it is lenient and returns true.
//
// This is useful for callers who don't want to handle errors (usually because the user would be
// prevented from getting to this point if license verification had failed, so it's not necessary to
// handle license verification errors here).
func IsFeatureEnabledLenient(feature Feature) bool {
	return !IsFeatureNotActivated(Check(feature))
}
