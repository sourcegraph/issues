package billing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
)

// GetOrAssignUserCustomerID returns the billing customer ID associated with the user. If no billing
// customer ID exists for the user, a new one is created and saved on the user's DB record.
func GetOrAssignUserCustomerID(ctx context.Context, userID int32) (_ string, err error) {
	// Wrap this operation in a transaction so we never have stored 2 auto-created billing customer
	// IDs for the same user.
	tx, err := dbconn.Global.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			rollErr := tx.Rollback()
			if rollErr != nil {
				err = multierror.Append(err, rollErr)
			}
			return
		}
		err = tx.Commit()
	}()

	custID, err := dbBilling{}.getUserBillingCustomerID(ctx, tx, userID)
	if err != nil {
		return "", err
	}
	if custID == nil {
		// There is no billing customer ID for this user yet, so we must make one. This is not racy
		// w.r.t. the DB because we are still in a DB transaction. It is still possible for a race
		// condition to result in 2 billing customers being created, but only one of them would ever
		// be stored in our DB.
		newCustID, err := createCustomerID(ctx, userID)
		if err != nil {
			return "", errors.WithMessage(err, fmt.Sprintf("auto-creating customer ID for user ID %d", userID))
		}

		// If we fail after here, then try to clean up the customer ID.
		defer func() {
			if err != nil {
				ctx, cancel := context.WithTimeout(ctx, 3*time.Second) // don't wait too long
				defer cancel()
				if err := deleteCustomerID(ctx, newCustID); err != nil {
					log15.Error("During cleanup of failed auto-creation of billing customer ID for user, failed to delete billing customer ID.", "userID", userID, "newCustomerID", newCustID, "err", err)
				}
			}
		}()

		if err := (dbBilling{}).setUserBillingCustomerID(ctx, tx, userID, &newCustID); err != nil {
			return "", err
		}
		custID = &newCustID
	}
	return *custID, nil
}

var (
	dummyCustomerMu sync.Mutex
	dummyCustomerID string
)

// GetDummyCustomerID returns the customer ID for a dummy customer that must be used only for
// pricing out invoices not associated with any particular customer. There is one dummy customer in
// the billing system that is used for all such operations (because the billing system requires
// providing a customer ID but we don't want to use any actual customer's ID).
//
// The first time this func is called, it looks up the ID of the existing dummy customer in the
// billing system and returns that if one exists (to avoid creating many dummy customer records). If
// the dummy customer doesn't exist yet, it is automatically created.
func GetDummyCustomerID(ctx context.Context) (string, error) {
	dummyCustomerMu.Lock()
	defer dummyCustomerMu.Unlock()
	if dummyCustomerID == "" {
		// Look up dummy customer.
		const dummyCustomerEmail = "dummy-customer@example.com"
		listParams := &stripe.CustomerListParams{
			ListParams: stripe.ListParams{Context: ctx},
		}
		listParams.Filters.AddFilter("email", "", dummyCustomerEmail)
		listParams.Limit = stripe.Int64(1)
		customers := customer.List(listParams)
		if err := customers.Err(); err != nil {
			return "", err
		}
		if customers.Next() {
			dummyCustomerID = customers.Customer().ID
		} else {
			// No dummy customer exists yet, so create it. Future calls to GetDummyCustomerID will reuse the dummy customer.
			params := &stripe.CustomerParams{
				Params:      stripe.Params{Context: ctx},
				Email:       stripe.String(dummyCustomerEmail),
				Description: stripe.String("DUMMY (only used for generating quotes for unauthenticated viewers)"),
			}
			cust, err := customer.New(params)
			if err != nil {
				return "", err
			}
			dummyCustomerID = cust.ID
		}
	}
	return dummyCustomerID, nil
}

var mockCreateCustomerID func(userID int32) (string, error)

// createCustomerID creates a customer record on the billing system and returns the customer ID of
// the new record.
func createCustomerID(ctx context.Context, userID int32) (string, error) {
	if mockCreateCustomerID != nil {
		return mockCreateCustomerID(userID)
	}

	user, err := graphqlbackend.UserByIDInt32(ctx, userID)
	if err != nil {
		return "", err
	}
	custParams := &stripe.CustomerParams{
		Params:      stripe.Params{Context: ctx},
		Description: stripe.String(fmt.Sprintf("%s (%d)", user.Username(), user.DatabaseID())),
	}

	// Use the user's first verified email (if any).
	emails, err := user.Emails(ctx)
	if err != nil {
		return "", err
	}
	for _, email := range emails {
		if email.Verified() {
			custParams.Email = stripe.String(email.Email())
			break
		}
	}

	// Create the billing customer.
	cust, err := customer.New(custParams)
	if err != nil {
		return "", err
	}
	return cust.ID, nil
}

// deleteCustomerID deletes the customer record on the billing system.
func deleteCustomerID(ctx context.Context, customerID string) error {
	// For simplicity of tests, just noop if the mockCreateCustomerID is set.
	if mockCreateCustomerID != nil {
		return nil
	}

	_, err := customer.Del(customerID, &stripe.CustomerParams{Params: stripe.Params{Context: ctx}})
	return err
}
