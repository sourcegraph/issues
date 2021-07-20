import { Duration } from 'date-fns'
import { isEqual } from 'lodash'

import {
    isSettingsValid,
    Settings,
    SettingsCascade,
    SettingsCascadeOrError,
} from '@sourcegraph/shared/src/settings/settings'
import { TelemetryService } from '@sourcegraph/shared/src/telemetry/telemetryService'

import { AuthenticatedUser } from '../../auth'

import { InsightTypePrefix, SearchBasedInsightSettings } from './types'

export function logInsightMetrics(
    oldSettingsCascade: SettingsCascadeOrError<Settings>,
    newSettingsCascade: SettingsCascadeOrError<Settings>,
    authUser: AuthenticatedUser,
    telemetryService: TelemetryService
): void {
    logCodeInsightsCount(oldSettingsCascade, newSettingsCascade, authUser, telemetryService)
    logSearchBasedInsightStepSize(oldSettingsCascade, newSettingsCascade, telemetryService)
    logCodeInsightsChanges(oldSettingsCascade, newSettingsCascade, telemetryService)
}

export function logCodeInsightsCount(
    oldSettingsCascade: SettingsCascadeOrError<Settings>,
    newSettingsCascade: SettingsCascadeOrError<Settings>,
    authUser: AuthenticatedUser,
    telemetryService: TelemetryService
): void {
    const oldSettings = isSettingsValid(oldSettingsCascade) && oldSettingsCascade
    const newSettings = isSettingsValid(newSettingsCascade) && newSettingsCascade

    try {
        if (oldSettings && newSettings) {
            const oldGroupedInsights = getInsightsGroupedByType(oldSettings, authUser)
            const newGroupedInsights = getInsightsGroupedByType(newSettings, authUser)

            if (!isEqual(oldGroupedInsights, newGroupedInsights)) {
                telemetryService.log('InsightsGroupedCount', newGroupedInsights)
            }
        }
    } catch {
        // noop
    }
}

export function logSearchBasedInsightStepSize(
    oldSettingsCascade: SettingsCascadeOrError<Settings>,
    newSettingsCascade: SettingsCascadeOrError<Settings>,
    telemetryService: TelemetryService
): void {
    const oldSettings = isSettingsValid(oldSettingsCascade) && oldSettingsCascade
    const newSettings = isSettingsValid(newSettingsCascade) && newSettingsCascade

    try {
        if (oldSettings && newSettings) {
            const oldGroupedStepSizes = getGroupedStepSizes(oldSettings.final)
            const newGroupedStepSizes = getGroupedStepSizes(newSettings.final)

            if (!isEqual(oldGroupedStepSizes, newGroupedStepSizes)) {
                telemetryService.log('InsightsGroupedStepSizes', newGroupedStepSizes)
            }
        }
    } catch {
        // noop
    }
}

/**
 * Collect number current insights that are org-visible by type of insight.
 * */
export function getGroupedStepSizes(settings: Settings): number[] {
    return Object.keys(settings)
        .filter(key => key.startsWith(InsightTypePrefix.search))
        .reduce<number[]>((stepsInDays, key) => {
            const insight = settings[key] as SearchBasedInsightSettings

            return [...stepsInDays, getDaysFromInsightStep(insight.step)]
        }, [])
}

export function getDaysFromInsightStep(step: Duration): number {
    return (Object.keys(step) as (keyof Duration)[]).reduce((days, stepKey) => {
        const stepValue = step[stepKey] ?? 0

        switch (stepKey) {
            case 'years': {
                return days + stepValue * 365
            }

            case 'months': {
                return days + stepValue * 30
            }

            case 'weeks': {
                return days + stepValue * 7
            }

            case 'days': {
                return days + stepValue
            }

            default:
                return days
        }
    }, 0)
}

interface InsightGroups {
    codeStatsInsights: number
    searchBasedInsights: number
}

/**
 * Collect insights count statistic from orgs settings according to
 * auth user organization list.
 * */
export function getInsightsGroupedByType(
    settingsCascade: SettingsCascade<Settings>,
    authUser: AuthenticatedUser
): InsightGroups {
    const { subjects } = settingsCascade
    const {
        organizations: { nodes: orgs },
    } = authUser
    const orgIDs = new Set(orgs.map(org => org.id))

    const orgSubjects = subjects.filter(subject => orgIDs.has(subject.subject.id))

    const finalSettingsOfAllOrgs = orgSubjects.reduce((finalSettings, orgSubject) => {
        const orgSettings = orgSubject.settings

        if (!orgSettings) {
            return finalSettings
        }
        return { ...finalSettings, ...orgSettings }
    }, {})

    const codeStatsInsights = Object.keys(finalSettingsOfAllOrgs).filter(key =>
        key.startsWith(InsightTypePrefix.langStats)
    ).length

    const searchBasedInsights = Object.keys(finalSettingsOfAllOrgs).filter(key =>
        key.startsWith(InsightTypePrefix.search)
    ).length

    return {
        codeStatsInsights,
        searchBasedInsights,
    }
}

export function logCodeInsightsChanges(
    oldSettingsCascade: SettingsCascadeOrError<Settings>,
    newSettingsCascade: SettingsCascadeOrError<Settings>,
    telemetryService: TelemetryService
): void {
    try {
        const oldSettings = isSettingsValid(oldSettingsCascade) && oldSettingsCascade.final
        const newSettings = isSettingsValid(newSettingsCascade) && newSettingsCascade.final

        if (oldSettings && newSettings) {
            for (const { action, insightType } of diffCodeInsightsSettings(oldSettings, newSettings)) {
                telemetryService.log(`Insight${action}`, { insightType })
            }
        }
    } catch {
        // noop
    }
}

interface InsightDiff {
    action: 'Addition' | 'Removal' | 'Edit'
    insightType: string
}

export function diffCodeInsightsSettings(oldSettings: Settings, newSettings: Settings): InsightDiff[] {
    const oldInsights = new Map<string, InsightData>()
    const newInsights = new Map<string, InsightData>()

    for (const key of Object.keys(oldSettings)) {
        const insightMetadata = parseInsightSettingsKey(key)
        if (insightMetadata) {
            const configuration = oldSettings[key]
            oldInsights.set(insightMetadata.name, { ...insightMetadata, configuration })
        }
    }

    for (const key of Object.keys(newSettings)) {
        const insightMetadata = parseInsightSettingsKey(key)
        if (insightMetadata) {
            const configuration = newSettings[key]
            newInsights.set(insightMetadata.name, { ...insightMetadata, configuration })
        }
    }

    const insightDiffs: InsightDiff[] = []

    for (const [name, { insightType, configuration: oldConfiguration }] of oldInsights) {
        const newInsight = newInsights.get(name)
        if (!newInsight) {
            insightDiffs.push({ insightType, action: 'Removal' })
        } else if (!isEqual(oldConfiguration, newInsight.configuration)) {
            insightDiffs.push({ insightType, action: 'Edit' })
        }
    }

    for (const [name, { insightType }] of newInsights) {
        if (!oldInsights.has(name)) {
            insightDiffs.push({ insightType, action: 'Addition' })
        }
    }

    return insightDiffs
}

interface InsightData {
    insightType: string
    name: string
    configuration: any
}

export function parseInsightSettingsKey(key: string): Omit<InsightData, 'configuration'> | undefined {
    const [insightType, maybeInsight, insightName] = key.split('.')

    if (!insightType) {
        return undefined
    }

    switch (insightType) {
        // Exceptional extensions that don't adhere to the standard insight settings key format
        case 'codeStatsInsights': {
            if (key.split('.')[1] !== 'query') {
                return
            }

            return {
                insightType,
                name: insightType,
            }
        }

        // Extensions that contribute insights should define settings
        // keys with the format `$EXTENSION_NAME.insight` or `$EXTENSION_NAME.insight.$INSIGHT_NAME`
        // example key: "searchInsights.insights.graphQLTypesMigration"
        default: {
            if (maybeInsight !== 'insight') {
                return
            }

            return {
                insightType,
                name: insightName ?? insightType, // fallback to type
            }
        }
    }
}
