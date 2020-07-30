import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import * as React from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import { combineLatest, merge, of, Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, map, mapTo, startWith, switchMap, filter } from 'rxjs/operators'
import { ActivationProps } from '../../../../shared/src/components/activation/Activation'
import { ExtensionsControllerProps } from '../../../../shared/src/extensions/controller'
import * as GQL from '../../../../shared/src/graphql/schema'
import { PlatformContextProps } from '../../../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../../../shared/src/settings/settings'
import { isErrorLike, asError } from '../../../../shared/src/util/errors'
import { ErrorBoundary } from '../../components/ErrorBoundary'
import { HeroPage } from '../../components/HeroPage'
import { NamespaceProps } from '../../namespaces'
import { ThemeProps } from '../../../../shared/src/theme'
import { RouteDescriptor } from '../../util/contributions'
import { UserSettingsAreaRoute } from '../settings/UserSettingsArea'
import { UserSettingsSidebarItems } from '../settings/UserSettingsSidebar'
import { UserAreaHeader, UserAreaHeaderNavItem } from './UserAreaHeader'
import { PatternTypeProps } from '../../search'
import { ErrorMessage } from '../../components/alerts'
import { isDefined } from '../../../../shared/src/util/types'
import { TelemetryProps } from '../../../../shared/src/telemetry/telemetryService'
import { fetchUser, User } from './backend'
import { OptionalAuthProps } from '../../auth'

const NotFoundPage: React.FunctionComponent = () => (
    <HeroPage icon={MapSearchIcon} title="404: Not Found" subtitle="Sorry, the requested user page was not found." />
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
        OptionalAuthProps,
        Omit<PatternTypeProps, 'setPatternType'> {
    userAreaRoutes: readonly UserAreaRoute[]
    userAreaHeaderNavItems: readonly UserAreaHeaderNavItem[]
    userSettingsSideBarItems: UserSettingsSidebarItems
    userSettingsAreaRoutes: readonly UserSettingsAreaRoute[]
}

interface UserAreaState {
    /**
     * The fetched user (who is the subject of the page), or an error if an error occurred; undefined while
     * loading.
     */
    userOrError?: User | Error
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
        OptionalAuthProps,
        Omit<PatternTypeProps, 'setPatternType'> {
    /** The user area main URL. */
    url: string

    /**
     * The user who is the subject of the page.
     */
    user: User

    /** Called when the user is updated and must be reloaded. */
    onDidUpdateUser: () => void

    userSettingsSideBarItems: UserSettingsSidebarItems
    userSettingsAreaRoutes: readonly UserSettingsAreaRoute[]
}

/**
 * A user's public profile area.
 */
export class UserArea extends React.Component<UserAreaProps, UserAreaState> {
    public state: UserAreaState = {}

    private componentUpdates = new Subject<UserAreaProps>()
    private refreshRequests = new Subject<void>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        // Changes to the route-matched username.
        const usernameChanges = this.componentUpdates.pipe(
            map(props => props.match.params.username),
            distinctUntilChanged()
        )

        // Fetch user.
        this.subscriptions.add(
            combineLatest([usernameChanges, merge(this.refreshRequests.pipe(mapTo(false)), of(true))])
                .pipe(
                    switchMap(([username, forceRefresh]) => {
                        type PartialStateUpdate = Pick<UserAreaState, 'userOrError'>
                        return fetchUser({
                            username,
                            siteAdmin: !!this.props.authenticatedUser?.siteAdmin,
                        }).pipe(
                            filter(isDefined),
                            catchError(error => [asError(error)]),
                            map((userOrError): PartialStateUpdate => ({ userOrError })),

                            // Don't clear old user data while we reload, to avoid unmounting all components during
                            // loading.
                            startWith<PartialStateUpdate>(forceRefresh ? { userOrError: undefined } : {})
                        )
                    })
                )
                .subscribe(
                    stateUpdate => this.setState(stateUpdate),
                    error => console.error(error)
                )
        )

        this.componentUpdates.next(this.props)
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        if (!this.state.userOrError) {
            return null // loading
        }
        if (isErrorLike(this.state.userOrError)) {
            return (
                <HeroPage
                    icon={AlertCircleIcon}
                    title="Error"
                    subtitle={<ErrorMessage error={this.state.userOrError} history={this.props.history} />}
                />
            )
        }

        const context: UserAreaRouteContext = {
            ...this.props,
            url: this.props.match.url,
            user: this.state.userOrError,
            onDidUpdateUser: this.onDidUpdateUser,
            namespace: this.state.userOrError,
        }
        return (
            <div className="user-area w-100">
                <UserAreaHeader
                    {...this.props}
                    {...context}
                    navItems={this.props.userAreaHeaderNavItems}
                    className="border-bottom mt-4"
                />
                <div className="container mt-3">
                    <ErrorBoundary location={this.props.location}>
                        <React.Suspense fallback={<LoadingSpinner className="icon-inline m-2" />}>
                            <Switch>
                                {this.props.userAreaRoutes.map(
                                    ({ path, exact, render, condition = () => true }) =>
                                        condition(context) && (
                                            <Route
                                                // eslint-disable-next-line react/jsx-no-bind
                                                render={routeComponentProps =>
                                                    render({ ...context, ...routeComponentProps })
                                                }
                                                path={this.props.match.url + path}
                                                key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                                                exact={exact}
                                            />
                                        )
                                )}
                                <Route key="hardcoded-key" component={NotFoundPage} />
                            </Switch>
                        </React.Suspense>
                    </ErrorBoundary>
                </div>
            </div>
        )
    }

    private onDidUpdateUser = (): void => {
        this.refreshRequests.next()
    }
}
