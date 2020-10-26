import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import AddIcon from 'mdi-react/AddIcon'
import CloseIcon from 'mdi-react/CloseIcon'
import EmailOpenOutlineIcon from 'mdi-react/EmailOpenOutlineIcon'
import * as React from 'react'
import { Link } from 'react-router-dom'
import { merge, Observable, of, Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, filter, map, mergeMap, startWith, tap, withLatestFrom } from 'rxjs/operators'
import { gql } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { createAggregateError } from '../../../../shared/src/util/errors'
import { mutateGraphQL } from '../../backend/graphql'
import { CopyableText } from '../../components/CopyableText'
import { DismissibleAlert } from '../../components/DismissibleAlert'
import { Form } from '../../../../branded/src/components/Form'
import { eventLogger } from '../../tracking/eventLogger'
import { ErrorAlert } from '../../components/alerts'
import * as H from 'history'
import { AuthenticatedUser } from '../../auth'

function inviteUserToOrganization(
    username: string,
    organization: GQL.ID
): Observable<GQL.IInviteUserToOrganizationResult> {
    return mutateGraphQL(
        gql`
            mutation InviteUserToOrganization($organization: ID!, $username: String!) {
                inviteUserToOrganization(organization: $organization, username: $username) {
                    sentInvitationEmail
                    invitationURL
                }
            }
        `,
        {
            username,
            organization,
        }
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.inviteUserToOrganization || (errors && errors.length > 0)) {
                eventLogger.log('InviteOrgMemberFailed')
                throw createAggregateError(errors)
            }
            eventLogger.log('OrgMemberInvited')
            return data.inviteUserToOrganization
        })
    )
}

function addUserToOrganization(username: string, organization: GQL.ID): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation AddUserToOrganization($organization: ID!, $username: String!) {
                addUserToOrganization(organization: $organization, username: $username) {
                    alwaysNil
                }
            }
        `,
        {
            username,
            organization,
        }
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.addUserToOrganization || (errors && errors.length > 0)) {
                eventLogger.log('AddOrgMemberFailed')
                throw createAggregateError(errors)
            }
            eventLogger.log('OrgMemberAdded')
        })
    )
}

const emailInvitesEnabled = window.context.emailEnabled

const InvitedNotification: React.FunctionComponent<{
    className: string
    username: string
    sentInvitationEmail: boolean
    invitationURL: string
    onDismiss: () => void
}> = ({ className, username, sentInvitationEmail, invitationURL, onDismiss }) => (
    <div className={`${className} invited-notification`}>
        <div className="invited-notification__message">
            {sentInvitationEmail ? (
                <>
                    Invitation sent to {username}. You can also send {username} the invitation link directly:
                </>
            ) : (
                <>Generated invitation link. Copy and send it to {username}:</>
            )}
            <CopyableText text={invitationURL} size={40} className="mt-2" />
        </div>
        <button type="button" className="btn btn-icon" title="Dismiss" onClick={onDismiss}>
            <CloseIcon className="icon-inline" />
        </button>
    </div>
)

interface Props {
    orgID: string
    authenticatedUser: AuthenticatedUser | null

    /** Called when the organization members list changes. */
    onDidUpdateOrganizationMembers: () => void

    onOrganizationUpdate: () => void
    history: H.History
}

interface SubmittedInvite extends Pick<GQL.IInviteUserToOrganizationResult, 'sentInvitationEmail' | 'invitationURL'> {
    username: string
}

interface State {
    username: string

    /** Loading state (undefined means not loading). */
    loading?: 'inviteUserToOrganization' | 'addUserToOrganization'

    invited?: SubmittedInvite[]
    error?: Error
}

export class InviteForm extends React.PureComponent<Props, State> {
    public state: State = { username: '' }

    private submits = new Subject<React.FormEvent<HTMLFormElement>>()
    private inviteClicks = new Subject<React.MouseEvent<HTMLButtonElement>>()
    private usernameChanges = new Subject<string>()
    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        const orgChanges = this.componentUpdates.pipe(distinctUntilChanged((a, b) => a.orgID !== b.orgID))

        type Update = (previousState: State) => State

        this.subscriptions.add(this.usernameChanges.subscribe(username => this.setState({ username })))

        // Invite clicks.
        this.subscriptions.add(
            merge(this.submits.pipe(filter(() => !this.viewerCanAddUserToOrganization)), this.inviteClicks)
                .pipe(
                    tap(event => event.preventDefault()),
                    withLatestFrom(orgChanges, this.usernameChanges),
                    tap(() => eventLogger.log('InviteOrgMemberClicked')),
                    mergeMap(([, { orgID }, username]) =>
                        inviteUserToOrganization(username, orgID).pipe(
                            tap(() => this.props.onOrganizationUpdate()),
                            tap(() => this.usernameChanges.next('')),
                            mergeMap(({ sentInvitationEmail, invitationURL }) =>
                                // Reset email, reenable submit button, flash "invited" text
                                of(
                                    (state: State): State => ({
                                        ...state,
                                        loading: undefined,
                                        error: undefined,
                                        username: '',
                                        invited: [
                                            ...(state.invited || []),
                                            { username, sentInvitationEmail, invitationURL },
                                        ],
                                    })
                                )
                            ),
                            // Disable button while loading
                            startWith<Update>(
                                (state: State): State => ({
                                    ...state,
                                    loading: 'inviteUserToOrganization',
                                })
                            ),
                            catchError(error => [(state: State): State => ({ ...state, loading: undefined, error })])
                        )
                    )
                )
                .subscribe(
                    stateUpdate => this.setState(stateUpdate),
                    error => console.error(error)
                )
        )

        // Adds.
        this.subscriptions.add(
            this.submits
                .pipe(filter(() => this.viewerCanAddUserToOrganization))
                .pipe(
                    tap(event => event.preventDefault()),
                    withLatestFrom(orgChanges, this.usernameChanges),
                    mergeMap(([, { orgID }, username]) =>
                        addUserToOrganization(username, orgID).pipe(
                            tap(() => this.props.onDidUpdateOrganizationMembers()),
                            tap(() => this.usernameChanges.next('')),
                            mergeMap(() =>
                                // Reset email, reenable submit button, flash "invited" text
                                of(
                                    (state: State): State => ({
                                        ...state,
                                        loading: undefined,
                                        error: undefined,
                                        username: '',
                                    })
                                )
                            ),
                            // Disable button while loading
                            startWith<Update>(
                                (state: State): State => ({
                                    ...state,
                                    loading: 'addUserToOrganization',
                                })
                            ),
                            catchError(error => [(state: State): State => ({ ...state, loading: undefined, error })])
                        )
                    )
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

    private get viewerCanAddUserToOrganization(): boolean {
        return !!this.props.authenticatedUser && this.props.authenticatedUser.siteAdmin
    }

    public render(): JSX.Element | null {
        const viewerCanAddUserToOrganization = this.viewerCanAddUserToOrganization

        return (
            <div className="invite-form">
                <div className="card invite-form__container">
                    <div className="card-body">
                        <h4 className="card-title">
                            {this.viewerCanAddUserToOrganization ? 'Add or invite member' : 'Invite member'}
                        </h4>
                        <Form className="form-inline align-items-start" onSubmit={this.onSubmit}>
                            <label className="sr-only" htmlFor="invite-form__username">
                                Username
                            </label>
                            <input
                                type="text"
                                className="form-control mb-2 mr-sm-2"
                                id="invite-form__username"
                                placeholder="Username"
                                onChange={this.onUsernameChange}
                                value={this.state.username}
                                autoComplete="off"
                                autoCapitalize="off"
                                autoCorrect="off"
                                required={true}
                                spellCheck={false}
                                size={30}
                            />
                            {viewerCanAddUserToOrganization && (
                                <button
                                    type="submit"
                                    disabled={!!this.state.loading}
                                    className="btn btn-primary mb-2 mr-sm-2"
                                    data-tooltip="Add immediately without sending invitation (site admins only)"
                                >
                                    {this.state.loading === 'addUserToOrganization' ? (
                                        <LoadingSpinner className="icon-inline" />
                                    ) : (
                                        <AddIcon className="icon-inline" />
                                    )}{' '}
                                    Add member
                                </button>
                            )}
                            {(emailInvitesEnabled || !this.viewerCanAddUserToOrganization) && (
                                <div className="form-group flex-column mb-2 mr-sm-2">
                                    {/* eslint-disable-next-line react/button-has-type */}
                                    <button
                                        // eslint-disable-next-line react/button-has-type
                                        type={viewerCanAddUserToOrganization ? 'button' : 'submit'}
                                        disabled={!!this.state.loading}
                                        className={`btn ${
                                            viewerCanAddUserToOrganization ? 'btn-secondary' : 'btn-primary'
                                        }`}
                                        data-tooltip={
                                            emailInvitesEnabled
                                                ? 'Send invitation email with link to join this organization'
                                                : 'Generate invitation link to manually send to user'
                                        }
                                        onClick={viewerCanAddUserToOrganization ? this.onInviteClick : undefined}
                                    >
                                        {this.state.loading === 'inviteUserToOrganization' ? (
                                            <LoadingSpinner className="icon-inline" />
                                        ) : (
                                            <EmailOpenOutlineIcon className="icon-inline" />
                                        )}{' '}
                                        {emailInvitesEnabled
                                            ? this.viewerCanAddUserToOrganization
                                                ? 'Send invitation to join'
                                                : 'Send invitation'
                                            : 'Generate invitation link'}
                                    </button>
                                </div>
                            )}
                        </Form>
                    </div>
                </div>
                {this.props.authenticatedUser?.siteAdmin && !window.context.emailEnabled && (
                    <DismissibleAlert className="alert-info" partialStorageKey="org-invite-email-config">
                        <p className=" mb-0">
                            Set <code>email.smtp</code> in{' '}
                            <Link to="/site-admin/configuration">site configuration</Link> to send email notfications
                            about invitations.
                        </p>
                    </DismissibleAlert>
                )}
                {this.state.invited?.map(({ username, sentInvitationEmail, invitationURL }, index) => (
                    /* eslint-disable react/jsx-no-bind */
                    <InvitedNotification
                        key={index}
                        className="alert alert-success invite-form__alert"
                        username={username}
                        sentInvitationEmail={sentInvitationEmail}
                        invitationURL={invitationURL}
                        onDismiss={() => this.dismissNotification(index)}
                    />
                    /* eslint-enable react/jsx-no-bind */
                ))}
                {this.state.error && (
                    <ErrorAlert className="invite-form__alert" error={this.state.error} history={this.props.history} />
                )}
            </div>
        )
    }

    private onUsernameChange: React.ChangeEventHandler<HTMLInputElement> = event =>
        this.usernameChanges.next(event.currentTarget.value)
    private onSubmit: React.FormEventHandler<HTMLFormElement> = event => this.submits.next(event)
    private onInviteClick: React.MouseEventHandler<HTMLButtonElement> = event => this.inviteClicks.next(event)

    private dismissNotification = (dismissedIndex: number): void => {
        this.setState(previousState => ({
            invited: (previousState.invited || []).filter((invite, index) => dismissedIndex !== index),
        }))
    }
}
