import classnames from 'classnames'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React, { useRef, useState } from 'react'
import { useHistory } from 'react-router-dom'

import { ExtensionsControllerProps } from '@sourcegraph/shared/src/extensions/controller'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'

import { HeroPage } from '../../../../../../components/HeroPage'
import { Settings } from '../../../../../../schema/settings.schema'
import { isVirtualDashboard } from '../../../../../core/types'
import { isSettingsBasedInsightsDashboard } from '../../../../../core/types/dashboard/real-dashboard'
import { useDashboards } from '../../../../../hooks/use-dashboards/use-dashboards'
import { AddInsightModal } from '../add-insight-modal/AddInsightModal'
import { DashboardMenu, DashboardMenuAction } from '../dashboard-menu/DashboardMenu'
import { DashboardSelect } from '../dashboard-select/DashboardSelect'
import { DeleteDashboardModal } from '../delete-dashboard-modal/DeleteDashboardModal'

import { DashboardInsights } from './components/dashboard-inisghts/DashboardInsights'
import styles from './DashboardsContent.module.scss'
import { useCopyURLHandler } from './hooks/use-copy-url-handler'
import { useDashboardSelectHandler } from './hooks/use-dashboard-select-handler'
import { findDashboardByUrlId } from './utils/find-dashboard-by-url-id'
import { isDashboardConfigurable } from './utils/is-dashboard-configurable'

export interface DashboardsContentProps
    extends SettingsCascadeProps<Settings>,
        ExtensionsControllerProps,
        TelemetryProps,
        PlatformContextProps<'updateSettings'> {
    /**
     * Possible dashboard id. All insights on the page will be get from
     * dashboard's info from the user or org settings by the dashboard id.
     * In case if id is undefined we get insights from the final
     * version of merged settings (all insights)
     */
    dashboardID: string
}

export const DashboardsContent: React.FunctionComponent<DashboardsContentProps> = props => {
    const { extensionsController, settingsCascade, dashboardID, telemetryService, platformContext } = props

    const history = useHistory()
    const dashboards = useDashboards(settingsCascade)

    // State to open/close add/remove insights modal UI
    const [isAddInsightOpen, setAddInsightsState] = useState<boolean>(false)
    const [isDeleteDashboardActive, setDeleteDashboardActive] = useState<boolean>(false)

    const currentDashboard = findDashboardByUrlId(dashboards, dashboardID)
    const handleDashboardSelect = useDashboardSelectHandler()
    const [copyURL, isCopied] = useCopyURLHandler()
    const menuReference = useRef<HTMLButtonElement | null>(null)

    const handleSelect = (action: DashboardMenuAction): void => {
        switch (action) {
            case DashboardMenuAction.Configure: {
                if (!isVirtualDashboard(currentDashboard) && isSettingsBasedInsightsDashboard(currentDashboard)) {
                    history.push(`/insights/dashboards/${currentDashboard.settingsKey}/edit`)
                }
                return
            }
            case DashboardMenuAction.AddRemoveInsights: {
                setAddInsightsState(true)
                return
            }
            case DashboardMenuAction.Delete: {
                setDeleteDashboardActive(true)
                return
            }
            case DashboardMenuAction.CopyLink: {
                copyURL()

                // Re-trigger trigger tooltip event catching logic to activate
                // copied tooltip appearance
                requestAnimationFrame(() => {
                    menuReference.current?.blur()
                    menuReference.current?.focus()
                })

                return
            }
        }
    }

    const handleAddInsightRequest = (): void => {
        setAddInsightsState(true)
    }

    return (
        <div>
            <section className="d-flex flex-wrap align-items-center">
                <span className={styles.dashboardSelectLabel}>Dashboard</span>

                <DashboardSelect
                    value={currentDashboard?.id}
                    dashboards={dashboards}
                    onSelect={handleDashboardSelect}
                    className={classnames(styles.dashboardSelect, 'mr-2')}
                />

                <DashboardMenu
                    innerRef={menuReference}
                    tooltipText={isCopied ? 'Copied!' : undefined}
                    dashboard={currentDashboard}
                    settingsCascade={settingsCascade}
                    onSelect={handleSelect}
                />
            </section>

            <hr className="mt-2 mb-3" />

            {currentDashboard ? (
                <DashboardInsights
                    dashboard={currentDashboard}
                    extensionsController={extensionsController}
                    telemetryService={telemetryService}
                    platformContext={platformContext}
                    settingsCascade={settingsCascade}
                    onAddInsightRequest={handleAddInsightRequest}
                />
            ) : (
                <HeroPage icon={MapSearchIcon} title="Hmm, the dashboard wasn't found." />
            )}

            {isAddInsightOpen && isDashboardConfigurable(currentDashboard) && (
                <AddInsightModal
                    platformContext={platformContext}
                    settingsCascade={settingsCascade}
                    dashboard={currentDashboard}
                    onClose={() => setAddInsightsState(false)}
                />
            )}

            {isDeleteDashboardActive && isDashboardConfigurable(currentDashboard) && (
                <DeleteDashboardModal
                    dashboard={currentDashboard}
                    platformContext={platformContext}
                    onClose={() => setDeleteDashboardActive(false)}
                />
            )}
        </div>
    )
}
