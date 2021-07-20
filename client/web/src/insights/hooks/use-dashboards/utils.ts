import { SettingsSubject } from '@sourcegraph/shared/src/settings/settings'
import { isDefined } from '@sourcegraph/shared/src/util/types'

import { Settings } from '../../../schema/settings.schema'
import {
    INSIGHTS_DASHBOARDS_SETTINGS_KEY,
    InsightsDashboardType,
    InsightDashboard,
    isInsightSettingKey,
    SettingsBasedInsightDashboard,
    InsightDashboardOwner,
    INSIGHTS_ALL_REPOS_SETTINGS_KEY,
} from '../../core/types'
import { isSubjectInsightSupported, SupportedInsightSubject } from '../../core/types/subjects'

/**
 * Returns all insights (extension based and be based) ids.
 */
export function getInsightIdsFromSettings(settings: Settings): string[] {
    // On top level of settings we store extension based insights only
    const extensionBasedInsightIds = Object.keys(settings).filter(isInsightSettingKey)
    // BE insights live in the 'insights.allrepos' map
    const backendBasedInsightIds = Object.keys((settings?.[INSIGHTS_ALL_REPOS_SETTINGS_KEY] as object) ?? {}).filter(
        isInsightSettingKey
    )

    return [...extensionBasedInsightIds, ...backendBasedInsightIds]
}

/**
 * Returns all subject dashboards and one special (built-in) dashboard that includes
 * all insights from subject settings.
 */
export function getSubjectDashboards(subject: SupportedInsightSubject, settings: Settings): InsightDashboard[] {
    const { dashboardType, ...owner } = getDashboardOwnerInfo(subject)

    const subjectBuiltInDashboard: InsightDashboard = {
        owner,
        id: owner.id,
        builtIn: true,
        title: owner.name,
        type: dashboardType,
        insightIds: getInsightIdsFromSettings(settings),
    }

    // Find all subject insights dashboards
    const subjectDashboards = Object.keys(settings[INSIGHTS_DASHBOARDS_SETTINGS_KEY] ?? {})
        .map(dashboardKey => getSubjectDashboardByID(subject, settings, dashboardKey))
        .filter(isDefined)

    return [subjectBuiltInDashboard, ...subjectDashboards]
}

/**
 * Returns settings based dashboard from subject settings by id (key).
 *
 * @param subject - settings subject
 * @param settings - settings map of current subject
 * @param dashboardKey - possible dashboard key (id)
 */
export function getSubjectDashboardByID(
    subject: SettingsSubject,
    settings: Settings,
    dashboardKey: string
): SettingsBasedInsightDashboard | undefined {
    if (!isSubjectInsightSupported(subject)) {
        return undefined
    }

    const { dashboardType, ...owner } = getDashboardOwnerInfo(subject)

    // Select dashboard configuration from the subject settings
    const dashboardSettings = settings[INSIGHTS_DASHBOARDS_SETTINGS_KEY]?.[dashboardKey]

    if (!dashboardSettings) {
        return undefined
    }

    return {
        owner,
        type: dashboardType,
        settingsKey: dashboardKey,
        ...dashboardSettings,
    }
}

interface DashboardOwnerInfo extends InsightDashboardOwner {
    /**
     * Currently we support three types of subject that can have insights dashboard.
     */
    dashboardType: InsightsDashboardType.Personal | InsightsDashboardType.Organization | InsightsDashboardType.Global
}

/**
 * Return dashboard owner info by subject configuration
 *
 * @param subject - subject settings (User, Organization, Site, Client)
 */
export function getDashboardOwnerInfo(subject: SupportedInsightSubject): DashboardOwnerInfo {
    switch (subject.__typename) {
        case 'Site': {
            return {
                id: subject.id,
                name: 'Global',
                dashboardType: InsightsDashboardType.Global,
            }
        }
        case 'Org':
            return {
                id: subject.id,
                name: subject.displayName ?? subject.name,
                dashboardType: InsightsDashboardType.Organization,
            }

        case 'User':
            return {
                id: subject.id,
                name: subject.displayName ?? subject.username,
                dashboardType: InsightsDashboardType.Personal,
            }
    }
}
