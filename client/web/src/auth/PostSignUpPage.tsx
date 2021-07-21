import React, { FunctionComponent, useState, useEffect, useCallback, useRef } from 'react'
import { useLocation, useHistory } from 'react-router'

import { LinkOrSpan } from '@sourcegraph/shared/src/components/LinkOrSpan'
import { TelemetryService } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { useLocalStorage } from '@sourcegraph/shared/src/util/useLocalStorage'
import { BrandLogo } from '@sourcegraph/web/src/components/branding/BrandLogo'
import { HeroPage } from '@sourcegraph/web/src/components/HeroPage'
import { Steps, Step, StepList, StepPanels, StepPanel, StepActions } from '@sourcegraph/wildcard/src/components/Steps'

import { AuthenticatedUser } from '../auth'
import { PageTitle } from '../components/PageTitle'
import { SourcegraphContext } from '../jscontext'
import { SelectAffiliatedRepos } from '../user/settings/repositories/SelectAffiliatedRepos'

import { getReturnTo } from './SignInSignUpCommon'
import { useExternalServices } from './useExternalServices'
import { CodeHostsConnection } from './welcome/CodeHostsConnection'
import { Footer } from './welcome/Footer'
import { StartSearching } from './welcome/StartSearching'

interface PostSignUpPage {
    authenticatedUser: AuthenticatedUser
    context: Pick<SourcegraphContext, 'authProviders'>
    telemetryService: TelemetryService
}

interface Step {
    content: React.ReactElement
    isComplete: () => boolean
    prefetch?: () => void
    onNextButtonClick?: () => Promise<void>
}

// type PerformanceNavigationTimingType = 'navigate' | 'reload' | 'back_forward' | 'prerender'

export type RepoSelectionMode = 'all' | 'selected' | undefined

const USER_FINISHED_WELCOME_FLOW = 'finished-welcome-flow'

export const PostSignUpPage: FunctionComponent<PostSignUpPage> = ({
    authenticatedUser: user,
    context,
    telemetryService,
}) => {
    const [didUserFinishWelcomeFlow, setUserFinishedWelcomeFlow] = useLocalStorage(USER_FINISHED_WELCOME_FLOW, false)
    const isOAuthCall = useRef(false)
    const location = useLocation()
    const history = useHistory()

    const goToSearch = (): void => history.push(getReturnTo(location))

    // if the welcome flow was already finished - navigate to search
    if (didUserFinishWelcomeFlow) {
        goToSearch()
    }

    const finishWelcomeFlow = (): void => {
        setUserFinishedWelcomeFlow(true)
        goToSearch()
    }

    const [repoSelectionMode, setRepoSelectionMode] = useState<RepoSelectionMode>()
    const { externalServices, loadingServices, errorServices, refetchExternalServices } = useExternalServices(user.id)

    const beforeUnload = useCallback((): void => {
        // user is not leaving the flow, it's an OAuth page refresh
        if (isOAuthCall.current) {
            return
        }

        // TODO: discuss
        // allow user to manually refresh the page
        // if (window.performance?.getEntriesByType) {
        //     const entries = window.performance?.getEntriesByType('navigation')
        //     // let TS know that we may expect PerformanceNavigationTiming.type
        //     const lastEntry = entries.pop() as PerformanceEntry & { type?: PerformanceNavigationTimingType }
        //     if (lastEntry?.type === 'reload') {
        //         return
        //     }
        // }

        setUserFinishedWelcomeFlow(true)
    }, [setUserFinishedWelcomeFlow])

    useEffect(() => {
        window.addEventListener('beforeunload', beforeUnload)

        return () => window.removeEventListener('beforeunload', beforeUnload)
    }, [beforeUnload])

    return (
        <>
            <LinkOrSpan to={getReturnTo(location)} className="post-signup-page__logo-link">
                <BrandLogo
                    className="position-absolute ml-3 mt-3 post-signup-page__logo"
                    isLightTheme={true}
                    variant="symbol"
                    onClick={finishWelcomeFlow}
                />
            </LinkOrSpan>

            <div className="signin-signup-page post-signup-page">
                <PageTitle title="Welcome" />
                <HeroPage
                    lessPadding={true}
                    className="text-left"
                    body={
                        <div className="post-signup-page__container">
                            <h2>Get started with Sourcegraph</h2>
                            <p className="text-muted pb-3">
                                Three quick steps to add your repositories and get searching with Sourcegraph
                            </p>
                            <div className="mt-4 pb-3">
                                <Steps initialStep={1}>
                                    <StepList numeric={true}>
                                        <Step borderColor="purple">Connect with code hosts</Step>
                                        <Step borderColor="blue">Add repositories</Step>
                                        <Step borderColor="orange">Start searching</Step>
                                    </StepList>
                                    <StepPanels>
                                        <StepPanel>
                                            {externalServices && (
                                                <CodeHostsConnection
                                                    user={user}
                                                    onNavigation={(called: boolean) => {
                                                        isOAuthCall.current = called
                                                    }}
                                                    loading={loadingServices}
                                                    error={errorServices}
                                                    externalServices={externalServices}
                                                    context={context}
                                                    refetch={refetchExternalServices}
                                                />
                                            )}
                                        </StepPanel>
                                        <StepPanel>
                                            <>
                                                <h3>Add repositories</h3>
                                                <p className="text-muted">
                                                    Choose repositories you own or collaborate on from your code hosts
                                                    to search with Sourcegraph. We’ll sync and index these repositories
                                                    so you can search your code all in one place.
                                                </p>
                                                <SelectAffiliatedRepos
                                                    authenticatedUser={user}
                                                    onRepoSelectionModeChange={setRepoSelectionMode}
                                                    telemetryService={telemetryService}
                                                />
                                            </>
                                        </StepPanel>
                                        <StepPanel>
                                            <StartSearching user={user} repoSelectionMode={repoSelectionMode} />
                                        </StepPanel>
                                    </StepPanels>
                                    <StepActions>
                                        <Footer onFinish={finishWelcomeFlow} />
                                    </StepActions>
                                </Steps>
                            </div>
                        </div>
                    }
                />
            </div>
        </>
    )
}
