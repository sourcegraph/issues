import React from 'react'
import { Switch, Route, useRouteMatch } from 'react-router'

import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'

import { AuthenticatedUser } from '../../../../auth'
import { lazyComponent } from '../../../../util/lazyComponent'

const IntroCreationLazyPage = lazyComponent(() => import('./intro/IntroCreationPage'), 'IntroCreationPage')
const SearchInsightCreationLazyPage = lazyComponent(
    () => import('./search-insight/SearchInsightCreationPage'),
    'SearchInsightCreationPage'
)

const LangStatsInsightCreationLazyPage = lazyComponent(
    () => import('./lang-stats/LangStatsInsightCreationPage'),
    'LangStatsInsightCreationPage'
)

interface CreationRoutesProps extends TelemetryProps, PlatformContextProps<'updateSettings'>, SettingsCascadeProps {
    /**
     * Authenticated user info, Used to decide where code insight will appears
     * in personal dashboard (private) or in organisation dashboard (public)
     * */
    authenticatedUser: AuthenticatedUser
}

/**
 * Code insight sub-router for the creation area/routes.
 * Renders code insights creation routes (insight creation UI pages, creation intro page)
 */
export const CreationRoutes: React.FunctionComponent<CreationRoutesProps> = props => {
    const { telemetryService, platformContext, settingsCascade } = props

    const match = useRouteMatch()

    return (
        <Switch>
            <Route
                exact={true}
                path={`${match.url}`}
                render={() => <IntroCreationLazyPage telemetryService={telemetryService} />}
            />

            <Route
                path={`${match.url}/search`}
                render={() => (
                    <SearchInsightCreationLazyPage
                        telemetryService={telemetryService}
                        platformContext={platformContext}
                        settingsCascade={settingsCascade}
                    />
                )}
            />

            <Route
                path={`${match.url}/lang-stats`}
                exact={true}
                render={() => (
                    <LangStatsInsightCreationLazyPage
                        telemetryService={telemetryService}
                        platformContext={platformContext}
                        settingsCascade={settingsCascade}
                    />
                )}
            />
        </Switch>
    )
}
