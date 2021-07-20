import { useMemo } from 'react'

import { ConfiguredSubjectOrError, SettingsCascadeOrError } from '@sourcegraph/shared/src/settings/settings'
import { isErrorLike, ErrorLike } from '@sourcegraph/shared/src/util/errors'

import { Settings } from '../../../schema/settings.schema'
import { InsightDashboard, InsightsDashboardType } from '../../core/types'
import { isSubjectInsightSupported } from '../../core/types/subjects'

import { getInsightIdsFromSettings, getSubjectDashboards } from './utils'

/**
 * Special virtual dashboard - "All Insights"
 */
export const ALL_INSIGHTS_DASHBOARD: InsightDashboard = {
    id: 'all',
    type: InsightsDashboardType.All,
    insightIds: [],
}

/**
 * React hook that returns all valid and available insights dashboards.
 */
export function useDashboards(settingsCascade: SettingsCascadeOrError): InsightDashboard[] {
    const { subjects, final } = settingsCascade

    return useMemo(() => getInsightsDashboards(subjects, final), [subjects, final])
}

/**
 * Returns all valid and reachable for a user insight-dashboards.
 */
export function getInsightsDashboards(
    subjects: ConfiguredSubjectOrError<Settings>[] | null,
    final: Settings | ErrorLike | null
): InsightDashboard[] {
    if (subjects === null) {
        return []
    }

    const subjectDashboards = subjects.flatMap(configuredSubject => {
        const { settings, subject } = configuredSubject

        if (isErrorLike(settings) || !settings || !isSubjectInsightSupported(subject)) {
            return []
        }

        return getSubjectDashboards(subject, settings)
    })

    const normalizedFinalSettings = !final || isErrorLike(final) ? {} : final

    return [
        {
            ...ALL_INSIGHTS_DASHBOARD,
            // Get all reachable insight ids from the final (merged settings)
            insightIds: getInsightIdsFromSettings(normalizedFinalSettings),
        },
        ...subjectDashboards,
    ]
}
