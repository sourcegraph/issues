import AddIcon from 'mdi-react/AddIcon'
import DeleteIcon from 'mdi-react/DeleteIcon'
import SettingsIcon from 'mdi-react/SettingsIcon'
import React, { useCallback } from 'react'
import { RouteComponentProps, Redirect } from 'react-router'
import { Link } from 'react-router-dom'
import { Observable, Subject, Subscription, concat, from } from 'rxjs'
import { map, tap, filter, switchMap, mapTo, catchError } from 'rxjs/operators'
import { ActivationProps } from '../../../shared/src/components/activation/Activation'
import { dataOrThrowErrors, gql } from '../../../shared/src/graphql/graphql'
import * as GQL from '../../../shared/src/graphql/schema'
import { createAggregateError, isErrorLike, asError, ErrorLike } from '../../../shared/src/util/errors'
import { mutateGraphQL, queryGraphQL } from '../backend/graphql'
import { FilteredConnection, FilteredConnectionQueryArgs } from '../components/FilteredConnection'
import { PageTitle } from '../components/PageTitle'
import { refreshSiteFlags } from '../site/backend'
import { eventLogger } from '../tracking/eventLogger'
import { ErrorAlert } from '../components/alerts'
import { useEventObservable } from '../../../shared/src/util/useObservable'
import * as H from 'history'
import { ExternalServicesResult, DeleteExternalServiceResult } from '../graphql-operations'

async function deleteExternalService(externalService: GQL.Scalars['ID']): Promise<void> {
    const result = await mutateGraphQL<DeleteExternalServiceResult>(
        gql`
            mutation DeleteExternalService($externalService: ID!) {
                deleteExternalService(externalService: $externalService) {
                    alwaysNil
                }
            }
        `,
        { externalService }
    ).toPromise()
    dataOrThrowErrors(result)
}

interface ExternalServiceNodeProps {
    node: GQL.ExternalService
    onDidUpdate: () => void
    history: H.History
}

const ExternalServiceNode: React.FunctionComponent<ExternalServiceNodeProps> = ({ node, onDidUpdate, history }) => {
    const [nextDeleteClick, deletedOrError] = useEventObservable(
        useCallback(
            (clicks: Observable<React.MouseEvent>) =>
                clicks.pipe(
                    filter(() => window.confirm(`Delete the external service ${node.displayName}?`)),
                    switchMap(() =>
                        concat(
                            ['in-progress' as const],
                            from(deleteExternalService(node.id)).pipe(
                                mapTo(true as const),
                                catchError((error): [ErrorLike] => [asError(error)])
                            )
                        )
                    ),
                    tap(onDidUpdate),
                    tap(deletedOrError => {
                        // eslint-disable-next-line rxjs/no-ignored-subscription
                        refreshSiteFlags().subscribe()
                        if (deletedOrError === true) {
                            history.push('/site-admin/repositories?repositoriesUpdated')
                        }
                    })
                ),
            [history, node.displayName, node.id, onDidUpdate]
        )
    )

    return (
        <li className="external-service-node list-group-item py-2" data-test-external-service-name={node.displayName}>
            <div className="d-flex align-items-center justify-content-between">
                <div>{node.displayName}</div>
                <div>
                    <Link
                        className="btn btn-secondary btn-sm test-edit-external-service-button"
                        to={`/site-admin/external-services/${node.id}`}
                        data-tooltip="External service settings"
                    >
                        <SettingsIcon className="icon-inline" /> Edit
                    </Link>{' '}
                    <button
                        type="button"
                        className="btn btn-sm btn-danger test-delete-external-service-button"
                        onClick={nextDeleteClick}
                        disabled={deletedOrError === 'in-progress'}
                        data-tooltip="Delete external service"
                    >
                        <DeleteIcon className="icon-inline" />
                    </button>
                </div>
            </div>
            {isErrorLike(deletedOrError) && <ErrorAlert className="mt-2" error={deletedOrError} history={history} />}
        </li>
    )
}

interface Props extends RouteComponentProps<{}>, ActivationProps {
    history: H.History
}

interface State {
    noExternalServices?: boolean
}

/**
 * A page displaying the external services on this site.
 */
export class SiteAdminExternalServicesPage extends React.PureComponent<Props, State> {
    private updates = new Subject<void>()
    private subscriptions = new Subscription()

    constructor(props: Props) {
        super(props)
        this.state = {}
    }

    public componentDidMount(): void {
        eventLogger.logViewEvent('SiteAdminExternalServices')
        this.subscriptions.add(
            this.queryExternalServices({ first: 1 }).subscribe(externalServicesResult =>
                this.setState({ noExternalServices: externalServicesResult.totalCount === 0 })
            )
        )
    }

    private completeConnectedCodeHostActivation = (externalServices: GQL.ExternalServiceConnection): void => {
        if (this.props.activation && externalServices.totalCount > 0) {
            this.props.activation.update({ ConnectedCodeHost: true })
        }
    }

    private queryExternalServices = (args: FilteredConnectionQueryArgs): Observable<GQL.ExternalServiceConnection> =>
        queryGraphQL<ExternalServicesResult>(
            gql`
                query ExternalServices($first: Int) {
                    externalServices(first: $first) {
                        nodes {
                            id
                            kind
                            displayName
                            config
                        }
                        totalCount
                        pageInfo {
                            hasNextPage
                        }
                    }
                }
            `,
            {
                first: args.first,
            }
        ).pipe(
            map(({ data, errors }) => {
                if (!data || !data.externalServices || errors) {
                    throw createAggregateError(errors)
                }
                return data.externalServices
            }),
            tap(externalServices => this.completeConnectedCodeHostActivation(externalServices))
        )

    public render(): JSX.Element | null {
        const nodeProps: Omit<ExternalServiceNodeProps, 'node'> = {
            onDidUpdate: this.onDidUpdateExternalServices,
            history: this.props.history,
        }

        if (this.state.noExternalServices) {
            return <Redirect to="/site-admin/external-services/new" />
        }
        return (
            <div className="site-admin-external-services-page">
                <PageTitle title="Manage repositories - Admin" />
                <div className="d-flex justify-content-between align-items-center mb-3">
                    <h2 className="mb-0">Manage repositories</h2>
                    <Link
                        className="btn btn-primary test-goto-add-external-service-page"
                        to="/site-admin/external-services/new"
                    >
                        <AddIcon className="icon-inline" /> Add repositories
                    </Link>
                </div>
                <p className="mt-2">Manage code host connections to sync repositories.</p>
                <FilteredConnection<GQL.ExternalService, Omit<ExternalServiceNodeProps, 'node'>>
                    className="list-group list-group-flush mt-3"
                    noun="external service"
                    pluralNoun="external services"
                    queryConnection={this.queryExternalServices}
                    nodeComponent={ExternalServiceNode}
                    nodeComponentProps={nodeProps}
                    hideSearch={true}
                    noSummaryIfAllNodesVisible={true}
                    updates={this.updates}
                    history={this.props.history}
                    location={this.props.location}
                />
            </div>
        )
    }

    private onDidUpdateExternalServices = (): void => this.updates.next()
}
