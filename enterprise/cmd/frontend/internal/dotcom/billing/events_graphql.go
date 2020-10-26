package billing

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/stripe/stripe-go"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
)

// productSubscriptionEvent implements the GraphQL type ProductSubscriptionEvent.
type productSubscriptionEvent struct {
	v *stripe.Event
}

// ToProductSubscriptionEvent returns a resolver for the GraphQL type ProductSubscriptionEvent from
// the given billing event.
//
// The okToShowUser return value reports whether the event should be shown to the user. It is false
// for internal events (e.g., an invoice being marked uncollectible).
func ToProductSubscriptionEvent(event *stripe.Event) (gqlEvent graphqlbackend.ProductSubscriptionEvent, okToShowUser bool) {
	_, _, okToShowUser = getProductSubscriptionEventInfo(event)
	return &productSubscriptionEvent{v: event}, okToShowUser
}

// getProductSubscriptionEventInfo returns a nice title and description for the event. See
// ToProductSubscriptionEvent for information about the okToShowUser return value.
func getProductSubscriptionEventInfo(v *stripe.Event) (title, description string, okToShowUser bool) {
	switch v.Type {
	case "charge.succeeded":
		title = "Charge succeeded"
		okToShowUser = true

	case "invoice.created":
		title = "Invoice created"
		okToShowUser = true
	case "invoice.payment_succeeded":
		title = "Invoice payment succeeded"
		description = fmt.Sprintf("An invoice payment of %s succeeded.", usdCentsToString(v.GetObjectValue("amount_paid")))
		okToShowUser = true
	case "invoice.payment_failed":
		title = "Invoice payment failed"
		description = fmt.Sprintf("An invoice payment of %s failed.", usdCentsToString(v.GetObjectValue("amount_paid")))
		okToShowUser = true
	case "invoice.sent":
		title = "Invoice email sent"
		okToShowUser = true
	case "invoice.updated":
		title = "Invoice updated"
		okToShowUser = true

	default:
		title = v.Type
	}
	return title, description, okToShowUser
}

func usdCentsToString(s string) string {
	// TODO(sqs): use a real currency lib
	usdCents, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return "unknown amount"
	}
	return fmt.Sprintf("$%.2f", usdCents/100)
}

func (r *productSubscriptionEvent) ID() string { return r.v.ID }

func (r *productSubscriptionEvent) Date() string {
	return time.Unix(r.v.Created, 0).Format(time.RFC3339)
}

func (r *productSubscriptionEvent) Title() string {
	title, _, _ := getProductSubscriptionEventInfo(r.v)
	return title
}

func (r *productSubscriptionEvent) Description() *string {
	_, description, _ := getProductSubscriptionEventInfo(r.v)
	if description == "" {
		return nil
	}
	return &description
}

func (r *productSubscriptionEvent) URL() *string {
	var u string
	if strings.HasPrefix(r.v.Type, "invoice.") {
		u = r.v.GetObjectValue("hosted_invoice_url")
	}
	if u == "" {
		return nil
	}
	return &u
}
