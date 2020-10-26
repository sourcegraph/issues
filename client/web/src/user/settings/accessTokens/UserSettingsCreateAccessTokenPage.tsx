import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import AddIcon from 'mdi-react/AddIcon'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { Link } from 'react-router-dom'
import { concat, Observable, Subject, Subscription } from 'rxjs'
import { catchError, concatMap, map, tap } from 'rxjs/operators'
import { gql } from '../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { asError, createAggregateError, ErrorLike, isErrorLike } from '../../../../../shared/src/util/errors'
import { AccessTokenScopes } from '../../../auth/accessToken'
import { mutateGraphQL } from '../../../backend/graphql'
import { Form } from '../../../../../branded/src/components/Form'
import { PageTitle } from '../../../components/PageTitle'
import { SiteAdminAlert } from '../../../site-admin/SiteAdminAlert'
import { eventLogger } from '../../../tracking/eventLogger'
import { UserAreaRouteContext } from '../../area/UserArea'
import { ErrorAlert } from '../../../components/alerts'

function createAccessToken(user: GQL.ID, scopes: string[], note: string): Observable<GQL.ICreateAccessTokenResult> {
    return mutateGraphQL(
        gql`
            mutation CreateAccessToken($user: ID!, $scopes: [String!]!, $note: String!) {
                createAccessToken(user: $user, scopes: $scopes, note: $note) {
                    id
                    token
                }
            }
        `,
        { user, scopes, note }
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.createAccessToken || (errors && errors.length > 0)) {
                eventLogger.log('CreateAccessTokenFailed')
                throw createAggregateError(errors)
            }
            eventLogger.log('AccessTokenCreated')
            return data.createAccessToken
        })
    )
}

interface Props extends UserAreaRouteContext, RouteComponentProps<{}> {
    /** Called when a new access token is created and should be temporarily displayed to the user. */
    onDidCreateAccessToken: (result: GQL.ICreateAccessTokenResult) => void
}

interface State {
    /** The contents of the note input field. */
    note: string

    /** The selected scopes checkboxes. */
    scopes: string[]

    creationOrError?: 'loading' | GQL.ICreateAccessTokenResult | ErrorLike
}

/**
 * A page with a form to create an access token for a user.
 */
export class UserSettingsCreateAccessTokenPage extends React.PureComponent<Props, State> {
    public state: State = {
        note: '',
        scopes: [AccessTokenScopes.UserAll],
    }

    private submits = new Subject<React.FormEvent<HTMLFormElement>>()
    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        eventLogger.logViewEvent('NewAccessToken')
        this.subscriptions.add(
            this.submits
                .pipe(
                    tap(event => event.preventDefault()),
                    concatMap(() =>
                        concat(
                            [{ creationOrError: 'loading' }],
                            createAccessToken(this.props.user.id, this.state.scopes, this.state.note).pipe(
                                tap(result => {
                                    // Go back to access tokens list page and display the token secret value.
                                    this.props.history.push(`${this.props.match.url.replace(/\/new$/, '')}`)
                                    this.props.onDidCreateAccessToken(result)
                                }),
                                map(result => ({ creationOrError: result })),
                                catchError(error => [{ creationOrError: asError(error) }])
                            )
                        )
                    )
                )
                .subscribe(
                    stateUpdate => this.setState(stateUpdate as State),
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
        const siteAdminViewingOtherUser =
            this.props.authenticatedUser && this.props.authenticatedUser.id !== this.props.user.id
        return (
            <div className="user-settings-create-access-token-page">
                <PageTitle title="Create access token" />
                <h2>New access token</h2>
                {siteAdminViewingOtherUser && (
                    <SiteAdminAlert className="sidebar__alert">
                        Creating access token for other user <strong>{this.props.user.username}</strong>
                    </SiteAdminAlert>
                )}
                <Form onSubmit={this.onSubmit}>
                    <div className="form-group">
                        <label htmlFor="user-settings-create-access-token-page__note">Token description</label>
                        <input
                            type="text"
                            className="form-control test-create-access-token-description"
                            id="user-settings-create-access-token-page__note"
                            onChange={this.onNoteChange}
                            required={true}
                            autoFocus={true}
                            placeholder="Description"
                        />
                        <small className="form-help text-muted">What's this token for?</small>
                    </div>
                    <div className="form-group">
                        <label className="mb-1" htmlFor="user-settings-create-access-token-page__note">
                            Token scope
                        </label>
                        <div>
                            <small className="form-help text-muted">
                                Tokens with limited user scopes are not yet supported.
                            </small>
                        </div>
                        <div className="form-check">
                            <input
                                className="form-check-input"
                                type="checkbox"
                                id="user-settings-create-access-token-page__scope-user:all"
                                checked={true}
                                value={AccessTokenScopes.UserAll}
                                onChange={this.onScopesChange}
                                disabled={true}
                            />
                            <label
                                className="form-check-label"
                                htmlFor="user-settings-create-access-token-page__scope-user:all"
                            >
                                <strong>{AccessTokenScopes.UserAll}</strong> — Full control of all resources accessible
                                to the user account
                            </label>
                        </div>
                        {this.props.user.siteAdmin && (
                            <div className="form-check">
                                <input
                                    className="form-check-input"
                                    type="checkbox"
                                    id="user-settings-create-access-token-page__scope-site-admin:sudo"
                                    checked={this.state.scopes.includes(AccessTokenScopes.SiteAdminSudo)}
                                    value={AccessTokenScopes.SiteAdminSudo}
                                    onChange={this.onScopesChange}
                                />
                                <label
                                    className="form-check-label"
                                    htmlFor="user-settings-create-access-token-page__scope-site-admin:sudo"
                                >
                                    <strong>{AccessTokenScopes.SiteAdminSudo}</strong> — Ability to perform any action
                                    as any other user
                                </label>
                            </div>
                        )}
                    </div>
                    <button
                        type="submit"
                        disabled={this.state.creationOrError === 'loading'}
                        className="btn btn-success test-create-access-token-submit"
                    >
                        {this.state.creationOrError === 'loading' ? (
                            <LoadingSpinner className="icon-inline" />
                        ) : (
                            <AddIcon className="icon-inline" />
                        )}{' '}
                        Generate token
                    </button>
                    <Link
                        className="btn btn-secondary ml-1 test-create-access-token-cancel"
                        to={this.props.match.url.replace(/\/new$/, '')}
                    >
                        Cancel
                    </Link>
                </Form>
                {isErrorLike(this.state.creationOrError) && (
                    <ErrorAlert
                        className="invite-form__alert"
                        error={this.state.creationOrError}
                        history={this.props.history}
                    />
                )}
            </div>
        )
    }

    private onNoteChange: React.ChangeEventHandler<HTMLInputElement> = event =>
        this.setState({ note: event.currentTarget.value })

    private onScopesChange: React.ChangeEventHandler<HTMLInputElement> = event => {
        const checked = event.currentTarget.checked
        const value = event.currentTarget.value
        this.setState(previousState => ({
            scopes: checked ? [...previousState.scopes, value] : previousState.scopes.filter(scope => scope !== value),
        }))
    }

    private onSubmit: React.FormEventHandler<HTMLFormElement> = event => this.submits.next(event)
}
