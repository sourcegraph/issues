import { Duration } from 'date-fns'

import { DataSeries } from '../backend/types'

export const INSIGHTS_ALL_REPOS_SETTINGS_KEY = 'insights.allrepos'

export enum InsightTypePrefix {
    search = 'searchInsights.insight',
    langStats = 'codeStatsInsights.insight',
}

export type Insight = SearchBasedInsight | LangStatsInsight
export type InsightConfiguration = SearchBasedInsightSettings | LangStatsInsightSettings

/**
 * Visibility setting which responsible for where insight will appear.
 * possible value 'personal' | '<org id 1> ... | ... <org id N>'
 * */
export type InsightVisibility = string

/**
 * [Synthetic] field needed only for code insight logic but not for extension
 */
export interface SyntheticInsightFields {
    /**
     * ID of insight <type of insight>.insight.<name of insight>
     * */
    id: string

    /**
     * Visibility of insight. Personal or organization setting cascade subject.
     * */
    visibility: InsightVisibility
}

export enum InsightType {
    Extension = 'extension',
    Backend = 'backend',
}

export interface SearchExtensionBasedInsight extends SearchBasedInsightSettings, SyntheticInsightFields {
    type: InsightType.Extension
}

export interface SearchBackendBasedInsight extends SearchBasedInsightSettings, SyntheticInsightFields {
    type: InsightType.Backend
}

/**
 * Search based insight supports type types of configuration
 *
 * Extension based works via insight extension and lives in settings file on top level
 * search "searchInsights.insight.<name>": {...config}
 *
 * Backend based works on BE and lives in "insights.allrepos": { "searchInsights.insight.<name>" : { ...config }}
 */
export type SearchBasedInsight = SearchExtensionBasedInsight | SearchBackendBasedInsight

/**
 * See public API of search insight extension
 * https://github.com/sourcegraph/sourcegraph-search-insights/blob/master/package.json#L26
 */
export interface SearchBasedInsightSettings {
    title: string
    repositories: string[]
    series: DataSeries[]
    step: Duration
}

/**
 * Extended Lang Stats Insight.
 * Some fields and settings (id, visibility) do not exist implicitly in user/org settings but
 * we have to have these to operate with insight properly.
 * */
export interface LangStatsInsight extends LangStatsInsightSettings, SyntheticInsightFields {
    type: InsightType.Extension
}

/**
 * Lang stats insight as it is presented in user/org settings cascade.
 * See public API of code stats extension
 * https://github.com/sourcegraph/sourcegraph-code-stats-insights/blob/master/package.json#L27
 */
export interface LangStatsInsightSettings {
    title: string
    repository: string
    otherThreshold: number
}

export const isSearchBasedInsightId = (id: string): boolean => id.startsWith(InsightTypePrefix.search)
export const isLangStatsdInsightId = (id: string): boolean => id.startsWith(InsightTypePrefix.langStats)

export function isInsightSettingKey(key: string): boolean {
    return isSearchBasedInsightId(key) || isLangStatsdInsightId(key)
}

export function isSearchBasedInsight(insight: Insight): insight is SearchBasedInsight {
    return isSearchBasedInsightId(insight.id)
}

export function isLangStatsInsight(insight: Insight): insight is LangStatsInsight {
    return isLangStatsdInsightId(insight.id)
}
