import * as H from 'history'
import * as React from 'react'
import { Redirect } from 'react-router'
import { userURL } from '..'
import * as GQL from '../../../../shared/src/graphql/schema'
import { withAuthenticatedUser } from '../../auth/withAuthenticatedUser'

/**
 * Redirects from /settings to /user/$USERNAME/settings, where $USERNAME is the currently authenticated user's
 * username.
 */
export const RedirectToUserSettings = withAuthenticatedUser(
    ({ authenticatedUser, location }: { authenticatedUser: GQL.User; location: H.Location }) => (
        <Redirect
            to={{
                pathname: `${userURL(authenticatedUser.username)}/settings`,
                search: location.search,
                hash: location.hash,
            }}
        />
    )
)
