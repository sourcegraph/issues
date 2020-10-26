import React from 'react'
import { Redirect, RouteComponentProps } from 'react-router'
import { UserSubscriptionsNewProductSubscriptionPage } from './UserSubscriptionsNewProductSubscriptionPage'
import { ThemeProps } from '../../../../../shared/src/theme'
import { AuthenticatedUser } from '../../../auth'

interface Props extends RouteComponentProps<{}>, ThemeProps {
    authenticatedUser: AuthenticatedUser | null
}

/**
 * Displays or redirects to the new product subscription page.
 *
 * For authenticated viewers, it redirects to the page under their user account.
 *
 * For unauthenticated viewers, it displays a page that lets them price out a subscription (but requires them to
 * sign in to actually buy it). This friendlier behavior for unauthed viewers (compared to dumping them on a
 * sign-in page) is the reason why this component exists.
 */
export const NewProductSubscriptionPageOrRedirectUser: React.FunctionComponent<Props> = props =>
    props.authenticatedUser ? (
        <Redirect to={`${props.authenticatedUser.settingsURL!}/subscriptions/new`} />
    ) : (
        <div className="container w-75 mt-4">
            <UserSubscriptionsNewProductSubscriptionPage {...props} user={null} />
        </div>
    )
