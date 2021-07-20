import { InsightDashboard, isRealDashboard, SettingsBasedInsightDashboard } from '../../../../../../core/types'
import { isSettingsBasedInsightsDashboard } from '../../../../../../core/types/dashboard/real-dashboard'

/**
 * Only dashboards that are stored in user/org/global settings can be edited.
 *
 * Besides these settings based (configurable dashboards) we have virtual and built-in dashboards
 * they can't be edited in any way (add/remove insights, delete)
 */
export const isDashboardConfigurable = (
    currentDashboard: InsightDashboard | undefined
): currentDashboard is SettingsBasedInsightDashboard =>
    isRealDashboard(currentDashboard) && isSettingsBasedInsightsDashboard(currentDashboard)
