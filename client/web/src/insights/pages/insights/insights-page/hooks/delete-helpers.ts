import { get } from 'lodash'

import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { isErrorLike } from '@sourcegraph/shared/src/util/errors'

import { Settings } from '../../../../../schema/settings.schema'
import {
    RemoveInsight,
    RemoveInsightFromDashboard,
    SettingsOperation,
    SettingsOperationType,
} from '../../../../core/settings-action/edits'
import { INSIGHTS_ALL_REPOS_SETTINGS_KEY, INSIGHTS_DASHBOARDS_SETTINGS_KEY } from '../../../../core/types'

interface Props extends SettingsCascadeProps<Settings> {
    insightId: string
}

/**
 * Returns list of edit operations. Remove insight from all setting cascade subjects
 * and from all dashboard that include this insightId in their insight ids.
 */
export function getDeleteInsightEditOperations(props: Props): SettingsOperation[] {
    const { settingsCascade, insightId } = props

    if (!settingsCascade.subjects) {
        return []
    }

    const removeInsightOperation = settingsCascade.subjects
        .filter(configuredSubject => {
            const { settings } = configuredSubject

            if (!settings || isErrorLike(settings)) {
                return false
            }

            const hasExtensionInsight = settings[insightId]
            const hasBackendInsight = get(settings, [INSIGHTS_ALL_REPOS_SETTINGS_KEY, insightId])

            return hasExtensionInsight || hasBackendInsight
        })
        .map<RemoveInsight>(configuredSubject => ({
            type: SettingsOperationType.removeInsight,
            subjectId: configuredSubject.subject.id,
            insightID: insightId,
        }))

    const removeInsightFromAllDashboards = settingsCascade.subjects.flatMap(configuredSubject => {
        const { settings, subject } = configuredSubject

        if (!settings || isErrorLike(settings)) {
            return []
        }

        const dashboards = settings[INSIGHTS_DASHBOARDS_SETTINGS_KEY] ?? {}

        return Object.keys(dashboards)
            .filter(key => dashboards[key].insightIds?.includes(insightId))
            .map<RemoveInsightFromDashboard>(key => ({
                type: SettingsOperationType.removeInsightFromDashboard,
                subjectId: subject.id,
                dashboardSettingKey: key,
                insightId,
            }))
    })

    return [...removeInsightOperation, ...removeInsightFromAllDashboards]
}
