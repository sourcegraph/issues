import { addInsightToDashboard, removeInsightFromDashboard } from '../dashboards'
import { addInsightToSettings, removeInsightFromSettings } from '../insights'

import { SettingsOperation, SettingsOperationType } from './edits'

/**
 * Apply edit operation over jsonc settings string and return serialized final string
 * with all applied edit operations.
 *
 * @param settings - original jsonc setting content
 * @param operations - list of edit operations
 */
export function applyEditOperations(settings: string, operations: SettingsOperation[]): string {
    let settingsContent: string = settings

    for (const operation of operations) {
        switch (operation.type) {
            case SettingsOperationType.addInsight:
                settingsContent = addInsightToSettings(settingsContent, operation.insight)
                continue
            case SettingsOperationType.removeInsight:
                settingsContent = removeInsightFromSettings({
                    originalSettings: settingsContent,
                    insightID: operation.insightID,
                })
                continue
            case SettingsOperationType.addInsightToDashboard:
                settingsContent = addInsightToDashboard(
                    settingsContent,
                    operation.dashboardSettingKey,
                    operation.insightId
                )
                continue
            case SettingsOperationType.removeInsightFromDashboard:
                settingsContent = removeInsightFromDashboard(
                    settingsContent,
                    operation.dashboardSettingKey,
                    operation.insightId
                )
        }
    }

    return settingsContent
}
