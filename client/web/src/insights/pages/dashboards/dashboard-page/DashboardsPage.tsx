import PlusIcon from 'mdi-react/PlusIcon'
import React from 'react'
import { useRouteMatch } from 'react-router'
import { Redirect } from 'react-router-dom'

import { Link } from '@sourcegraph/shared/src/components/Link'
import { ExtensionsControllerProps } from '@sourcegraph/shared/src/extensions/controller'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { PageHeader } from '@sourcegraph/wildcard'

import { FeedbackBadge } from '../../../../components/FeedbackBadge'
import { Page } from '../../../../components/Page'
import { Settings } from '../../../../schema/settings.schema'
import { CodeInsightsIcon } from '../../../components'
import { InsightsDashboardType } from '../../../core/types'

import { DashboardsContent } from './components/dashboards-content/DashboardsContent'

export interface DashboardsPageProps
    extends PlatformContextProps<'updateSettings'>,
        TelemetryProps,
        SettingsCascadeProps<Settings>,
        ExtensionsControllerProps {
    /**
     * Possible dashboard id. All insights on the page will be get from
     * dashboard's info from the user or org settings by the dashboard id.
     * In case if id is undefined we get insights from the final
     * version of merged settings (all insights)
     */
    dashboardID?: string
}

/**
 * Displays insights dashboard page - dashboard selector and grid of dashboard insights.
 */
export const DashboardsPage: React.FunctionComponent<DashboardsPageProps> = props => {
    const { dashboardID, settingsCascade, extensionsController, telemetryService, platformContext } = props
    const { url } = useRouteMatch()

    if (!dashboardID) {
        // In case if url doesn't have a dashboard id we should fallback on
        // built-in "All insights" dashboard
        return <Redirect to={`${url}/${InsightsDashboardType.All}`} />
    }

    return (
        <div className="w-100">
            <Page>
                <PageHeader
                    annotation={<FeedbackBadge status="prototype" feedback={{ mailto: 'support@sourcegraph.com' }} />}
                    path={[{ icon: CodeInsightsIcon, text: 'Insights' }]}
                    actions={
                        <>
                            <Link to="/insights/add-dashboard" className="btn btn-outline-secondary mr-2">
                                <PlusIcon className="icon-inline" /> Create new dashboard
                            </Link>
                            <Link to="/insights/create" className="btn btn-secondary">
                                <PlusIcon className="icon-inline" /> Create new insight
                            </Link>
                        </>
                    }
                    className="mb-3"
                />

                <DashboardsContent
                    platformContext={platformContext}
                    extensionsController={extensionsController}
                    telemetryService={telemetryService}
                    settingsCascade={settingsCascade}
                    dashboardID={dashboardID}
                />
            </Page>
        </div>
    )
}
