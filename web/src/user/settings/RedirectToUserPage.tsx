import * as H from 'history'
import * as React from 'react'
import { Redirect } from 'react-router'
import { userURL } from '..'
import * as GQL from '../../../../shared/src/graphql/schema'
import { withAuthenticatedUser } from '../../auth/withAuthenticatedUser'

/**
 * Redirects from /user/$PATH to /user/$USERNAME/$PATH, where $USERNAME is the currently
 * authenticated user's username.
 */
export const RedirectToUserPage = withAuthenticatedUser(
    ({ authenticatedUser, location }: { authenticatedUser: GQL.User; location: H.Location }) => {
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
