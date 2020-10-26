/* eslint rxjs/no-ignored-subscription: warn */
import DeleteIcon from 'mdi-react/DeleteIcon'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { Observable, Subject, Subscription } from 'rxjs'
import { map } from 'rxjs/operators'
import { gql } from '../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { createAggregateError, asError } from '../../../../../shared/src/util/errors'
import { mutateGraphQL, queryGraphQL } from '../../../backend/graphql'
import { FilteredConnection } from '../../../components/FilteredConnection'
import { PageTitle } from '../../../components/PageTitle'
import { SiteFlags } from '../../../site'
import { siteFlags } from '../../../site/backend'
import { eventLogger } from '../../../tracking/eventLogger'
import { setUserEmailVerified } from '../backend'
import { AddUserEmailForm } from './AddUserEmailForm'
import { ErrorAlert } from '../../../components/alerts'
import * as H from 'history'
import { UserAreaUserFields } from '../../../graphql-operations'

interface UserEmailNodeProps {
    node: GQL.IUserEmail
    user: UserAreaUserFields

    onDidUpdate: () => void
    history: H.History
}

interface UserEmailNodeState {
    loading: boolean
    errorDescription?: string
}

class UserEmailNode extends React.PureComponent<UserEmailNodeProps, UserEmailNodeState> {
    public state: UserEmailNodeState = {
        loading: false,
    }

    public render(): JSX.Element | null {
        let verifiedFragment: React.ReactFragment
        if (this.props.node.verified) {
            verifiedFragment = <span className="badge badge-success">Verified</span>
        } else if (this.props.node.verificationPending) {
            verifiedFragment = <span className="badge badge-info">Verification pending</span>
        } else {
            verifiedFragment = <span className="badge badge-secondary">Not verified</span>
        }

        return (
            <li className="list-group-item py-2">
                <div className="d-flex align-items-center justify-content-between">
                    <div>
                        <strong>{this.props.node.email}</strong> &nbsp;{verifiedFragment}&nbsp;
                        {this.props.node.isPrimary && <span className="badge badge-primary">Primary</span>}
                    </div>
                    <div>
                        <button
                            type="button"
                            className="btn btn-sm btn-danger"
                            onClick={this.remove}
                            disabled={this.state.loading}
                            data-tooltip="Remove email address"
                        >
                            <DeleteIcon className="icon-inline" />
                        </button>{' '}
                        {this.props.node.viewerCanManuallyVerify && (
                            <button
                                type="button"
                                className="btn btn-sm btn-secondary"
                                onClick={this.props.node.verified ? this.setAsUnverified : this.setAsVerified}
                                disabled={this.state.loading}
                            >
                                {this.props.node.verified ? 'Mark as unverified' : 'Mark as verified'}
                            </button>
                        )}
                    </div>
                </div>
                {this.state.errorDescription && (
                    <ErrorAlert className="mt-2" error={this.state.errorDescription} history={this.props.history} />
                )}
            </li>
        )
    }

    private remove = (): void => {
        if (!window.confirm(`Remove the email address ${this.props.node.email}?`)) {
            return
        }

        this.setState({
            errorDescription: undefined,
            loading: true,
        })

        mutateGraphQL(
            gql`
                mutation RemoveUserEmail($user: ID!, $email: String!) {
                    removeUserEmail(user: $user, email: $email) {
                        alwaysNil
                    }
                }
            `,
            { user: this.props.user.id, email: this.props.node.email }
        )
            .pipe(
                map(({ data, errors }) => {
                    if (!data || (errors && errors.length > 0)) {
                        throw createAggregateError(errors)
                    }
                })
            )
            .subscribe(
                () => {
                    this.setState({ loading: false })
                    eventLogger.log('UserEmailAddressDeleted')
                    if (this.props.onDidUpdate) {
                        this.props.onDidUpdate()
                    }
                },
                error => this.setState({ loading: false, errorDescription: asError(error).message })
            )
    }

    private setAsVerified = (): void => this.setVerified(true)
    private setAsUnverified = (): void => this.setVerified(false)

    private setVerified(verified: boolean): void {
        this.setState({
            errorDescription: undefined,
            loading: true,
        })

        // TODO this may call setState() after the component was unmounted
        // eslint-disable-next-line rxjs/no-ignored-subscription
        setUserEmailVerified(this.props.user.id, this.props.node.email, verified).subscribe(
            () => {
                this.setState({ loading: false })
                if (verified) {
                    eventLogger.log('UserEmailAddressMarkedVerified')
                } else {
                    eventLogger.log('UserEmailAddressMarkedUnverified')
                }
                if (this.props.onDidUpdate) {
                    this.props.onDidUpdate()
                }
            },
            error => this.setState({ loading: false, errorDescription: asError(error).message })
        )
    }
}

interface Props extends RouteComponentProps<{}> {
    user: UserAreaUserFields
    history: H.History
}

interface State {
    siteFlags?: SiteFlags
}

/** We fake a XyzConnection type because our GraphQL API doesn't have one (or need one) for user emails. */
interface UserEmailConnection {
    nodes: GQL.IUserEmail[]
    totalCount: number
}

export class UserSettingsEmailsPage extends React.Component<Props, State> {
    public state: State = {}

    private userEmailUpdates = new Subject<void>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        eventLogger.logViewEvent('UserSettingsEmails')

        this.subscriptions.add(siteFlags.subscribe(siteFlags => this.setState({ siteFlags })))
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        const nodeProps: Omit<UserEmailNodeProps, 'node'> = {
            user: this.props.user,
            onDidUpdate: this.onDidUpdateUserEmail,
            history: this.props.history,
        }

        return (
            <div className="user-settings-emails-page">
                <PageTitle title="Emails" />
                <h2>Emails</h2>
                {this.state.siteFlags && !this.state.siteFlags.sendsEmailVerificationEmails && (
                    <div className="alert alert-warning mt-2">
                        Sourcegraph is not configured to send email verifications. Newly added email addresses must be
                        manually verified by a site admin.
                    </div>
                )}
                <FilteredConnection<GQL.IUserEmail, Omit<UserEmailNodeProps, 'node'>>
                    className="list-group list-group-flush mt-3"
                    noun="email address"
                    pluralNoun="email addresses"
                    queryConnection={this.queryUserEmails}
                    nodeComponent={UserEmailNode}
                    nodeComponentProps={nodeProps}
                    updates={this.userEmailUpdates}
                    hideSearch={true}
                    noSummaryIfAllNodesVisible={true}
                    history={this.props.history}
                    location={this.props.location}
                />
                <AddUserEmailForm
                    className="mt-4"
                    user={this.props.user.id}
                    onDidAdd={this.onDidUpdateUserEmail}
                    history={this.props.history}
                />
            </div>
        )
    }

    private queryUserEmails = (): Observable<UserEmailConnection> =>
        queryGraphQL(
            gql`
                query UserEmails($user: ID!) {
                    node(id: $user) {
                        ... on User {
                            emails {
                                email
                                isPrimary
                                verified
                                verificationPending
                                viewerCanManuallyVerify
                            }
                        }
                    }
                }
            `,
            { user: this.props.user.id }
        ).pipe(
            map(({ data, errors }) => {
                if (!data || !data.node) {
                    throw createAggregateError(errors)
                }
                const user = data.node as GQL.IUser
                if (!user.emails) {
                    throw createAggregateError(errors)
                }
                return {
                    nodes: user.emails,
                    totalCount: user.emails.length,
                }
            })
        )

    private onDidUpdateUserEmail = (): void => this.userEmailUpdates.next()
}
