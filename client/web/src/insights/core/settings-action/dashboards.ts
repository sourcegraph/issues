import { camelCase } from 'lodash'

import { isErrorLike } from '@sourcegraph/shared/src/util/errors'
import { modify, parseJSONCOrError } from '@sourcegraph/shared/src/util/jsonc'

import { InsightDashboard, Settings } from '../../../schema/settings.schema'
import { INSIGHTS_DASHBOARDS_SETTINGS_KEY } from '../types'

/**
 * Adds sanitized dashboard configuration to the settings content.
 *
 * @param settings - original subject settings
 * @param dashboardConfiguration - a dashboard configurations
 */
export function addDashboardToSettings(settings: string, dashboardConfiguration: InsightDashboard): string {
    return modify(
        settings,
        [INSIGHTS_DASHBOARDS_SETTINGS_KEY, camelCase(dashboardConfiguration.title)],
        dashboardConfiguration
    )
}

/**
 * Removes dashboard configurations from jsonc settings string
 *
 * @param settings - settings jsonc string
 * @param dashboardId - dashboard id to remove
 */
export function removeDashboardFromSettings(settings: string, dashboardId: string): string {
    return modify(settings, [INSIGHTS_DASHBOARDS_SETTINGS_KEY, dashboardId], undefined)
}

/**
 * Adds insight id in dashboard configuration.
 */
export function addInsightToDashboard(settings: string, dashboardId: string, insightId: string): string {
    const currentDashboard = getDashboard(settings, dashboardId)

    if (!currentDashboard) {
        return settings
    }

    const insightIds = currentDashboard.insightIds ?? []

    return modify(settings, [INSIGHTS_DASHBOARDS_SETTINGS_KEY, dashboardId, 'insightIds'], [...insightIds, insightId])
}

/**
 * Removes insight id from the dashboard configuration insight ids setting.
 */
export function removeInsightFromDashboard(settings: string, dashboardId: string, insightId: string): string {
    const currentDashboard = getDashboard(settings, dashboardId)

    if (!currentDashboard) {
        return settings
    }

    const insightIds = currentDashboard.insightIds ?? []

    return modify(
        settings,
        [INSIGHTS_DASHBOARDS_SETTINGS_KEY, dashboardId, 'insightIds'],
        insightIds.filter(id => id !== insightId)
    )
}

/**
 * Updates dashboard insight ids setting field.
 */
export function updateDashboardInsightIds(settings: string, dashboardId: string, insightIds: string[]): string {
    const currentDashboard = getDashboard(settings, dashboardId)

    if (!currentDashboard) {
        return settings
    }

    return modify(settings, [INSIGHTS_DASHBOARDS_SETTINGS_KEY, dashboardId, 'insightIds'], insightIds)
}

function getDashboard(settings: string, dashboardId: string): InsightDashboard | undefined {
    const parsedSettings = parseJSONCOrError<Settings>(settings)

    if (isErrorLike(parsedSettings)) {
        return
    }

    const dashboards = parsedSettings[INSIGHTS_DASHBOARDS_SETTINGS_KEY] ?? {}

    return dashboards[dashboardId]
}
