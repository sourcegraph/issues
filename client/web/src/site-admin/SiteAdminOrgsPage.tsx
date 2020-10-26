import AddIcon from 'mdi-react/AddIcon'
import DeleteIcon from 'mdi-react/DeleteIcon'
import SettingsIcon from 'mdi-react/SettingsIcon'
import UserIcon from 'mdi-react/UserIcon'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { Link } from 'react-router-dom'
import { Subject, Subscription } from 'rxjs'
import * as GQL from '../../../shared/src/graphql/schema'
import { pluralize } from '../../../shared/src/util/strings'
import { FilteredConnection } from '../components/FilteredConnection'
import { PageTitle } from '../components/PageTitle'
import { orgURL } from '../org'
import { eventLogger } from '../tracking/eventLogger'
import { deleteOrganization, fetchAllOrganizations } from './backend'
import { ErrorAlert } from '../components/alerts'
import { asError } from '../../../shared/src/util/errors'
import * as H from 'history'

interface OrgNodeProps {
    /**
     * The org to display in this list item.
     */
    node: GQL.IOrg

    /**
     * Called when the org is updated by an action in this list item.
     */
    onDidUpdate?: () => void
    history: H.History
}

interface OrgNodeState {
    loading: boolean
    errorDescription?: string
}

class OrgNode extends React.PureComponent<OrgNodeProps, OrgNodeState> {
    public state: OrgNodeState = {
        loading: false,
    }

    public render(): JSX.Element | null {
        return (
            <li className="list-group-item py-2">
                <div className="d-flex align-items-center justify-content-between">
                    <div>
                        <Link to={orgURL(this.props.node.name)}>
                            <strong>{this.props.node.name}</strong>
                        </Link>
                        <br />
                        <span className="text-muted">{this.props.node.displayName}</span>
                    </div>
                    <div>
                        <Link
                            to={`${orgURL(this.props.node.name)}/settings`}
                            className="btn btn-sm btn-secondary"
                            data-tooltip="Organization settings"
                        >
                            <SettingsIcon className="icon-inline" /> Settings
                        </Link>{' '}
                        <Link
                            to={`${orgURL(this.props.node.name)}/members`}
                            className="btn btn-sm btn-secondary"
                            data-tooltip="Organization members"
                        >
                            <UserIcon className="icon-inline" />{' '}
                            {this.props.node.members && (
                                <>
                                    {this.props.node.members.totalCount}{' '}
                                    {pluralize('member', this.props.node.members.totalCount)}
                                </>
                            )}
                        </Link>{' '}
                        <button
                            type="button"
                            className="btn btn-sm btn-danger"
                            onClick={this.deleteOrg}
                            disabled={this.state.loading}
                            data-tooltip="Delete organization"
                        >
                            <DeleteIcon className="icon-inline" />
                        </button>
                    </div>
                </div>
                {this.state.errorDescription && (
                    <ErrorAlert className="mt-2" error={this.state.errorDescription} history={this.props.history} />
                )}
            </li>
        )
    }

    private deleteOrg = (): void => {
        if (!window.confirm(`Delete the organization ${this.props.node.name}?`)) {
            return
        }

        this.setState({
            errorDescription: undefined,
            loading: true,
        })

        deleteOrganization(this.props.node.id)
            .toPromise()
            .then(
                () => {
                    this.setState({ loading: false })
                    if (this.props.onDidUpdate) {
                        this.props.onDidUpdate()
                    }
                },
                error => this.setState({ loading: false, errorDescription: asError(error).message })
            )
    }
}

interface Props extends RouteComponentProps<{}> {
    history: H.History
}

interface State {
    orgs?: GQL.IOrg[]
    totalCount?: number
}

/**
 * A page displaying the orgs on this site.
 */
export class SiteAdminOrgsPage extends React.Component<Props, State> {
    public state: State = {}

    private orgUpdates = new Subject<void>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        eventLogger.logViewEvent('SiteAdminOrgs')
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        const nodeProps: Omit<OrgNodeProps, 'node'> = {
            onDidUpdate: this.onDidUpdateOrg,
            history: this.props.history,
        }

        return (
            <div className="site-admin-orgs-page">
                <PageTitle title="Organizations - Admin" />
                <div className="d-flex justify-content-between align-items-center mb-3">
                    <h2 className="mb-0">Organizations</h2>
                    <Link to="/organizations/new" className="btn btn-primary test-create-org-button">
                        <AddIcon className="icon-inline" /> Create organization
                    </Link>
                </div>
                <p>
                    An organization is a set of users with associated configuration. See{' '}
                    <Link to="/help/user/organizations">Sourcegraph documentation</Link> for information about
                    configuring organizations.
                </p>
                <FilteredConnection<GQL.IOrg, Omit<OrgNodeProps, 'node'>>
                    className="list-group list-group-flush mt-3"
                    noun="organization"
                    pluralNoun="organizations"
                    queryConnection={fetchAllOrganizations}
                    nodeComponent={OrgNode}
                    nodeComponentProps={nodeProps}
                    updates={this.orgUpdates}
                    history={this.props.history}
                    location={this.props.location}
                />
            </div>
        )
    }

    private onDidUpdateOrg = (): void => this.orgUpdates.next()
}
