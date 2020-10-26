package billing

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/plan"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
)

// productPlan implements the GraphQL type ProductPlan.
type productPlan struct {
	billingPlanID       string
	productPlanID       string
	name                string
	pricePerUserPerYear int32
	minQuantity         *int32
	maxQuantity         *int32
	tiersMode           string
	planTiers           []graphqlbackend.PlanTier
}

// planTier implements the GraphQL type PlanTier.
type planTier struct {
	unitAmount int64
	upTo       int64
	flatAmount int64
}

func (r *productPlan) ProductPlanID() string      { return r.productPlanID }
func (r *productPlan) BillingPlanID() string      { return r.billingPlanID }
func (r *productPlan) Name() string               { return r.name }
func (r *productPlan) NameWithBrand() string      { return "Sourcegraph " + r.name }
func (r *productPlan) PricePerUserPerYear() int32 { return r.pricePerUserPerYear }
func (r *productPlan) MinQuantity() *int32        { return r.minQuantity }
func (r *productPlan) MaxQuantity() *int32        { return r.maxQuantity }
func (r *productPlan) TiersMode() string          { return r.tiersMode }
func (r *productPlan) PlanTiers() []graphqlbackend.PlanTier {
	if r.planTiers == nil {
		return nil
	}
	return r.planTiers
}

func (r *planTier) UnitAmount() int32 { return int32(r.unitAmount) }
func (r *planTier) UpTo() int32       { return int32(r.upTo) }
func (r *planTier) FlatAmount() int32 { return int32(r.flatAmount) }

// ToProductPlan returns a resolver for the GraphQL type ProductPlan from the given billing plan.
func ToProductPlan(plan *stripe.Plan) (graphqlbackend.ProductPlan, error) {
	// Sanity check.
	if plan.Product.Name == "" {
		return nil, fmt.Errorf("unexpected empty product name for plan %q", plan.ID)
	}
	if plan.Currency != stripe.CurrencyUSD {
		return nil, fmt.Errorf("unexpected currency %q for plan %q", plan.Currency, plan.ID)
	}
	if plan.IntervalCount != 1 {
		return nil, fmt.Errorf("unexpected plan interval count %d for plan %q", plan.IntervalCount, plan.ID)
	}

	var tiers []graphqlbackend.PlanTier
	for _, tier := range plan.Tiers {
		tiers = append(tiers, &planTier{
			flatAmount: tier.FlatAmount,
			unitAmount: tier.UnitAmount,
			upTo:       tier.UpTo,
		})
	}

	minQuantity, maxQuantity := ProductPlanMinMaxQuantity(plan)

	return &productPlan{
		productPlanID:       plan.Product.ID,
		billingPlanID:       plan.ID,
		name:                plan.Product.Name,
		pricePerUserPerYear: int32(plan.Amount),
		minQuantity:         minQuantity,
		maxQuantity:         maxQuantity,
		planTiers:           tiers,
		tiersMode:           plan.TiersMode,
	}, nil
}

// ProductPlanMinMaxQuantity returns the plan's product's minQuantity and maxQuantity metadata
// values, or nil if unset.
func ProductPlanMinMaxQuantity(plan *stripe.Plan) (min, max *int32) {
	get := func(key string) *int32 {
		if v, err := strconv.Atoi(plan.Product.Metadata[key]); err == nil {
			tmp := int32(v)
			return &tmp
		}
		return nil
	}
	return get("minQuantity"), get("maxQuantity")
}

// ProductPlans implements the GraphQL field Query.dotcom.productPlans.
func (BillingResolver) ProductPlans(ctx context.Context) ([]graphqlbackend.ProductPlan, error) {
	params := &stripe.PlanListParams{
		ListParams: stripe.ListParams{Context: ctx},
		Active:     stripe.Bool(true),
	}
	params.AddExpand("data.product")
	plans := plan.List(params)
	var gqlPlans []graphqlbackend.ProductPlan
	for plans.Next() {
		plan := plans.Plan()
		if plan.Interval != stripe.PlanIntervalYear {
			continue
		}
		if !plan.Product.Active || !plan.Active {
			continue
		}
		gqlPlan, err := ToProductPlan(plan)
		if err != nil {
			return nil, err
		}
		gqlPlans = append(gqlPlans, gqlPlan)
	}
	if err := plans.Err(); err != nil {
		return nil, err
	}

	// Sort free first, cheapest first (a reasonable assumption).
	sort.Slice(gqlPlans, func(i, j int) bool {
		fi := gqlPlans[i].PlanTiers() == nil && gqlPlans[i].PricePerUserPerYear() == 0
		fj := gqlPlans[j].PlanTiers() == nil && gqlPlans[j].PricePerUserPerYear() == 0
		return (fi && !fj) || (fi == fj &&
			gqlPlans[i].PricePerUserPerYear() < gqlPlans[j].PricePerUserPerYear())
	})

	return gqlPlans, nil
}
