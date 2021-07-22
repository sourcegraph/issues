import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React, { useMemo } from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'

import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { ActivationProps } from '@sourcegraph/shared/src/components/activation/Activation'
import { ExtensionsControllerProps } from '@sourcegraph/shared/src/extensions/controller'
import { gql, useQuery } from '@sourcegraph/shared/src/graphql/graphql'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { ThemeProps } from '@sourcegraph/shared/src/theme'

import { AuthenticatedUser } from '../../auth'
import { BatchChangesProps } from '../../batches/batches'
import { BreadcrumbsProps, BreadcrumbSetters } from '../../components/Breadcrumbs'
import { ErrorBoundary } from '../../components/ErrorBoundary'
import { HeroPage } from '../../components/HeroPage'
import { Page } from '../../components/Page'
import { UserAreaUserFields, UserAreaUserProfileResult, UserAreaUserProfileVariables } from '../../graphql-operations'
import { NamespaceProps } from '../../namespaces'
import { PatternTypeProps, OnboardingTourProps } from '../../search'
import { UserExternalServicesOrRepositoriesUpdateProps } from '../../util'
import { RouteDescriptor } from '../../util/contributions'
import { EditUserProfilePageGQLFragment } from '../settings/profile/UserSettingsProfilePage'
import { UserSettingsAreaRoute } from '../settings/UserSettingsArea'
import { UserSettingsSidebarItems } from '../settings/UserSettingsSidebar'

import { UserAreaHeader, UserAreaHeaderNavItem } from './UserAreaHeader'

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
        tags @include(if: $siteAdmin)
        ...EditUserProfilePage
    }
    ${EditUserProfilePageGQLFragment}
`

export const USER_AREA_USER_PROFILE = gql`
    query UserAreaUserProfile($username: String!, $siteAdmin: Boolean!) {
        user(username: $username) {
            ...UserAreaUserFields
        }
    }
    ${UserAreaGQLFragment}
`

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
        BatchChangesProps,
        Omit<PatternTypeProps, 'setPatternType'>,
        UserExternalServicesOrRepositoriesUpdateProps {
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
        BatchChangesProps,
        Omit<PatternTypeProps, 'setPatternType'>,
        UserExternalServicesOrRepositoriesUpdateProps {
    /** The user area main URL. */
    url: string

    /**
     * The user who is the subject of the page.
     */
    user: UserAreaUserFields

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
    const { data, error, loading, previousData } = useQuery<UserAreaUserProfileResult, UserAreaUserProfileVariables>(
        USER_AREA_USER_PROFILE,
        {
            variables: { username, siteAdmin: Boolean(props.authenticatedUser?.siteAdmin) },
        }
    )

    const childBreadcrumbSetters = useBreadcrumb(
        useMemo(
            () =>
                data?.user
                    ? {
                          key: 'UserArea',
                          link: { to: data.user.url, label: data.user.username },
                      }
                    : null,
            [data]
        )
    )

    // Accept stale data if recently updated, avoids unmounting components due to a brief lack of data
    const user = data?.user ?? previousData?.user

    if (loading && !user) {
        return null
    }

    if (error) {
        throw new Error(error.message)
    }

    if (!user) {
        throw new Error(`User not found: ${JSON.stringify(username)}`)
    }

    const context: UserAreaRouteContext = {
        ...props,
        url,
        user,
        namespace: user,
        ...childBreadcrumbSetters,
    }

    return (
        <Page className="user-area">
            <UserAreaHeader {...props} {...context} navItems={props.userAreaHeaderNavItems} />
            <div className="container mt-3">
                <ErrorBoundary location={props.location}>
                    <React.Suspense fallback={<LoadingSpinner className="icon-inline m-2" />}>
                        <Switch>
                            {userAreaRoutes.map(
                                ({ path, exact, render, condition = () => true }) =>
                                    condition(context) && (
                                        <Route
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
        </Page>
    )
}
