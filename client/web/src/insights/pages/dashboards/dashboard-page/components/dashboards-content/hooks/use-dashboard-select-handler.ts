import { useHistory } from 'react-router-dom'

import { InsightDashboard, isVirtualDashboard } from '../../../../../../core/types'
import { isSettingsBasedInsightsDashboard } from '../../../../../../core/types/dashboard/real-dashboard'

type SelectHandler = (dashboard: InsightDashboard) => void

/**
 * Hook for managing URL of the dashboard page whenever the user picks
 * another dashboard via dashboard select
 */
export function useDashboardSelectHandler(): SelectHandler {
    const history = useHistory()

    return (dashboard: InsightDashboard): void => {
        if (isVirtualDashboard(dashboard)) {
            history.push(`/insights/dashboards/${dashboard.type}`)

            return
        }

        if (isSettingsBasedInsightsDashboard(dashboard)) {
            history.push(`/insights/dashboards/${dashboard.settingsKey}`)

            return
        }

        history.push(`/insights/dashboards/${dashboard.id}`)
    }
}
