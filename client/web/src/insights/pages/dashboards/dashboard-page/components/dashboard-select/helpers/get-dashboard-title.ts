import { InsightsDashboardType, RealInsightDashboard } from '../../../../../../core/types'

/**
 * Get formatted dashboard title for the dashboard select option.
 */
export const getDashboardTitle = (dashboard: RealInsightDashboard): string => {
    const { builtIn } = dashboard

    if (builtIn) {
        return `${dashboard.owner.name} Insights`
    }

    return dashboard.title
}

/**
 * Get formatted dashboard owner name. Used for list option badge element.
 */
export const getDashboardOwnerName = (dashboard: RealInsightDashboard): string => {
    const { type } = dashboard

    if (type === InsightsDashboardType.Personal) {
        return 'Private'
    }

    if (type === InsightsDashboardType.Global) {
        return 'Global'
    }

    return dashboard.owner.name
}
