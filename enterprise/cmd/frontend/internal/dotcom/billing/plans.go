package billing

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/plan"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/license"
)

// InfoForProductPlan returns the license key tags and min/max quantities that should be used for
// the given product plan.
//
// License key tags indicate which product plan the license is for, so they are stored on the
// billing system in the metadata of the product plans.
func InfoForProductPlan(ctx context.Context, planID string) (licenseTags []string, minQuantity, maxQuantity *int32, err error) {
	params := &stripe.PlanParams{Params: stripe.Params{Context: ctx}}
	params.AddExpand("product")
	plan, err := plan.Get(planID, params)
	if err != nil {
		return nil, nil, nil, err
	}

	var tags []string
	switch {
	case plan.Product.Metadata["licenseTags"] != "":
		tags = license.ParseTagsInput(plan.Product.Metadata["licenseTags"])
	default:
		return nil, nil, nil, fmt.Errorf("unable to determine license tags for plan %q (nickname %q)", planID, plan.Nickname)
	}

	minQuantity, maxQuantity = ProductPlanMinMaxQuantity(plan)

	return tags, minQuantity, maxQuantity, nil
}
