import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { combineLatest, concat, Observable, Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, filter, map, mergeMap, startWith, switchMap, tap } from 'rxjs/operators'
import { USER_DISPLAY_NAME_MAX_LENGTH } from '../..'
import { percentageDone } from '../../../../../shared/src/components/activation/Activation'
import { ActivationChecklist } from '../../../../../shared/src/components/activation/ActivationChecklist'
import { gql } from '../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { asError, createAggregateError, ErrorLike, isErrorLike } from '../../../../../shared/src/util/errors'
import { refreshAuthenticatedUser } from '../../../auth'
import { UsernameInput } from '../../../auth/SignInSignUpCommon'
import { queryGraphQL } from '../../../backend/graphql'
import { Form } from '../../../components/Form'
import { PageTitle } from '../../../components/PageTitle'
import { eventLogger } from '../../../tracking/eventLogger'
import { UserAreaRouteContext } from '../../area/UserArea'
import { UserAvatar } from '../../UserAvatar'
import { updateUser } from '../backend'
import { ErrorAlert } from '../../../components/alerts'
import { UserForProfilePageResult } from '../../../graphql-operations'

function queryUser(user: GQL.Scalars['ID']): Observable<GQL.User> {
    return queryGraphQL<UserForProfilePageResult>(
        gql`
            query UserForProfilePage($user: ID!) {
                node(id: $user) {
                    ... on User {
                        id
                        username
                        displayName
                        avatarURL
                        viewerCanChangeUsername
                    }
                }
            }
        `,
        { user }
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.node) {
                throw createAggregateError(errors)
            }
            return data.node as GQL.User
        })
    )
}

interface Props extends UserAreaRouteContext, RouteComponentProps<{}> {}

interface State {
    /** The user to edit, or an error, or undefined while loading. */
    userOrError?: GQL.User | ErrorLike

    loading: boolean
    saved: boolean
    error?: ErrorLike

    /** undefined means unchanged from Props.user */
    username?: string
    displayName?: string
    avatarURL?: string
}

export class UserSettingsProfilePage extends React.Component<Props, State> {
    public state: State = { loading: false, saved: false }

    private componentUpdates = new Subject<Props>()
    private refreshRequests = new Subject<void>()
    private submits = new Subject<React.FormEvent<HTMLFormElement>>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        eventLogger.logViewEvent('UserProfile')

        const userChanges = this.componentUpdates.pipe(
            distinctUntilChanged((a, b) => a.user.id === b.user.id),
            map(({ user }) => user)
        )

        // Reset the fields upon navigation to a different user.
        this.subscriptions.add(
            userChanges.subscribe(() =>
                this.setState({
                    userOrError: undefined,
                    loading: false,
                    saved: false,
                    username: undefined,
                    displayName: undefined,
                    avatarURL: undefined,
                })
            )
        )

        // Fetch the user with all of the fields we need (Props.user might not have them all).
        this.subscriptions.add(
            combineLatest([userChanges, this.refreshRequests.pipe(startWith<void>(undefined))])
                .pipe(
                    switchMap(([user]) =>
                        queryUser(user.id).pipe(
                            catchError(error => [asError(error)]),
                            map((userOrError): Pick<State, 'userOrError'> => ({ userOrError }))
                        )
                    )
                )
                .subscribe(
                    stateUpdate => this.setState(stateUpdate),
                    error => console.error(error)
                )
        )

        this.subscriptions.add(
            this.submits
                .pipe(
                    tap(event => {
                        event.preventDefault()
                        eventLogger.log('UpdateUserClicked')
                    }),
                    filter(event => event.currentTarget.checkValidity()),
                    tap(() => this.setState({ loading: true })),
                    mergeMap(() =>
                        updateUser(this.props.user.id, {
                            username: this.state.username === undefined ? null : this.state.username,
                            displayName: this.state.displayName === undefined ? null : this.state.displayName,
                            avatarURL: this.state.avatarURL === undefined ? null : this.state.avatarURL,
                        }).pipe(catchError(error => this.handleError(error)))
                    ),
                    tap(() => {
                        this.setState({ loading: false, saved: true })
                        this.props.onDidUpdateUser()

                        // Handle when username changes.
                        if (this.state.username !== undefined && this.state.username !== this.props.user.username) {
                            this.props.history.push(`/users/${this.state.username}/settings/profile`)
                            return
                        }

                        this.refreshRequests.next()
                        setTimeout(() => this.setState({ saved: false }), 500)
                    }),

                    // In case the edited user is the current user, immediately reflect the changes in the UI.
                    mergeMap(() => concat(refreshAuthenticatedUser(), [null]))
                )
                .subscribe({ error: error => this.handleError(error) })
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
        return (
            <div className="user-settings-profile-page">
                <PageTitle title="Profile" />
                <h2>Profile</h2>

                {this.props.activation?.completed && percentageDone(this.props.activation.completed) < 100 && (
                    <div className="card mb-3">
                        <div className="card-body">
                            <h3 className="mb-0">Almost there!</h3>
                            <p className="mb-0">Complete the steps below to finish onboarding to Sourcegraph.</p>
                        </div>
                        <ActivationChecklist
                            history={this.props.history}
                            steps={this.props.activation.steps}
                            completed={this.props.activation.completed}
                        />
                    </div>
                )}

                {isErrorLike(this.state.userOrError) && (
                    <ErrorAlert error={this.state.userOrError.message} history={this.props.history} />
                )}
                {this.state.error && <ErrorAlert error={this.state.error.message} history={this.props.history} />}
                {this.state.userOrError && !isErrorLike(this.state.userOrError) && (
                    <Form className="user-settings-profile-page__form" onSubmit={this.handleSubmit}>
                        <div className="form-group">
                            <label htmlFor="user-settings-profile-page__form-username">Username</label>
                            <UsernameInput
                                id="user-settings-profile-page__form-username"
                                className="test-user-settings-profile-page-username"
                                value={
                                    this.state.username === undefined
                                        ? this.state.userOrError.username
                                        : this.state.username
                                }
                                onChange={this.onUsernameFieldChange}
                                required={true}
                                disabled={
                                    !this.state.userOrError ||
                                    !this.state.userOrError.viewerCanChangeUsername ||
                                    this.state.loading
                                }
                                aria-describedby="user-settings-profile-page__form-username-help"
                            />
                            <small className="form-text text-muted">
                                A username consists of letters, numbers, hyphens (-), dots (.) and may not begin or end
                                with a dot, nor begin with a hyphen.
                            </small>
                        </div>
                        <div className="form-group">
                            <label htmlFor="user-settings-profile-page__form-display-name">Display name</label>
                            <input
                                id="user-settings-profile-page__form-display-name"
                                type="text"
                                className="form-control test-user-settings-profile-page__display-name"
                                value={
                                    this.state.displayName === undefined
                                        ? this.state.userOrError.displayName || ''
                                        : this.state.displayName
                                }
                                onChange={this.onDisplayNameFieldChange}
                                disabled={this.state.loading}
                                spellCheck={false}
                                placeholder="Display name"
                                maxLength={USER_DISPLAY_NAME_MAX_LENGTH}
                            />
                        </div>
                        <div className="user-settings-profile-page__avatar-row">
                            <div className="form-group user-settings-profile-page__field-column">
                                <label htmlFor="user-settings-profile-page__form-avatar-url">Avatar URL</label>
                                <input
                                    id="user-settings-profile-page__form-avatar-url"
                                    type="url"
                                    className="form-control test-user-settings-profile-page__avatar_url"
                                    value={
                                        this.state.avatarURL === undefined
                                            ? this.state.userOrError.avatarURL || ''
                                            : this.state.avatarURL
                                    }
                                    onChange={this.onAvatarURLFieldChange}
                                    disabled={this.state.loading}
                                    spellCheck={false}
                                    placeholder="URL to avatar photo"
                                />
                            </div>
                            {this.state.userOrError.avatarURL && (
                                <div className="user-settings-profile-page__avatar-column">
                                    <UserAvatar user={this.state.userOrError} />
                                </div>
                            )}
                        </div>
                        <button
                            className="btn btn-primary user-settings-profile-page__button test-user-settings-profile-page-update-profile"
                            type="submit"
                            disabled={this.state.loading}
                        >
                            Update profile
                        </button>
                        {this.state.loading && (
                            <div>
                                <LoadingSpinner className="icon-inline" />
                            </div>
                        )}
                        {this.state.saved && (
                            <p className="alert alert-success user-settings-profile-page__alert test-user-settings-profile-page-alert-success">
                                Profile saved!
                            </p>
                        )}
                        {window.context.sourcegraphDotComMode && (
                            <p className="mt-4">
                                <a href="https://about.sourcegraph.com/contact">Contact support</a> to delete your
                                account.
                            </p>
                        )}
                    </Form>
                )}
            </div>
        )
    }

    private onUsernameFieldChange = (event: React.ChangeEvent<HTMLInputElement>): void => {
        this.setState({ username: event.target.value })
    }

    private onDisplayNameFieldChange = (event: React.ChangeEvent<HTMLInputElement>): void => {
        this.setState({ displayName: event.target.value })
    }

    private onAvatarURLFieldChange = (event: React.ChangeEvent<HTMLInputElement>): void => {
        this.setState({ avatarURL: event.target.value })
    }

    private handleSubmit = (event: React.FormEvent<HTMLFormElement>): void => {
        this.submits.next(event)
    }

    private handleError = (error: Error): [] => {
        console.error(error)
        this.setState({ loading: false, saved: false, error })
        return []
    }
}
