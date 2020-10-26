package productsubscription

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/keegancsmith/sqlf"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
)

// dbLicense describes an product license row in the product_licenses DB table.
type dbLicense struct {
	ID                    string // UUID
	ProductSubscriptionID string // UUID
	LicenseKey            string
	CreatedAt             time.Time
}

// errLicenseNotFound occurs when a database operation expects a specific Sourcegraph
// license to exist but it does not exist.
var errLicenseNotFound = errors.New("product license not found")

// dbLicenses exposes product licenses in the product_licenses DB table.
type dbLicenses struct{}

// Create creates a new product license entry given a license key.
func (dbLicenses) Create(ctx context.Context, subscriptionID, licenseKey string) (id string, err error) {
	if mocks.licenses.Create != nil {
		return mocks.licenses.Create(subscriptionID, licenseKey)
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	if err := dbconn.Global.QueryRowContext(ctx, `
INSERT INTO product_licenses(id, product_subscription_id, license_key) VALUES($1, $2, $3) RETURNING id
`,
		uuid, subscriptionID, licenseKey,
	).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

// GetByID retrieves the product license (if any) given its ID.
//
// 🚨 SECURITY: The caller must ensure that the actor is permitted to view this product license.
func (s dbLicenses) GetByID(ctx context.Context, id string) (*dbLicense, error) {
	if mocks.licenses.GetByID != nil {
		return mocks.licenses.GetByID(id)
	}
	results, err := s.list(ctx, []*sqlf.Query{sqlf.Sprintf("id=%s", id)}, nil)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errLicenseNotFound
	}
	return results[0], nil
}

// GetByID retrieves the product license (if any) given its license key.
func (s dbLicenses) GetByLicenseKey(ctx context.Context, licenseKey string) (*dbLicense, error) {
	if mocks.licenses.GetByLicenseKey != nil {
		return mocks.licenses.GetByLicenseKey(licenseKey)
	}
	results, err := s.list(ctx, []*sqlf.Query{sqlf.Sprintf("license_key=%s", licenseKey)}, nil)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errLicenseNotFound
	}
	return results[0], nil
}

// dbLicensesListOptions contains options for listing product licenses.
type dbLicensesListOptions struct {
	LicenseKeySubstring   string
	ProductSubscriptionID string // only list product licenses for this subscription (by UUID)
	*db.LimitOffset
}

func (o dbLicensesListOptions) sqlConditions() []*sqlf.Query {
	conds := []*sqlf.Query{sqlf.Sprintf("TRUE")}
	if o.LicenseKeySubstring != "" {
		conds = append(conds, sqlf.Sprintf("license_key LIKE %s", "%"+o.LicenseKeySubstring+"%"))
	}
	if o.ProductSubscriptionID != "" {
		conds = append(conds, sqlf.Sprintf("product_subscription_id=%s", o.ProductSubscriptionID))
	}
	return conds
}

// List lists all product licenses that satisfy the options.
func (s dbLicenses) List(ctx context.Context, opt dbLicensesListOptions) ([]*dbLicense, error) {
	if mocks.licenses.List != nil {
		return mocks.licenses.List(ctx, opt)
	}

	return s.list(ctx, opt.sqlConditions(), opt.LimitOffset)
}

func (dbLicenses) list(ctx context.Context, conds []*sqlf.Query, limitOffset *db.LimitOffset) ([]*dbLicense, error) {
	q := sqlf.Sprintf(`
SELECT id, product_subscription_id, license_key, created_at FROM product_licenses
WHERE (%s)
ORDER BY created_at DESC
%s`,
		sqlf.Join(conds, ") AND ("),
		limitOffset.SQL(),
	)

	rows, err := dbconn.Global.QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*dbLicense
	for rows.Next() {
		var v dbLicense
		if err := rows.Scan(&v.ID, &v.ProductSubscriptionID, &v.LicenseKey, &v.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, &v)
	}
	return results, nil
}

// Count counts all product licenses that satisfy the options (ignoring limit and offset).
func (dbLicenses) Count(ctx context.Context, opt dbLicensesListOptions) (int, error) {
	q := sqlf.Sprintf("SELECT COUNT(*) FROM product_licenses WHERE (%s)", sqlf.Join(opt.sqlConditions(), ") AND ("))
	var count int
	if err := dbconn.Global.QueryRowContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

type mockLicenses struct {
	Create          func(subscriptionID, licenseKey string) (id string, err error)
	GetByID         func(id string) (*dbLicense, error)
	GetByLicenseKey func(licenseKey string) (*dbLicense, error)
	List            func(ctx context.Context, opt dbLicensesListOptions) ([]*dbLicense, error)
}
