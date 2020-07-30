import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import * as React from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import { combineLatest, merge, Observable, of, Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, map, mapTo, startWith, switchMap } from 'rxjs/operators'
import { ExtensionsControllerProps } from '../../../../shared/src/extensions/controller'
import { gql, dataOrThrowErrors } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { PlatformContextProps } from '../../../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../../../shared/src/settings/settings'
import { ErrorLike, isErrorLike, asError } from '../../../../shared/src/util/errors'
import { queryGraphQL } from '../../backend/graphql'
import { ErrorBoundary } from '../../components/ErrorBoundary'
import { HeroPage } from '../../components/HeroPage'
import { NamespaceProps } from '../../namespaces'
import { RouteDescriptor } from '../../util/contributions'
import { OrgAreaHeaderNavItem, OrgHeader } from './OrgHeader'
import { OrgInvitationPage } from './OrgInvitationPage'
import { PatternTypeProps } from '../../search'
import { ThemeProps } from '../../../../shared/src/theme'
import { ErrorMessage } from '../../components/alerts'
import * as H from 'history'
import { TelemetryProps } from '../../../../shared/src/telemetry/telemetryService'
import { OrganizationResult } from '../../graphql-operations'
import { OptionalAuthProps } from '../../auth'

function queryOrganization(args: { name: string }): Observable<GQL.Org> {
    return queryGraphQL<OrganizationResult>(
        gql`
            query Organization($name: String!) {
                organization(name: $name) {
                    __typename
                    id
                    name
                    displayName
                    url
                    settingsURL
                    viewerPendingInvitation {
                        id
                        sender {
                            username
                            displayName
                            avatarURL
                            createdAt
                        }
                        respondURL
                    }
                    viewerIsMember
                    viewerCanAdminister
                    createdAt
                }
            }
        `,
        args
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.organization) {
                throw new Error(`Organization not found: ${JSON.stringify(args.name)}`)
            }
            return data.organization
        })
    )
}

const NotFoundPage: React.FunctionComponent = () => (
    <HeroPage icon={MapSearchIcon} title="404: Not Found" subtitle="Sorry, the requested organization was not found." />
)

export interface OrgAreaRoute extends RouteDescriptor<OrgAreaPageProps> {}

interface Props
    extends RouteComponentProps<{ name: string }>,
        PlatformContextProps,
        SettingsCascadeProps,
        ThemeProps,
        OptionalAuthProps,
        TelemetryProps,
        ExtensionsControllerProps,
        Omit<PatternTypeProps, 'setPatternType'> {
    orgAreaRoutes: readonly OrgAreaRoute[]
    orgAreaHeaderNavItems: readonly OrgAreaHeaderNavItem[]
    history: H.History
}

interface State {
    /**
     * The fetched org or an error if an error occurred; undefined while loading.
     */
    orgOrError?: GQL.Org | ErrorLike
}

/**
 * Properties passed to all page components in the org area.
 */
export interface OrgAreaPageProps
    extends ExtensionsControllerProps,
        PlatformContextProps,
        SettingsCascadeProps,
        ThemeProps,
        TelemetryProps,
        NamespaceProps,
        OptionalAuthProps,
        Omit<PatternTypeProps, 'setPatternType'> {
    /** The org that is the subject of the page. */
    org: GQL.Org

    /** Called when the organization is updated and must be reloaded. */
    onOrganizationUpdate: () => void
}

/**
 * An organization's public profile area.
 */
export class OrgArea extends React.Component<Props> {
    public state: State = {}

    private componentUpdates = new Subject<Props>()
    private refreshRequests = new Subject<void>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        // Changes to the route-matched org name.
        const nameChanges = this.componentUpdates.pipe(
            map(props => props.match.params.name),
            distinctUntilChanged()
        )

        // Fetch organization.
        this.subscriptions.add(
            combineLatest([nameChanges, merge(this.refreshRequests.pipe(mapTo(false)), of(true))])
                .pipe(
                    switchMap(([name, forceRefresh]) => {
                        type PartialStateUpdate = Pick<State, 'orgOrError'>
                        return queryOrganization({ name }).pipe(
                            catchError((error): [ErrorLike] => [asError(error)]),
                            map((orgOrError): PartialStateUpdate => ({ orgOrError })),

                            // Don't clear old org data while we reload, to avoid unmounting all components during
                            // loading.
                            startWith<PartialStateUpdate>(forceRefresh ? { orgOrError: undefined } : {})
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
        if (!this.state.orgOrError) {
            return null // loading
        }
        if (isErrorLike(this.state.orgOrError)) {
            return (
                <HeroPage
                    icon={AlertCircleIcon}
                    title="Error"
                    subtitle={<ErrorMessage error={this.state.orgOrError} history={this.props.history} />}
                />
            )
        }

        const context: OrgAreaPageProps = {
            authenticatedUser: this.props.authenticatedUser,
            org: this.state.orgOrError,
            onOrganizationUpdate: this.onDidUpdateOrganization,
            extensionsController: this.props.extensionsController,
            platformContext: this.props.platformContext,
            settingsCascade: this.props.settingsCascade,
            isLightTheme: this.props.isLightTheme,
            namespace: this.state.orgOrError,
            patternType: this.props.patternType,
            telemetryService: this.props.telemetryService,
        }

        if (this.props.location.pathname === `${this.props.match.url}/invitation`) {
            // The OrgInvitationPage is displayed without the OrgHeader because it is modal-like.
            return (
                <OrgInvitationPage
                    {...context}
                    onDidRespondToInvitation={this.onDidRespondToInvitation}
                    history={this.props.history}
                />
            )
        }

        return (
            <div className="org-area w-100">
                <OrgHeader
                    {...this.props}
                    {...context}
                    navItems={this.props.orgAreaHeaderNavItems}
                    className="border-bottom mt-4"
                />
                <div className="container mt-3">
                    <ErrorBoundary location={this.props.location}>
                        <React.Suspense fallback={<LoadingSpinner className="icon-inline m-2" />}>
                            <Switch>
                                {this.props.orgAreaRoutes.map(
                                    /* eslint-disable react/jsx-no-bind */
                                    ({ path, exact, render, condition = () => true }) =>
                                        condition(context) && (
                                            <Route
                                                path={this.props.match.url + path}
                                                key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                                                exact={exact}
                                                render={routeComponentProps =>
                                                    render({ ...context, ...routeComponentProps })
                                                }
                                            />
                                        )
                                    /* eslint-enable react/jsx-no-bind */
                                )}
                                <Route key="hardcoded-key" component={NotFoundPage} />
                            </Switch>
                        </React.Suspense>
                    </ErrorBoundary>
                </div>
            </div>
        )
    }

    private onDidRespondToInvitation = (): void => this.refreshRequests.next()

    private onDidUpdateOrganization = (): void => this.refreshRequests.next()
}
