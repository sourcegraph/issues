package billing

import (
	"context"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
)

func init() {
	// TODO(efritz) - de-globalize assignments in this function
	graphqlbackend.UserURLForSiteAdminBilling = func(ctx context.Context, userID int32) (*string, error) {
		// 🚨 SECURITY: Only site admins may view the billing URL, because it may contain sensitive
		// data or identifiers.
		if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
			return nil, err
		}
		custID, err := dbBilling{}.getUserBillingCustomerID(ctx, nil, userID)
		if err != nil {
			return nil, err
		}
		if custID != nil {
			u := CustomerURL(*custID)
			return &u, nil
		}
		return nil, nil
	}
}

func (BillingResolver) SetUserBilling(ctx context.Context, args *graphqlbackend.SetUserBillingArgs) (*graphqlbackend.EmptyResponse, error) {
	// 🚨 SECURITY: Only site admins may set a user's billing info.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}

	userID, err := graphqlbackend.UnmarshalUserID(args.User)
	if err != nil {
		return nil, err
	}

	// Ensure the billing customer ID refers to a valid customer in the billing system.
	if args.BillingCustomerID != nil {
		if _, err := customer.Get(*args.BillingCustomerID, &stripe.CustomerParams{Params: stripe.Params{Context: ctx}}); err != nil {
			return nil, err
		}
	}

	if err := (dbBilling{}).setUserBillingCustomerID(ctx, nil, userID, args.BillingCustomerID); err != nil {
		return nil, err
	}
	return &graphqlbackend.EmptyResponse{}, nil
}
