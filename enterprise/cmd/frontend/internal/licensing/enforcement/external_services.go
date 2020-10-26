package enforcement

import (
	"context"
	"fmt"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/licensing"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

// ExternalServicesStore is implemented by any type that can act as a
// repository for external services (e.g. GitHub, GitLab).
type ExternalServicesStore interface {
	Count(context.Context, db.ExternalServicesListOptions) (int, error)
}

// NewPreCreateExternalServiceHook enforces any per-tier validations prior to
// creating a new external service.
func NewPreCreateExternalServiceHook(externalServices ExternalServicesStore) func(ctx context.Context) error {
	if !licensing.EnforceTiers {
		return nil
	}

	return func(ctx context.Context) error {
		// First we need to determine how many external services are permissible.
		var maxExtSvcCount int
		if mockGetExternalServicesLimit != nil {
			maxExtSvcCount = mockGetExternalServicesLimit(ctx)
		}

		// TODO(flying-robot, unknwon) - wrap licensing Info type with per-tier
		// limits that we can reference here. The count check below will pass
		// the validation right now since maxExternalServices = 0.

		// Next we'll grab the current count of external services.
		extSvcCount, err := externalServices.Count(ctx, db.ExternalServicesListOptions{})
		if err != nil {
			return err
		}

		// If we have none configured or we're under the limit, we can pass the
		// validation. Otherwise an error will be returned. Note that we consider
		// a maximum of 0 to be "unlimited", which is consistent with other checks.
		if extSvcCount == 0 || maxExtSvcCount == 0 || extSvcCount < maxExtSvcCount {
			return nil
		}
		return errcode.NewPresentationError(
			fmt.Sprintf(
				"Unable to create external service: the current plan cannot exceed %d external services (this instance now has %d). Contact Sourcegraph to learn more at https://about.sourcegraph.com/contact/sales.",
				maxExtSvcCount,
				extSvcCount,
			),
		)
	}
}

// These mocks are used for test purposes.
var (
	mockGetExternalServicesLimit func(_ context.Context) int
)
