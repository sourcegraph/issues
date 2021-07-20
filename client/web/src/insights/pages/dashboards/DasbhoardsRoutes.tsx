import React from 'react'
import { Route, RouteComponentProps, Switch, useRouteMatch } from 'react-router'

import { ExtensionsControllerProps } from '@sourcegraph/shared/src/extensions/controller'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'

import { AuthenticatedUser } from '../../../auth'
import { Settings } from '../../../schema/settings.schema'

import { InsightsDashboardCreationPage } from './creation/InsightsDashboardCreationPage'
import { DashboardsPage } from './dashboard-page/DashboardsPage'
import { EditDashboardPage } from './edit-dashboard/EditDashobardPage'

export interface DashboardsRoutesProps
    extends TelemetryProps,
        SettingsCascadeProps<Settings>,
        ExtensionsControllerProps,
        PlatformContextProps<'updateSettings'> {
    authenticatedUser: AuthenticatedUser
}

/**
 * Displays Code Insights dashboard area.
 */
export const DashboardsRoutes: React.FunctionComponent<DashboardsRoutesProps> = props => {
    const { authenticatedUser, settingsCascade, platformContext, telemetryService, extensionsController } = props
    const match = useRouteMatch()

    return (
        <Switch>
            <Route
                path={`${match.url}/dashboards/:dashboardId/edit`}
                render={(routeProps: RouteComponentProps<{ dashboardId: string }>) => (
                    <EditDashboardPage
                        platformContext={platformContext}
                        authenticatedUser={authenticatedUser}
                        settingsCascade={settingsCascade}
                        dashboardId={routeProps.match.params.dashboardId}
                    />
                )}
            />

            <Route
                path={`${match.url}/dashboards/:dashboardId?`}
                render={(routeProps: RouteComponentProps<{ dashboardId: string }>) => (
                    <DashboardsPage
                        platformContext={platformContext}
                        telemetryService={telemetryService}
                        extensionsController={extensionsController}
                        settingsCascade={settingsCascade}
                        dashboardID={routeProps.match.params.dashboardId}
                    />
                )}
            />

            <Route
                path={`${match.url}/add-dashboard`}
                render={() => (
                    <InsightsDashboardCreationPage
                        platformContext={platformContext}
                        telemetryService={telemetryService}
                        authenticatedUser={authenticatedUser}
                        settingsCascade={settingsCascade}
                    />
                )}
            />
        </Switch>
    )
}
