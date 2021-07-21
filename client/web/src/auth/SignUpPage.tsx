import React, { useEffect } from 'react'
import { Link, Redirect, useLocation } from 'react-router-dom'

import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { ThemeProps } from '@sourcegraph/shared/src/theme'

import { AuthenticatedUser } from '../auth'
import { HeroPage } from '../components/HeroPage'
import { PageTitle } from '../components/PageTitle'
import { FeatureFlagProps } from '../featureFlags/featureFlags'
import { SourcegraphContext } from '../jscontext'
import { eventLogger } from '../tracking/eventLogger'

import { ExperimentalSignUpPage, ShowEmailFormQueryParameter } from './ExperimentalSignUpPage'
import { SourcegraphIcon } from './icons'
import { getReturnTo } from './SignInSignUpCommon'
import { SignUpArguments, SignUpForm } from './SignUpForm'

interface SignUpPageProps extends ThemeProps, TelemetryProps, FeatureFlagProps {
    authenticatedUser: AuthenticatedUser | null
    context: Pick<
        SourcegraphContext,
        'allowSignup' | 'experimentalFeatures' | 'authProviders' | 'sourcegraphDotComMode' | 'xhrHeaders'
    >
}

export const SignUpPage: React.FunctionComponent<SignUpPageProps> = ({
    authenticatedUser,
    context,
    featureFlags,
    isLightTheme,
    telemetryService,
}) => {
    const location = useLocation()
    // const history = useHistory()
    const query = new URLSearchParams(location.search)

    useEffect(() => {
        eventLogger.logViewEvent('SignUp', null, false)
    }, [])

    if (authenticatedUser) {
        const returnTo = getReturnTo(location)
        return <Redirect to={returnTo} />
    }

    if (!context.allowSignup) {
        return <Redirect to="/sign-in" />
    }

    const handleSignUp = (args: SignUpArguments): Promise<void> =>
        fetch('/-/sign-up', {
            credentials: 'same-origin',
            method: 'POST',
            headers: {
                ...context.xhrHeaders,
                Accept: 'application/json',
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(args),
        }).then(response => {
            if (response.status !== 200) {
                return response.text().then(text => Promise.reject(new Error(text)))
            }

            // if sign up is successful and enablePostSignupFlow feature is ON -
            // redirect user to the /post-sign-up page
            if (context.experimentalFeatures.enablePostSignupFlow) {
                window.location.replace(new URL('/welcome', window.location.href).pathname)
            } else {
                window.location.replace(getReturnTo(location))
            }

            return Promise.resolve()
        })

    if (context.sourcegraphDotComMode && featureFlags.get('w1-signup-optimisation') && query.get('src')) {
        return (
            <ExperimentalSignUpPage
                source={query.get('src')}
                onSignUp={handleSignUp}
                isLightTheme={isLightTheme}
                showEmailForm={query.has(ShowEmailFormQueryParameter)}
                context={context}
                telemetryService={telemetryService}
            />
        )
    }

    return (
        <div className="signin-signup-page sign-up-page">
            <PageTitle title="Sign up" />
            <HeroPage
                icon={SourcegraphIcon}
                iconLinkTo={context.sourcegraphDotComMode ? '/search' : undefined}
                iconClassName="bg-transparent"
                title={
                    context.sourcegraphDotComMode ? 'Sign up for Sourcegraph Cloud' : 'Sign up for Sourcegraph Server'
                }
                lessPadding={true}
                body={
                    <div className="signup-page__container pb-5">
                        {context.sourcegraphDotComMode && <p className="pt-1 pb-2">Start searching public code now</p>}
                        <SignUpForm context={context} onSignUp={handleSignUp} />
                        <p className="mt-3">
                            Already have an account? <Link to={`/sign-in${location.search}`}>Sign in</Link>
                        </p>
                    </div>
                }
            />
        </div>
    )
}
