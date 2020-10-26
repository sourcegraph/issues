import * as H from 'history'
import * as React from 'react'
import { Redirect } from 'react-router'
import { userURL } from '..'
import { withAuthenticatedUser } from '../../auth/withAuthenticatedUser'
import { AuthenticatedUser } from '../../auth'

/**
 * Redirects from /user/$PATH to /user/$USERNAME/$PATH, where $USERNAME is the currently
 * authenticated user's username.
 */
export const RedirectToUserPage = withAuthenticatedUser<{ authenticatedUser: AuthenticatedUser; location: H.Location }>(
    ({ authenticatedUser, location }) => {
        const path = location.pathname.replace(/^\/user\//, '') // trim leading '/user/'
        return (
            <Redirect
                to={{
                    pathname: `${userURL(authenticatedUser.username)}/${path}`,
                    search: location.search,
                    hash: location.hash,
                }}
            />
        )
    }
)
