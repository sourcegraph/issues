import * as H from 'history'
import * as React from 'react'
import { Link, Redirect } from 'react-router-dom'
import { HeroPage } from '../components/HeroPage'
import { PageTitle } from '../components/PageTitle'
import { eventLogger } from '../tracking/eventLogger'
import { getReturnTo } from './SignInSignUpCommon'
import { SignUpArguments, SignUpForm } from './SignUpForm'
import { AuthenticatedUser } from '../auth'
import { SourcegraphIcon } from './icons'
import { SourcegraphContext } from '../jscontext'

interface SignUpPageProps {
    location: H.Location
    history: H.History
    authenticatedUser: AuthenticatedUser | null
    context: Pick<SourcegraphContext, 'allowSignup' | 'authProviders' | 'sourcegraphDotComMode' | 'xhrHeaders'>
}

export class SignUpPage extends React.Component<SignUpPageProps> {
    public componentDidMount(): void {
        eventLogger.logViewEvent('SignUp', null, false)
    }

    public render(): JSX.Element | null {
        if (this.props.authenticatedUser) {
            const returnTo = getReturnTo(this.props.location)
            return <Redirect to={returnTo} />
        }

        if (!this.props.context.allowSignup) {
            return <Redirect to="/sign-in" />
        }

        return (
            <div className="signin-signup-page sign-up-page web-content">
                <PageTitle title="Sign up" />
                <HeroPage
                    icon={SourcegraphIcon}
                    iconLinkTo={this.props.context.sourcegraphDotComMode ? '/search' : undefined}
                    iconClassName="bg-transparent"
                    title={
                        this.props.context.sourcegraphDotComMode
                            ? 'Sign up for Sourcegraph Cloud'
                            : 'Sign up for Sourcegraph Server'
                    }
                    lessPadding={true}
                    body={
                        <div className="signup-page__container pb-5">
                            {this.props.context.sourcegraphDotComMode && (
                                <p className="pt-1 pb-2">Start searching public code now</p>
                            )}
                            <SignUpForm {...this.props} doSignUp={this.doSignUp} />
                            <p className="mt-3">
                                Already have an account?{' '}
                                <Link to={`/sign-in${this.props.location.search}`}>Sign in</Link>
                            </p>
                        </div>
                    }
                />
            </div>
        )
    }

    private doSignUp = (args: SignUpArguments): Promise<void> =>
        fetch('/-/sign-up', {
            credentials: 'same-origin',
            method: 'POST',
            headers: {
                ...this.props.context.xhrHeaders,
                Accept: 'application/json',
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(args),
        }).then(response => {
            if (response.status !== 200) {
                return response.text().then(text => Promise.reject(new Error(text)))
            }
            window.location.replace(getReturnTo(this.props.location))
            return Promise.resolve()
        })
}
