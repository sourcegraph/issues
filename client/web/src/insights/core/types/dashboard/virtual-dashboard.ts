import { InsightsDashboardType } from './core'

/**
 * Special 'virtual' dashboard that includes all insights from the personal and organization
 * level dashboards. Virtual dashboard doesn't exist in settings but lives only in a runtime.
 */
export interface VirtualInsightsDashboard {
    type: InsightsDashboardType.All
    id: string
    insightIds: string[]
}
