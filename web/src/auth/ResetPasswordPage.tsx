import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import KeyIcon from 'mdi-react/KeyIcon'
import * as React from 'react'
import { Link, RouteComponentProps } from 'react-router-dom'
import { asError, ErrorLike, isErrorLike } from '../../../shared/src/util/errors'
import { Form } from '../components/Form'
import { HeroPage } from '../components/HeroPage'
import { PageTitle } from '../components/PageTitle'
import { eventLogger } from '../tracking/eventLogger'
import { PasswordInput } from './SignInSignUpCommon'
import { ErrorAlert } from '../components/alerts'
import * as H from 'history'
import { OptionalAuthProps } from '../auth'

interface ResetPasswordInitFormState {
    /** The user's email input value. */
    email: string

    /**
     * The state of the form submission. If undefined, the form has not been submitted. If null, the form was
     * submitted successfully.
     */
    submitOrError: undefined | 'loading' | ErrorLike | null
}

interface ResetPasswordInitFormProps {
    history: H.History
}

/**
 * A form where the user can initiate the reset-password flow. This is the 1st step in the
 * reset-password flow; ResetPasswordCodePage is the 2nd step.
 */
class ResetPasswordInitForm extends React.PureComponent<ResetPasswordInitFormProps, ResetPasswordInitFormState> {
    public state: ResetPasswordInitFormState = {
        email: '',
        submitOrError: undefined,
    }

    public render(): JSX.Element | null {
        if (this.state.submitOrError === null) {
            return <div className="alert alert-success">Password reset email sent.</div>
        }

        return (
            <Form className="reset-password-page__form" onSubmit={this.handleSubmitResetPasswordInit}>
                <p>
                    Enter the email address associated with your account. You'll receive an email with a link to reset
                    your password.
                </p>
                <div className="form-group">
                    <input
                        className="form-control"
                        onChange={this.onEmailFieldChange}
                        value={this.state.email}
                        type="email"
                        name="email"
                        autoFocus={true}
                        spellCheck={false}
                        placeholder="Email"
                        required={true}
                        autoComplete="email"
                        disabled={this.state.submitOrError === 'loading'}
                    />
                </div>
                <button
                    className="btn btn-primary btn-block"
                    type="submit"
                    disabled={this.state.submitOrError === 'loading'}
                >
                    Send reset password link
                </button>
                {this.state.submitOrError === 'loading' && <LoadingSpinner className="icon-inline mt-2" />}
                {isErrorLike(this.state.submitOrError) && (
                    <ErrorAlert className="mt-2" error={this.state.submitOrError} history={this.props.history} />
                )}
            </Form>
        )
    }

    private onEmailFieldChange = (event: React.ChangeEvent<HTMLInputElement>): void => {
        this.setState({ email: event.target.value })
    }

    private handleSubmitResetPasswordInit = (event: React.FormEvent<HTMLFormElement>): void => {
        event.preventDefault()
        this.setState({ submitOrError: 'loading' })
        fetch('/-/reset-password-init', {
            credentials: 'same-origin',
            method: 'POST',
            headers: {
                ...window.context.xhrHeaders,
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                email: this.state.email,
            }),
        })
            .then(response => {
                if (response.status === 200) {
                    this.setState({ submitOrError: null })
                } else if (response.status === 429) {
                    this.setState({
                        submitOrError: new Error('Too many password reset requests. Try again in a few minutes.'),
                    })
                } else {
                    response
                        .text()
                        .catch(() => null)
                        .then(text => this.setState({ submitOrError: new Error(text || 'Unknown error') }))
                        .catch(error => console.error(error))
                }
            })
            .catch(error => this.setState({ submitOrError: asError(error) }))
    }
}

interface ResetPasswordCodeFormProps {
    userID: number
    code: string
    history: H.History
}

interface ResetPasswordCodeFormState {
    /** The user's new password input value. */
    password: string

    /**
     * The state of the form submission. If undefined, the form has not been submitted. If null, the form was
     * submitted successfully.
     */
    submitOrError: undefined | 'loading' | ErrorLike | null
}

class ResetPasswordCodeForm extends React.PureComponent<ResetPasswordCodeFormProps, ResetPasswordCodeFormState> {
    public state: ResetPasswordCodeFormState = {
        password: '',
        submitOrError: undefined,
    }

    public render(): JSX.Element | null {
        if (this.state.submitOrError === null) {
            return (
                <div className="alert alert-success">
                    Your password was reset. <Link to="/sign-in">Sign in with your new password</Link> to continue.
                </div>
            )
        }

        return (
            <Form className="reset-password-page__form" onSubmit={this.handleSubmitResetPassword}>
                <p>Enter a new password for your account.</p>
                <div className="form-group">
                    <PasswordInput
                        name="password"
                        onChange={this.onPasswordFieldChange}
                        value={this.state.password}
                        required={true}
                        autoFocus={true}
                        autoComplete="new-password"
                        disabled={this.state.submitOrError === 'loading'}
                    />
                </div>
                <button
                    className="btn btn-primary btn-block"
                    type="submit"
                    disabled={this.state.submitOrError === 'loading'}
                >
                    Reset password
                </button>
                {this.state.submitOrError === 'loading' && <LoadingSpinner className="icon-inline mt-2" />}
                {isErrorLike(this.state.submitOrError) && (
                    <ErrorAlert className="mt-2" error={this.state.submitOrError} history={this.props.history} />
                )}
            </Form>
        )
    }

    private onPasswordFieldChange = (event: React.ChangeEvent<HTMLInputElement>): void => {
        this.setState({ password: event.target.value })
    }

    private handleSubmitResetPassword = (event: React.FormEvent<HTMLFormElement>): void => {
        event.preventDefault()
        this.setState({ submitOrError: 'loading' })
        fetch('/-/reset-password-code', {
            credentials: 'same-origin',
            method: 'POST',
            headers: {
                ...window.context.xhrHeaders,
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                userID: this.props.userID,
                code: this.props.code,
                password: this.state.password,
            }),
        })
            .then(response => {
                if (response.status === 200) {
                    this.setState({ submitOrError: null })
                } else if (response.status === 401) {
                    this.setState({ submitOrError: new Error('Password reset code was invalid or expired.') })
                } else {
                    this.setState({ submitOrError: new Error('Password reset failed.') })
                }
            })
            .catch(error => this.setState({ submitOrError: asError(error) }))
    }
}

interface ResetPasswordPageProps extends RouteComponentProps<{}>, OptionalAuthProps {}

/**
 * A page that implements the reset-password flow for a user: (1) initiate the flow by providing the email address
 * of the account whose password to reset, and (2) complete the flow by providing the password-reset code.
 */
export class ResetPasswordPage extends React.PureComponent<ResetPasswordPageProps> {
    public componentDidMount(): void {
        eventLogger.logViewEvent('ResetPassword', false)
    }

    public render(): JSX.Element | null {
        let body: JSX.Element
        if (this.props.authenticatedUser) {
            body = <div className="alert alert-danger">Authenticated users may not perform password reset.</div>
        } else if (window.context.resetPasswordEnabled) {
            const searchParameters = new URLSearchParams(this.props.location.search)
            if (searchParameters.has('code') || searchParameters.has('userID')) {
                const code = searchParameters.get('code')
                const userID = parseInt(searchParameters.get('userID') || '', 10)
                if (code && !isNaN(userID)) {
                    body = <ResetPasswordCodeForm code={code} userID={userID} history={this.props.history} />
                } else {
                    body = <div className="alert alert-danger">The password reset link you followed is invalid.</div>
                }
            } else {
                body = <ResetPasswordInitForm history={this.props.history} />
            }
        } else {
            body = (
                <div className="alert alert-warning">
                    Password reset is disabled. Ask a site administrator to manually reset your password.
                </div>
            )
        }

        return (
            <>
                <PageTitle title="Reset password" />
                <HeroPage icon={KeyIcon} title="Reset password" cta={body} />
            </>
        )
    }
}
