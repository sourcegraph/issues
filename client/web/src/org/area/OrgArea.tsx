import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import * as React from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import { combineLatest, merge, Observable, of, Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, map, mapTo, startWith, switchMap } from 'rxjs/operators'

import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { ExtensionsControllerProps } from '@sourcegraph/shared/src/extensions/controller'
import { gql, dataOrThrowErrors } from '@sourcegraph/shared/src/graphql/graphql'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { ErrorLike, isErrorLike, asError } from '@sourcegraph/shared/src/util/errors'

import { AuthenticatedUser } from '../../auth'
import { requestGraphQL } from '../../backend/graphql'
import { BatchChangesProps } from '../../batches/batches'
import { ErrorMessage } from '../../components/alerts'
import { BreadcrumbsProps, BreadcrumbSetters } from '../../components/Breadcrumbs'
import { ErrorBoundary } from '../../components/ErrorBoundary'
import { HeroPage } from '../../components/HeroPage'
import { Page } from '../../components/Page'
import { OrganizationResult, OrganizationVariables, OrgAreaOrganizationFields } from '../../graphql-operations'
import { NamespaceProps } from '../../namespaces'
import { PatternTypeProps } from '../../search'
import { RouteDescriptor } from '../../util/contributions'

import { OrgAreaHeaderNavItem, OrgHeader } from './OrgHeader'
import { OrgInvitationPage } from './OrgInvitationPage'

function queryOrganization(args: { name: string }): Observable<OrgAreaOrganizationFields> {
    return requestGraphQL<OrganizationResult, OrganizationVariables>(
        gql`
            query Organization($name: String!) {
                organization(name: $name) {
                    ...OrgAreaOrganizationFields
                }
            }

            fragment OrgAreaOrganizationFields on Org {
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
        TelemetryProps,
        BreadcrumbsProps,
        BreadcrumbSetters,
        ExtensionsControllerProps,
        BatchChangesProps,
        Omit<PatternTypeProps, 'setPatternType'> {
    orgAreaRoutes: readonly OrgAreaRoute[]
    orgAreaHeaderNavItems: readonly OrgAreaHeaderNavItem[]

    /**
     * The currently authenticated user.
     */
    authenticatedUser: AuthenticatedUser | null
    isSourcegraphDotCom: boolean
}

interface State extends BreadcrumbSetters {
    /**
     * The fetched org or an error if an error occurred; undefined while loading.
     */
    orgOrError?: OrgAreaOrganizationFields | ErrorLike
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
        BreadcrumbsProps,
        BreadcrumbSetters,
        BatchChangesProps,
        Omit<PatternTypeProps, 'setPatternType'> {
    /** The org that is the subject of the page. */
    org: OrgAreaOrganizationFields

    /** Called when the organization is updated and must be reloaded. */
    onOrganizationUpdate: () => void

    /** The currently authenticated user. */
    authenticatedUser: AuthenticatedUser | null

    isSourcegraphDotCom: boolean
}

/**
 * An organization's public profile area.
 */
export class OrgArea extends React.Component<Props> {
    public state: State

    private componentUpdates = new Subject<Props>()
    private refreshRequests = new Subject<void>()
    private subscriptions = new Subscription()

    constructor(props: Props) {
        super(props)
        this.state = {
            setBreadcrumb: props.setBreadcrumb,
            useBreadcrumb: props.useBreadcrumb,
        }
    }

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
                    stateUpdate => {
                        if (stateUpdate.orgOrError && !isErrorLike(stateUpdate.orgOrError)) {
                            const childBreadcrumbSetters = this.props.setBreadcrumb({
                                key: 'OrgArea',
                                link: { to: stateUpdate.orgOrError.url, label: stateUpdate.orgOrError.name },
                            })
                            this.subscriptions.add(childBreadcrumbSetters)
                            this.setState({
                                useBreadcrumb: childBreadcrumbSetters.useBreadcrumb,
                                setBreadcrumb: childBreadcrumbSetters.setBreadcrumb,
                                orgOrError: stateUpdate.orgOrError,
                            })
                        } else {
                            this.setState(stateUpdate)
                        }
                    },
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
                    subtitle={<ErrorMessage error={this.state.orgOrError} />}
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
            isSourcegraphDotCom: this.props.isSourcegraphDotCom,
            batchChangesEnabled: this.props.batchChangesEnabled,
            breadcrumbs: this.props.breadcrumbs,
            setBreadcrumb: this.state.setBreadcrumb,
            useBreadcrumb: this.state.useBreadcrumb,
        }

        if (this.props.location.pathname === `${this.props.match.url}/invitation`) {
            // The OrgInvitationPage is displayed without the OrgHeader because it is modal-like.
            return <OrgInvitationPage {...context} onDidRespondToInvitation={this.onDidRespondToInvitation} />
        }

        return (
            <Page className="org-area">
                <OrgHeader {...this.props} {...context} navItems={this.props.orgAreaHeaderNavItems} />
                <div className="container mt-3">
                    <ErrorBoundary location={this.props.location}>
                        <React.Suspense fallback={<LoadingSpinner className="icon-inline m-2" />}>
                            <Switch>
                                {this.props.orgAreaRoutes.map(
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
                                )}
                                <Route key="hardcoded-key" component={NotFoundPage} />
                            </Switch>
                        </React.Suspense>
                    </ErrorBoundary>
                </div>
            </Page>
        )
    }

    private onDidRespondToInvitation = (): void => this.refreshRequests.next()

    private onDidUpdateOrganization = (): void => this.refreshRequests.next()
}
