package productsubscription

import (
	"context"
	"testing"
	"time"

	"github.com/efritz/glock"
	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/license"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/licensing"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/slack"
	"github.com/sourcegraph/sourcegraph/schema"
)

type fakeSlackClient struct {
	payloads []*slack.Payload
}

func (c *fakeSlackClient) Post(ctx context.Context, payload *slack.Payload) error {
	c.payloads = append(c.payloads, payload)
	return nil
}

func TestCheckForUpcomingLicenseExpirations(t *testing.T) {
	clock := glock.NewMockClock()

	cfg := conf.Get()
	conf.Mock(&conf.Unified{
		SiteConfiguration: schema.SiteConfiguration{
			Dotcom: &schema.Dotcom{
				SlackLicenseExpirationWebhook: "https://slack.com/webhook",
			},
		},
	})
	mocks.subscriptions.List = func(ctx context.Context, opt dbSubscriptionsListOptions) ([]*dbSubscription, error) {
		return []*dbSubscription{
			{ID: "e9450fb2-87c7-47ae-a713-a376c4618faa"},
			{ID: "26136564-b319-4be4-98ff-7b8710abf4af"},
		}, nil
	}
	mocks.licenses.List = func(ctx context.Context, opt dbLicensesListOptions) ([]*dbLicense, error) {
		return []*dbLicense{{LicenseKey: opt.ProductSubscriptionID}}, nil
	}
	licensing.MockParseProductLicenseKeyWithBuiltinOrGenerationKey = func(licenseKey string) (*licensing.Info, string, error) {
		infos := map[string]*licensing.Info{
			"e9450fb2-87c7-47ae-a713-a376c4618faa": {
				Info: license.Info{
					ExpiresAt: clock.Now().Add((24 + 1) * time.Hour), // day away
				},
			},
			"26136564-b319-4be4-98ff-7b8710abf4af": {
				Info: license.Info{
					ExpiresAt: clock.Now().Add((7*24 + 1) * time.Hour), // week away
				},
			},
		}
		return infos[licenseKey], "", nil
	}
	db.Mocks.Users.GetByID = func(ctx context.Context, id int32) (*types.User, error) {
		return &types.User{Username: "alice"}, nil
	}
	t.Cleanup(func() {
		conf.Mock(cfg)
		mocks.subscriptions = mockSubscriptions{}
		mocks.licenses = mockLicenses{}
		licensing.MockParseProductLicenseKeyWithBuiltinOrGenerationKey = nil
		db.Mocks.Users = db.MockUsers{}
	})

	client := &fakeSlackClient{}
	checkForUpcomingLicenseExpirations(clock, client)

	wantPayloads := []*slack.Payload{
		{Text: "The license for user `alice` <https://sourcegraph.com/site-admin/dotcom/product/subscriptions/e9450fb2-87c7-47ae-a713-a376c4618faa|will expire *in the next 24 hours*> :rotating_light:"},
		{Text: "The license for user `alice` <https://sourcegraph.com/site-admin/dotcom/product/subscriptions/26136564-b319-4be4-98ff-7b8710abf4af|will expire *in 7 days*>"},
	}
	if diff := cmp.Diff(wantPayloads, client.payloads); diff != "" {
		t.Fatalf("Payloads mismatch (-want +got):\n%s", diff)
	}
}
