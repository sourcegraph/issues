import { InsightDashboard as InsightDashboardConfiguration } from '../../../../schema/settings.schema'

/**
 * All insights dashboards are separated on three categories.
 */
export enum InsightsDashboardType {
    /**
     * This type of dashboard includes all insights from the personal and organization
     * settings.
     */
    All = 'all',

    /**
     * This type of dashboard includes insights from the personal settings or from
     * dashboards that are stored in the personal settings (personal dashboard)
     */
    Personal = 'personal',

    /**
     * This type of dashboard includes insights from the organization settings or from
     * dashboards that are stored in the organization settings (org dashboard)
     */
    Organization = 'organization',

    /**
     * This type of dashboard includes all insights from the site (global settings) or
     * from dashboards that are stored in global (site) settings subject.
     */
    Global = 'global',
}

/**
 * Information about dashboard owner. It can be a user-type subject (personal dashboard), org subject
 * (org level dashboard)
 */
export interface InsightDashboardOwner {
    id: string
    name: string
}

export interface ExtendedInsightDashboard extends InsightDashboardConfiguration {
    /**
     * All dashboards that were created in users or org settings explicitly are
     * custom dashboards.
     */
    type: InsightsDashboardType.Personal | InsightsDashboardType.Organization | InsightsDashboardType.Global

    /**
     * Subject that has a particular dashboard, it can be personal setting
     * or organization setting subject.
     */
    owner: InsightDashboardOwner
}
