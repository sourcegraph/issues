package productsubscription

import (
	"context"
	"sync"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/license"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/licensing"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

// productLicense implements the GraphQL type ProductLicense.
type productLicense struct {
	db dbutil.DB
	v  *dbLicense
}

// ProductLicenseByID looks up and returns the ProductLicense with the given GraphQL ID. If no such
// ProductLicense exists, it returns a non-nil error.
func (p ProductSubscriptionLicensingResolver) ProductLicenseByID(ctx context.Context, id graphql.ID) (graphqlbackend.ProductLicense, error) {
	return productLicenseByID(ctx, p.DB, id)
}

// productLicenseByID looks up and returns the ProductLicense with the given GraphQL ID. If no such
// ProductLicense exists, it returns a non-nil error.
func productLicenseByID(ctx context.Context, db dbutil.DB, id graphql.ID) (*productLicense, error) {
	idInt32, err := unmarshalProductLicenseID(id)
	if err != nil {
		return nil, err
	}
	return productLicenseByDBID(ctx, db, idInt32)
}

// productLicenseByDBID looks up and returns the ProductLicense with the given database ID. If no
// such ProductLicense exists, it returns a non-nil error.
func productLicenseByDBID(ctx context.Context, db dbutil.DB, id string) (*productLicense, error) {
	v, err := dbLicenses{db: db}.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 🚨 SECURITY: Only site admins and the license's subscription's account's user may view a
	// product license.
	sub, err := productSubscriptionByDBID(ctx, db, v.ProductSubscriptionID)
	if err != nil {
		return nil, err
	}
	if err := backend.CheckSiteAdminOrSameUser(ctx, db, sub.v.UserID); err != nil {
		return nil, err
	}

	return &productLicense{db: db, v: v}, nil
}

func (r *productLicense) ID() graphql.ID {
	return marshalProductLicenseID(r.v.ID)
}

const ProductLicenseIDKind = "ProductLicense"

func marshalProductLicenseID(id string) graphql.ID {
	return relay.MarshalID(ProductLicenseIDKind, id)
}

func unmarshalProductLicenseID(id graphql.ID) (productLicenseID string, err error) {
	err = relay.UnmarshalSpec(id, &productLicenseID)
	return
}

func (r *productLicense) Subscription(ctx context.Context) (graphqlbackend.ProductSubscription, error) {
	return productSubscriptionByDBID(ctx, r.db, r.v.ProductSubscriptionID)
}

func (r *productLicense) Info() (*graphqlbackend.ProductLicenseInfo, error) {
	// Call this instead of licensing.ParseProductLicenseKey so that license info can be read from
	// license keys generated using the test license generation private key.
	info, _, err := licensing.ParseProductLicenseKeyWithBuiltinOrGenerationKey(r.v.LicenseKey)
	if err != nil {
		return nil, err
	}
	return &graphqlbackend.ProductLicenseInfo{
		TagsValue:      info.Tags,
		UserCountValue: info.UserCount,
		ExpiresAtValue: info.ExpiresAt,
	}, nil
}

func (r *productLicense) LicenseKey() string { return r.v.LicenseKey }

func (r *productLicense) CreatedAt() graphqlbackend.DateTime {
	return graphqlbackend.DateTime{Time: r.v.CreatedAt}
}

func generateProductLicenseForSubscription(ctx context.Context, db dbutil.DB, subscriptionID string, input *graphqlbackend.ProductLicenseInput) (id string, err error) {
	licenseKey, err := licensing.GenerateProductLicenseKey(license.Info{
		Tags:      license.SanitizeTagsList(input.Tags),
		UserCount: uint(input.UserCount),
		ExpiresAt: time.Unix(int64(input.ExpiresAt), 0),
	})
	if err != nil {
		return "", err
	}
	return dbLicenses{db: db}.Create(ctx, subscriptionID, licenseKey)
}

func (r ProductSubscriptionLicensingResolver) GenerateProductLicenseForSubscription(ctx context.Context, args *graphqlbackend.GenerateProductLicenseForSubscriptionArgs) (graphqlbackend.ProductLicense, error) {
	// 🚨 SECURITY: Only site admins may generate product licenses.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.DB); err != nil {
		return nil, err
	}
	sub, err := productSubscriptionByID(ctx, r.DB, args.ProductSubscriptionID)
	if err != nil {
		return nil, err
	}
	id, err := generateProductLicenseForSubscription(ctx, r.DB, sub.v.ID, args.License)
	if err != nil {
		return nil, err
	}
	return productLicenseByDBID(ctx, r.DB, id)
}

func (r ProductSubscriptionLicensingResolver) ProductLicenses(ctx context.Context, args *graphqlbackend.ProductLicensesArgs) (graphqlbackend.ProductLicenseConnection, error) {
	// 🚨 SECURITY: Only site admins may list product licenses.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.DB); err != nil {
		return nil, err
	}

	var sub *productSubscription
	if args.ProductSubscriptionID != nil {
		var err error
		sub, err = productSubscriptionByID(ctx, r.DB, *args.ProductSubscriptionID)
		if err != nil {
			return nil, err
		}
	}

	var opt dbLicensesListOptions
	if sub != nil {
		opt.ProductSubscriptionID = sub.v.ID
	}
	if args.LicenseKeySubstring != nil {
		opt.LicenseKeySubstring = *args.LicenseKeySubstring
	}
	args.ConnectionArgs.Set(&opt.LimitOffset)
	return &productLicenseConnection{db: r.DB, opt: opt}, nil
}

// productLicenseConnection implements the GraphQL type ProductLicenseConnection.
//
// 🚨 SECURITY: When instantiating a productLicenseConnection value, the caller MUST
// check permissions.
type productLicenseConnection struct {
	opt dbLicensesListOptions
	db  dbutil.DB

	// cache results because they are used by multiple fields
	once    sync.Once
	results []*dbLicense
	err     error
}

func (r *productLicenseConnection) compute(ctx context.Context) ([]*dbLicense, error) {
	r.once.Do(func() {
		opt2 := r.opt
		if opt2.LimitOffset != nil {
			tmp := *opt2.LimitOffset
			opt2.LimitOffset = &tmp
			opt2.Limit++ // so we can detect if there is a next page
		}

		r.results, r.err = dbLicenses{db: r.db}.List(ctx, opt2)
	})
	return r.results, r.err
}

func (r *productLicenseConnection) Nodes(ctx context.Context) ([]graphqlbackend.ProductLicense, error) {
	results, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}

	var l []graphqlbackend.ProductLicense
	for _, result := range results {
		l = append(l, &productLicense{db: r.db, v: result})
	}
	return l, nil
}

func (r *productLicenseConnection) TotalCount(ctx context.Context) (int32, error) {
	count, err := dbLicenses{db: r.db}.Count(ctx, r.opt)
	return int32(count), err
}

func (r *productLicenseConnection) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	results, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}
	return graphqlutil.HasNextPage(r.opt.LimitOffset != nil && len(results) > r.opt.Limit), nil
}
