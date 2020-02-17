import * as React from 'react'
import { Redirect, RouteComponentProps } from 'react-router'
import { LayoutProps } from './Layout'
import { parseSearchURLQuery } from './search'
import { lazyComponent } from './util/lazyComponent'

const SearchPage = lazyComponent(() => import('./search/input/SearchPage'), 'SearchPage')
const SearchResults = lazyComponent(() => import('./search/results/SearchResults'), 'SearchResults')
const SiteAdminArea = lazyComponent(() => import('./site-admin/SiteAdminArea'), 'SiteAdminArea')
const ExtensionsArea = lazyComponent(() => import('./extensions/ExtensionsArea'), 'ExtensionsArea')

interface LayoutRouteComponentProps<Params extends { [K in keyof Params]?: string }>
    extends RouteComponentProps<Params>,
        Omit<LayoutProps, 'match'> {}

export interface LayoutRouteProps<Params extends { [K in keyof Params]?: string }> {
    path: string
    exact?: boolean
    render: (props: LayoutRouteComponentProps<Params>) => React.ReactNode

    /**
     * A condition function that needs to return true if the route should be rendered
     *
     * @default () => true
     */
    condition?: (props: LayoutRouteComponentProps<Params>) => boolean
}

/**
 * Holds all top-level routes for the app because both the navbar and the main content area need to
 * switch over matched path.
 *
 * See https://reacttraining.com/react-router/web/example/sidebar
 */
export const routes: readonly LayoutRouteProps<any>[] = [
    {
        path: '/',
        render: (props: any) =>
            window.context.sourcegraphDotComMode && !props.user ? (
                <Redirect to="https://about.sourcegraph.com" />
            ) : (
                <Redirect to="/search" />
            ),
        exact: true,
    },
    {
        path: '/search',
        render: props =>
            parseSearchURLQuery(props.location.search) ? (
                <SearchResults {...props} deployType={window.context.deployType} />
            ) : (
                <SearchPage {...props} />
            ),
        exact: true,
    },
    {
        path: '/search/query-builder',
        render: lazyComponent(() => import('./search/queryBuilder/QueryBuilderPage'), 'QueryBuilderPage'),
        exact: true,
    },
    {
        path: '/sign-in',
        render: lazyComponent(() => import('./auth/SignInPage'), 'SignInPage'),
        exact: true,
    },
    {
        path: '/sign-up',
        render: lazyComponent(() => import('./auth/SignUpPage'), 'SignUpPage'),
        exact: true,
    },
    {
        path: '/settings',
        render: lazyComponent(() => import('./user/settings/RedirectToUserSettings'), 'RedirectToUserSettings'),
    },
    {
        path: '/user',
        render: lazyComponent(() => import('./user/settings/RedirectToUserPage'), 'RedirectToUserPage'),
    },
    {
        path: '/organizations',
        render: lazyComponent(() => import('./org/OrgsArea'), 'OrgsArea'),
    },
    {
        path: '/search',
        render: props => <SearchResults {...props} deployType={window.context.deployType} />,
        exact: true,
    },
    {
        path: '/site-admin/init',
        exact: true,
        render: lazyComponent(() => import('./site-admin/SiteInitPage'), 'SiteInitPage'),
    },
    {
        path: '/site-admin',
        render: props => (
            <SiteAdminArea
                {...props}
                routes={props.siteAdminAreaRoutes}
                sideBarGroups={props.siteAdminSideBarGroups}
                overviewComponents={props.siteAdminOverviewComponents}
            />
        ),
    },
    {
        path: '/password-reset',
        render: lazyComponent(() => import('./auth/ResetPasswordPage'), 'ResetPasswordPage'),
        exact: true,
    },
    {
        path: '/explore',
        render: lazyComponent(() => import('./explore/ExploreArea'), 'ExploreArea'),
        exact: true,
    },
    {
        path: '/discussions',
        render: lazyComponent(() => import('./discussions/DiscussionsPage'), 'DiscussionsPage'),
        exact: true,
    },
    {
        path: '/search/scope/:id',
        render: lazyComponent(() => import('./search/ScopePage'), 'ScopePage'),
        exact: true,
    },
    {
        path: '/api/console',
        render: lazyComponent(() => import('./api/APIConsole'), 'APIConsole'),
        exact: true,
    },
    {
        path: '/users/:username',
        render: lazyComponent(() => import('./user/area/UserArea'), 'UserArea'),
    },
    {
        path: '/survey/:score?',
        render: lazyComponent(() => import('./marketing/SurveyPage'), 'SurveyPage'),
    },
    {
        path: '/extensions',
        render: props => <ExtensionsArea {...props} routes={props.extensionsAreaRoutes} />,
    },
    {
        path: '/help',
        render: lazyComponent(() => import('./marketing/HelpPage'), 'HelpPage'),
    },
    {
        path: '/documentation',
        render: () => {
            // Force a hard reload so that we delegate to the HTTP handler for /documentation, which handles
            // redirecting /documentation to https://docs.sourcegraph.com. That logic is not duplicated in
            // the web app because that would add complexity with no user benefit.
            window.location.reload()
            return null
        },
    },
    {
        path: '/snippets',
        render: lazyComponent(() => import('./snippets/SnippetsPage'), 'SnippetsPage'),
    },
    {
        path: '/:repoRevAndRest+',
        render: lazyComponent(() => import('./repo/RepoContainer'), 'RepoContainer'),
    },
]
