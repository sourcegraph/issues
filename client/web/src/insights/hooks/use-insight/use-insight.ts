import { useMemo } from 'react'

import { SettingsCascadeOrError, SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { isErrorLike } from '@sourcegraph/shared/src/util/errors'

import { Settings } from '../../../schema/settings.schema'
import {
    Insight,
    InsightConfiguration,
    INSIGHTS_ALL_REPOS_SETTINGS_KEY,
    InsightType,
    SearchBasedInsightSettings,
} from '../../core/types'

export interface UseInsightProps extends SettingsCascadeProps<Settings> {
    insightId: string
}

/**
 * Returns insight from the setting cascade.
 */
export function useInsight(props: UseInsightProps): Insight | null {
    const { settingsCascade, insightId } = props

    return useMemo(() => findInsightById(settingsCascade, insightId), [settingsCascade, insightId])
}

export function findInsightById(settingsCascade: SettingsCascadeOrError<Settings>, insightId: string): Insight | null {
    const subjects = settingsCascade.subjects

    const subject = subjects?.find(
        ({ settings }) =>
            settings &&
            !isErrorLike(settings) &&
            (settings[insightId] ||
                // Also check insights all repos map as a second place if insights store
                (settings[INSIGHTS_ALL_REPOS_SETTINGS_KEY] as Record<string, Insight>)?.[insightId])
    )

    if (!subject?.settings || isErrorLike(subject.settings)) {
        return null
    }

    // Top level match means we are dealing with extension based insights
    if (subject.settings[insightId]) {
        const insightConfiguration = subject.settings[insightId] as InsightConfiguration

        return {
            id: insightId,
            visibility: subject.subject.id,
            type: InsightType.Extension,
            ...insightConfiguration,
        }
    }

    const allReposInsights =
        (subject.settings[INSIGHTS_ALL_REPOS_SETTINGS_KEY] as Record<string, SearchBasedInsightSettings>) ?? {}

    // Match in all repos object means that we are dealing with backend search based insight.
    if (allReposInsights[insightId]) {
        const insightConfiguration = allReposInsights[insightId]

        return {
            id: insightId,
            visibility: subject.subject.id,
            type: InsightType.Backend,
            ...insightConfiguration,
        }
    }

    return null
}

interface InsightsInputs {
    insightKey: string
    insightConfiguration: InsightConfiguration
    ownerId: string
}

export function createExtensionInsightFromSettings(input: InsightsInputs): Insight {
    const { insightKey, ownerId, insightConfiguration } = input

    return {
        id: insightKey,
        type: InsightType.Extension,
        visibility: ownerId,
        ...insightConfiguration,
    }
}
