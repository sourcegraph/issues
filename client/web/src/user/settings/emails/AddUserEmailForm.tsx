import * as React from 'react'
import { merge, Observable, of, Subject, Subscription } from 'rxjs'
import { catchError, map, switchMap, tap } from 'rxjs/operators'
import { gql } from '../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { createAggregateError, ErrorLike } from '../../../../../shared/src/util/errors'
import { mutateGraphQL } from '../../../backend/graphql'
import { Form } from '../../../../../branded/src/components/Form'
import { eventLogger } from '../../../tracking/eventLogger'
import { ErrorAlert } from '../../../components/alerts'
import * as H from 'history'

interface Props {
    /** The GraphQL ID of the user with whom the new emails are associated. */
    user: GQL.ID

    /** Called after successfully adding an email to the user. */
    onDidAdd: () => void

    className?: string
    history: H.History
}

interface State {
    email: string
    error?: ErrorLike | null
}

export class AddUserEmailForm extends React.PureComponent<Props, State> {
    public state: State = { email: '', error: null }

    private submits = new Subject<React.FormEvent<HTMLFormElement>>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        this.subscriptions.add(
            this.submits
                .pipe(
                    tap(event => event.preventDefault()),
                    switchMap(() =>
                        merge(
                            of<Pick<State, 'error'>>({ error: undefined }),
                            this.addUserEmail(this.state.email).pipe(
                                tap(() => this.props.onDidAdd()),
                                map(() => ({ error: null, email: '' })),
                                catchError(error => [{ error, email: this.state.email }])
                            )
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
        const loading = this.state.error === undefined
        return (
            <div className={`add-user-email-form ${this.props.className || ''}`}>
                <h3>Add email address</h3>
                <Form className="form-inline" onSubmit={this.onSubmit}>
                    <label className="sr-only" htmlFor="AddUserEmailForm-email">
                        Email address
                    </label>
                    <input
                        type="email"
                        name="email"
                        className="form-control mr-sm-2 test-user-email-add-input"
                        id="AddUserEmailForm-email"
                        onChange={this.onChange}
                        size={32}
                        value={this.state.email}
                        required={true}
                        autoCorrect="off"
                        spellCheck={false}
                        autoCapitalize="off"
                        readOnly={loading}
                        placeholder="Email"
                    />{' '}
                    <button type="submit" className="btn btn-primary" disabled={loading}>
                        {loading ? 'Adding...' : 'Add'}
                    </button>
                </Form>
                {this.state.error && (
                    <ErrorAlert className="mt-2" error={this.state.error} history={this.props.history} />
                )}
            </div>
        )
    }

    private onChange: React.ChangeEventHandler<HTMLInputElement> = event =>
        this.setState({ email: event.currentTarget.value })
    private onSubmit: React.FormEventHandler<HTMLFormElement> = event => this.submits.next(event)

    private addUserEmail = (email: string): Observable<void> =>
        mutateGraphQL(
            gql`
                mutation AddUserEmail($user: ID!, $email: String!) {
                    addUserEmail(user: $user, email: $email) {
                        alwaysNil
                    }
                }
            `,
            { user: this.props.user, email }
        ).pipe(
            map(({ data, errors }) => {
                if (!data || (errors && errors.length > 0)) {
                    throw createAggregateError(errors)
                }
                eventLogger.log('NewUserEmailAddressAdded')
            })
        )
}
