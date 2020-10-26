import * as React from 'react'
import { Link } from 'react-router-dom'
import { Observable, Subject, Subscription } from 'rxjs'
import { catchError, filter, map, mapTo, startWith, switchMap, tap } from 'rxjs/operators'
import { gql } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { asError, createAggregateError, ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import { mutateGraphQL } from '../../backend/graphql'
import { Timestamp } from '../../components/time/Timestamp'
import { userURL } from '../../user'
import { AccessTokenCreatedAlert } from './AccessTokenCreatedAlert'
import { ErrorAlert } from '../../components/alerts'
import * as H from 'history'

export const accessTokenFragment = gql`
    fragment AccessTokenFields on AccessToken {
        id
        scopes
        note
        createdAt
        lastUsedAt
        subject {
            username
        }
        creator {
            username
        }
    }
`

function deleteAccessToken(tokenID: GQL.ID): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation DeleteAccessToken($tokenID: ID!) {
                deleteAccessToken(byID: $tokenID) {
                    alwaysNil
                }
            }
        `,
        { tokenID }
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.deleteAccessToken || (errors && errors.length > 0)) {
                throw createAggregateError(errors)
            }
        })
    )
}

export interface AccessTokenNodeProps {
    node: GQL.IAccessToken

    /** Whether the token's subject user should be displayed. */
    showSubject?: boolean

    /**
     * The newly created token, if any. This contains the secret for this node's token iff node.id
     * === newToken.id.
     */
    newToken?: GQL.ICreateAccessTokenResult

    onDidUpdate: () => void
    history: H.History
}

interface AccessTokenNodeState {
    /** Undefined means in progress, null means done or not started. */
    deletionOrError?: null | ErrorLike
}

export class AccessTokenNode extends React.PureComponent<AccessTokenNodeProps, AccessTokenNodeState> {
    public state: AccessTokenNodeState = { deletionOrError: null }

    private deletes = new Subject<void>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        this.subscriptions.add(
            this.deletes
                .pipe(
                    filter(() =>
                        window.confirm(
                            'Delete and revoke this token? Any clients using it will no longer be able to access the Sourcegraph API.'
                        )
                    ),
                    switchMap(() =>
                        deleteAccessToken(this.props.node.id).pipe(
                            mapTo(null),
                            catchError(error => [asError(error)]),
                            map(deletionOrError => ({ deletionOrError })),
                            tap(() => {
                                if (this.props.onDidUpdate) {
                                    this.props.onDidUpdate()
                                }
                            }),
                            startWith<Pick<AccessTokenNodeState, 'deletionOrError'>>({ deletionOrError: undefined })
                        )
                    )
                )
                .subscribe(
                    stateUpdate => this.setState(stateUpdate),
                    error => console.error(error)
                )
        )
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        const note = this.props.node.note || '(no description)'
        const loading = this.state.deletionOrError === undefined
        return (
            <li className="list-group-item p-3 d-block" data-test-access-token-description={this.props.node.note}>
                <div className="d-flex w-100 justify-content-between">
                    <div className="mr-2">
                        {this.props.showSubject ? (
                            <>
                                <strong>
                                    <Link to={userURL(this.props.node.subject.username)}>
                                        {this.props.node.subject.username}
                                    </Link>
                                </strong>{' '}
                                &mdash; {note}
                            </>
                        ) : (
                            <strong>{note}</strong>
                        )}{' '}
                        <small className="text-muted">
                            {' '}
                            &mdash; <em>{this.props.node.scopes?.join(', ')}</em>
                            <br />
                            {this.props.node.lastUsedAt ? (
                                <>
                                    Last used <Timestamp date={this.props.node.lastUsedAt} />
                                </>
                            ) : (
                                'Never used'
                            )}
                            , created <Timestamp date={this.props.node.createdAt} />
                            {this.props.node.subject.username !== this.props.node.creator.username && (
                                <>
                                    {' '}
                                    by{' '}
                                    <Link to={userURL(this.props.node.creator.username)}>
                                        {this.props.node.creator.username}
                                    </Link>
                                </>
                            )}
                        </small>
                    </div>
                    <div>
                        <button
                            type="button"
                            className="btn btn-danger test-access-token-delete"
                            onClick={this.deleteAccessToken}
                            disabled={loading}
                        >
                            Delete
                        </button>
                        {isErrorLike(this.state.deletionOrError) && (
                            <ErrorAlert
                                className="mt-2"
                                error={this.state.deletionOrError}
                                history={this.props.history}
                            />
                        )}
                    </div>
                </div>
                {this.props.newToken && this.props.node.id === this.props.newToken.id && (
                    <AccessTokenCreatedAlert
                        className="alert alert-success mt-4"
                        tokenSecret={this.props.newToken.token}
                        token={this.props.node}
                    />
                )}
            </li>
        )
    }

    private deleteAccessToken = (): void => this.deletes.next()
}
