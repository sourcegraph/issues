package enforcement

import (
	"context"
	"fmt"

	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/licensing"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

// NewPreCreateUserHook returns a PreCreateUserHook closure with
// the given UsersStore.
func NewPreCreateUserHook(s licensing.UsersStore) func(context.Context) error {
	return func(ctx context.Context) error {
		info, err := licensing.GetConfiguredProductLicenseInfo()
		if err != nil {
			return err
		}
		var licensedUserCount int32
		if info != nil {
			licensedUserCount = int32(info.UserCount)
		} else {
			licensedUserCount = licensing.NoLicenseMaximumAllowedUserCount
		}

		// Block creation of a new user beyond the licensed user count (unless true-up is allowed).
		userCount, err := s.Count(ctx)
		if err != nil {
			return err
		}
		// Be conservative and treat 0 as unlimited. We don't plan to intentionally generate
		// licenses with UserCount == 0, but that might result from a bug in license decoding, and
		// we don't want that to immediately disable Sourcegraph instances.
		if licensedUserCount > 0 && int32(userCount) >= licensedUserCount {
			if info != nil && info.HasTag(licensing.TrueUpUserCountTag) {
				log15.Info("Licensed user count exceeded, but license supports true-up and will not block creation of new user. The new user will be retroactively charged for in the next billing period. Contact sales@sourcegraph.com for help.", "activeUserCount", userCount, "licensedUserCount", licensedUserCount)
			} else {
				message := "Unable to create user account: "
				if info == nil {
					message += fmt.Sprintf("a Sourcegraph subscription is required to exceed %d users (this instance now has %d users). Contact Sourcegraph to learn more at https://about.sourcegraph.com/contact/sales.", licensing.NoLicenseMaximumAllowedUserCount, userCount)
				} else {
					message += "the Sourcegraph subscription's maximum user count has been reached. A site admin must upgrade the Sourcegraph subscription to allow for more users. Contact Sourcegraph at https://about.sourcegraph.com/contact/sales."
				}
				return errcode.NewPresentationError(message)
			}
		}

		return nil
	}
}
