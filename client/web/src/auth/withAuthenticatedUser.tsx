import React from 'react'
import { Redirect } from 'react-router'
import { AuthenticatedUser } from '../auth'

/**
 * Wraps a React component and requires an authenticated user. If the viewer is not authenticated, it redirects to
 * the sign-in flow.
 */
export const withAuthenticatedUser = <P extends object & { authenticatedUser: AuthenticatedUser }>(
    Component: React.ComponentType<P>
): React.ComponentType<
    Pick<P, Exclude<keyof P, 'authenticatedUser'>> & { authenticatedUser: AuthenticatedUser | null }
> => ({ authenticatedUser, ...props }) => {
    // If not logged in, redirect to sign in.
    if (!authenticatedUser) {
        const newUrl = new URL(window.location.href)
        newUrl.pathname = '/sign-in'
        // Return to the current page after sign up/in.
        newUrl.searchParams.set('returnTo', window.location.href)
        return <Redirect to={newUrl.pathname + newUrl.search} />
    }
    return <Component {...({ ...props, authenticatedUser } as P)} />
}
