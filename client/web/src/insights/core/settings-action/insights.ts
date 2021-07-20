import { get } from 'lodash'

import { isErrorLike } from '@sourcegraph/shared/src/util/errors'
import { modify, parseJSONCOrError } from '@sourcegraph/shared/src/util/jsonc'

import {
    Insight,
    INSIGHTS_ALL_REPOS_SETTINGS_KEY,
    InsightType,
    InsightTypePrefix,
    isLangStatsInsight,
    isSearchBasedInsight,
} from '../types'

/**
 * Returns insights extension name based on insight id.
 */
const getExtensionNameByInsight = (insight: Insight): string | undefined => {
    if (isSearchBasedInsight(insight)) {
        return 'sourcegraph/search-insights'
    }

    if (isLangStatsInsight(insight)) {
        return 'sourcegraph/code-stats-insights'
    }

    return undefined
}

/**
 * Returns insight settings key. Since different types of insight live in different
 * places in the settings we have to derive this place (settings key) by insight types
 */
const getInsightSettingKey = (insight: Insight): string[] => {
    // Lang stats insight always lives on top level by its unique id
    if (isLangStatsInsight(insight)) {
        return [insight.id]
    }

    // Search based insight may live in two main places
    switch (insight.type) {
        // Extension based lives on top level of settings file by its id
        case InsightType.Extension: {
            return [insight.id]
        }

        // Backend based insight lives in insights.allrepos map
        case InsightType.Backend: {
            return [INSIGHTS_ALL_REPOS_SETTINGS_KEY, insight.id]
        }
    }
}

/**
 * Serializes and adds insight configurations to the settings content string (jsonc).
 * Returns settings content string with serialized insight inside.
 *
 * @param settings - original settings content string
 * @param insight - insight configuration to add in settings file
 */
export const addInsightToSettings = (settings: string, insight: Insight): string => {
    // remove all synthetic properties from the insight object
    const { id, visibility, type, ...originalInsight } = insight

    const extensionName = getExtensionNameByInsight(insight)

    if (!extensionName) {
        return settings
    }

    // Turn on extension if user in creation code insight.
    const settingsWithExtension = modify(settings, ['extensions', extensionName], true)
    const insightSettingsKey = getInsightSettingKey(insight)

    // Add insight to the user settings
    return modify(settingsWithExtension, insightSettingsKey, originalInsight)
}

interface RemoveInsightFromSettingsInputs {
    originalSettings: string
    insightID: string
    isOldCodeStatsInsight?: boolean
}

/**
 * Return edited settings without deleted insight.
 */
export const removeInsightFromSettings = (props: RemoveInsightFromSettingsInputs): string => {
    const {
        originalSettings,
        insightID,
        // For backward compatibility with old code stats insight api we have to delete
        // this insight in a special way. See link below for more information.
        // https://github.com/sourcegraph/sourcegraph-code-stats-insights/blob/master/src/code-stats-insights.ts#L33
        isOldCodeStatsInsight = insightID === `${InsightTypePrefix.langStats}.language`,
    } = props

    if (isOldCodeStatsInsight) {
        const editedSettings = modify(
            originalSettings,
            // According to our naming convention <type>.insight.<name>
            ['codeStatsInsights.query'],
            undefined
        )

        return modify(
            editedSettings,
            // According to our naming convention <type>.insight.<name>
            ['codeStatsInsights.otherThreshold'],
            undefined
        )
    }

    // Just to be sure that we removed this insight whatever this insight is (backend or extension based)
    // Remove this insight from top level of settings file and from insights.allrepos
    const allPossibleInsightSettingsKeys = [[insightID], [INSIGHTS_ALL_REPOS_SETTINGS_KEY, insightID]]

    let editedSettings = originalSettings
    const parsedSettings = parseJSONCOrError<object>(originalSettings)

    if (isErrorLike(parsedSettings)) {
        return originalSettings
    }

    for (const settingsKey of allPossibleInsightSettingsKeys) {
        // If settings content jsonc doesn't have a value under the settingsKey
        // it fails with parsing error. We should check existence of the property that
        // we're about to remove
        if (get(parsedSettings, settingsKey)) {
            editedSettings = modify(
                originalSettings,
                // According to our naming convention <type>.insight.<name>
                settingsKey,
                undefined
            )
        }
    }

    return editedSettings
}
