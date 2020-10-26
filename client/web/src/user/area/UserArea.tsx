import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React, { useMemo, useState } from 'react'
import { matchPath, Route, RouteComponentProps, Switch } from 'react-router'
import { Observable } from 'rxjs'
import { map, tap } from 'rxjs/operators'
import { ActivationProps } from '../../../../shared/src/components/activation/Activation'
import { ExtensionsControllerProps } from '../../../../shared/src/extensions/controller'
import { gql, dataOrThrowErrors } from '../../../../shared/src/graphql/graphql'
import { PlatformContextProps } from '../../../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../../../shared/src/settings/settings'
import { ErrorBoundary } from '../../components/ErrorBoundary'
import { HeroPage } from '../../components/HeroPage'
import { NamespaceProps } from '../../namespaces'
import { ThemeProps } from '../../../../shared/src/theme'
import { RouteDescriptor } from '../../util/contributions'
import { UserSettingsAreaRoute } from '../settings/UserSettingsArea'
import { UserSettingsSidebarItems } from '../settings/UserSettingsSidebar'
import { UserAreaHeader, UserAreaHeaderNavItem } from './UserAreaHeader'
import { PatternTypeProps, OnboardingTourProps } from '../../search'
import { TelemetryProps } from '../../../../shared/src/telemetry/telemetryService'
import { AuthenticatedUser } from '../../auth'
import { UserAreaUserFields, UserAreaResult, UserAreaVariables } from '../../graphql-operations'
import { BreadcrumbsProps, BreadcrumbSetters } from '../../components/Breadcrumbs'
import { useObservable } from '../../../../shared/src/util/useObservable'
import { requestGraphQL } from '../../backend/graphql'
import { EditUserProfilePageGQLFragment } from '../settings/profile/UserSettingsProfilePage'

/** GraphQL fragment for the User fields needed by UserArea. */
export const UserAreaGQLFragment = gql`
    fragment UserAreaUserFields on User {
        __typename
        id
        username
        displayName
        url
        settingsURL
        avatarURL
        viewerCanAdminister
        siteAdmin @include(if: $siteAdmin)
        builtinAuth
        createdAt
        emails @include(if: $siteAdmin) {
            email
            verified
        }
        organizations {
            nodes {
                id
                displayName
                name
            }
        }
        permissionsInfo @include(if: $siteAdmin) {
            syncedAt
            updatedAt
        }
        tags @include(if: $siteAdmin)
        ...EditUserProfilePage
    }
    ${EditUserProfilePageGQLFragment}
`

const fetchUser = (args: UserAreaVariables): Observable<UserAreaUserFields> =>
    requestGraphQL<UserAreaResult, UserAreaVariables>(
        gql`
            query UserArea($username: String!, $siteAdmin: Boolean!) {
                user(username: $username) {
                    ...UserAreaUserFields
                    ...EditUserProfilePage
                }
            }
            ${UserAreaGQLFragment}
        `,
        args
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.user) {
                throw new Error(`User not found: ${JSON.stringify(args.username)}`)
            }
            return data.user
        })
    )

export interface UserAreaRoute extends RouteDescriptor<UserAreaRouteContext> {}

interface UserAreaProps
    extends RouteComponentProps<{ username: string }>,
        ExtensionsControllerProps,
        PlatformContextProps,
        SettingsCascadeProps,
        ThemeProps,
        TelemetryProps,
        ActivationProps,
        OnboardingTourProps,
        BreadcrumbsProps,
        BreadcrumbSetters,
        Omit<PatternTypeProps, 'setPatternType'> {
    userAreaRoutes: readonly UserAreaRoute[]
    userAreaHeaderNavItems: readonly UserAreaHeaderNavItem[]
    userSettingsSideBarItems: UserSettingsSidebarItems
    userSettingsAreaRoutes: readonly UserSettingsAreaRoute[]

    /**
     * The currently authenticated user, NOT the user whose username is specified in the URL's "username" route
     * parameter.
     */
    authenticatedUser: AuthenticatedUser | null

    isSourcegraphDotCom: boolean
}

/**
 * Properties passed to all page components in the user area.
 */
export interface UserAreaRouteContext
    extends ExtensionsControllerProps,
        PlatformContextProps,
        SettingsCascadeProps,
        ThemeProps,
        TelemetryProps,
        ActivationProps,
        NamespaceProps,
        OnboardingTourProps,
        BreadcrumbsProps,
        BreadcrumbSetters,
        Omit<PatternTypeProps, 'setPatternType'> {
    /** The user area main URL. */
    url: string

    /**
     * The user who is the subject of the page.
     */
    user: UserAreaUserFields

    /** Called when the user is updated, with the full new user data. */
    onUserUpdate: (newUser: UserAreaUserFields) => void

    /**
     * The currently authenticated user, NOT (necessarily) the user who is the subject of the page.
     *
     * For example, if Alice is viewing a user area page about Bob, then the authenticatedUser is Alice and the
     * user is Bob.
     */
    authenticatedUser: AuthenticatedUser | null
    userSettingsSideBarItems: UserSettingsSidebarItems
    userSettingsAreaRoutes: readonly UserSettingsAreaRoute[]

    isSourcegraphDotCom: boolean
}

/**
 * A user's public profile area.
 */
export const UserArea: React.FunctionComponent<UserAreaProps> = ({
    useBreadcrumb,
    userAreaRoutes,
    match: {
        url,
        params: { username },
    },
    ...props
}) => {
    // When onUserUpdate is called (e.g., when updating a user's profile), we want to immediately
    // use the newly updated user data instead of re-querying it. Therefore, we store the user in
    // state. The initial GraphQL query populates it, and onUserUpdate calls update it.
    const [user, setUser] = useState<UserAreaUserFields>()
    useObservable(
        useMemo(
            () =>
                fetchUser({
                    username,
                    siteAdmin: Boolean(props.authenticatedUser?.siteAdmin),
                }).pipe(tap(setUser)),
            [props.authenticatedUser?.siteAdmin, username]
        )
    )

    const childBreadcrumbSetters = useBreadcrumb(
        useMemo(
            () =>
                user
                    ? {
                          key: 'UserArea',
                          link: { to: user.url, label: user.username },
                      }
                    : null,
            [user]
        )
    )

    if (user === undefined) {
        return null // loading
    }

    const context: UserAreaRouteContext = {
        ...props,
        url,
        user,
        onUserUpdate: setUser,
        namespace: user,
        ...childBreadcrumbSetters,
    }

    const routeMatch = userAreaRoutes.find(({ path, exact }) =>
        matchPath(props.location.pathname, { path: url + path, exact })
    )?.path

    // Hide header and use full-width container for campaigns pages.
    const isCampaigns = routeMatch === '/campaigns'
    const hideHeader = isCampaigns

    return (
        <div className="user-area w-100">
            {!hideHeader && (
                <UserAreaHeader
                    {...props}
                    {...context}
                    navItems={props.userAreaHeaderNavItems}
                    className="border-bottom mt-4"
                />
            )}
            <div className="container mt-3">
                <ErrorBoundary location={props.location}>
                    <React.Suspense fallback={<LoadingSpinner className="icon-inline m-2" />}>
                        <Switch>
                            {userAreaRoutes.map(
                                ({ path, exact, render, condition = () => true }) =>
                                    condition(context) && (
                                        <Route
                                            // eslint-disable-next-line react/jsx-no-bind
                                            render={routeComponentProps =>
                                                render({ ...context, ...routeComponentProps })
                                            }
                                            path={url + path}
                                            key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                                            exact={exact}
                                        />
                                    )
                            )}
                            <Route key="hardcoded-key">
                                <HeroPage
                                    icon={MapSearchIcon}
                                    title="404: Not Found"
                                    subtitle="Sorry, the requested user page was not found."
                                />
                            </Route>
                        </Switch>
                    </React.Suspense>
                </ErrorBoundary>
            </div>
        </div>
    )
}
