import * as H from 'history'
import * as React from 'react'
import { Redirect } from 'react-router'
import { userURL } from '..'
import { withAuthenticatedUser } from '../../auth/withAuthenticatedUser'
import { AuthenticatedUser } from '../../auth'

/**
 * Redirects from /settings to /user/$USERNAME/settings, where $USERNAME is the currently authenticated user's
 * username.
 */
export const RedirectToUserSettings = withAuthenticatedUser<{
    authenticatedUser: AuthenticatedUser
    location: H.Location
}>(({ authenticatedUser, location }) => (
    <Redirect
        to={{
            pathname: `${userURL(authenticatedUser.username)}/settings`,
            search: location.search,
            hash: location.hash,
        }}
    />
))
