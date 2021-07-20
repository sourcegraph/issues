import { useCallback } from 'react'

import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'

import { InsightTypePrefix } from '../../../../core/types'
import { usePersistEditOperations } from '../../../../hooks/use-persist-edit-operations'

import { getDeleteInsightEditOperations } from './delete-helpers'

export interface UseDeleteInsightProps extends SettingsCascadeProps, PlatformContextProps<'updateSettings'> {}

export interface UseDeleteInsightAPI {
    handleDelete: (insightID: string) => Promise<void>
}

/**
 * Returns delete handler that deletes insight from all subject settings and from all dashboards
 * that include this insight.
 */
export function useDeleteInsight(props: UseDeleteInsightProps): UseDeleteInsightAPI {
    const { settingsCascade, platformContext } = props
    const { persist } = usePersistEditOperations({ platformContext })

    const handleDelete = useCallback(
        async (insightID: string) => {
            // For backward compatibility with old code stats insight api we have to delete
            // this insight in a special way. See link below for more information.
            // https://github.com/sourcegraph/sourcegraph-code-stats-insights/blob/master/src/code-stats-insights.ts#L33
            const isOldCodeStatsInsight = insightID === `${InsightTypePrefix.langStats}.language`

            const keyForSearchInSettings = isOldCodeStatsInsight
                ? // Hardcoded value of id from old version of stats insight extension API
                  'codeStatsInsights.query'
                : insightID

            try {
                const deleteInsightOperations = getDeleteInsightEditOperations({
                    insightId: keyForSearchInSettings,
                    settingsCascade,
                })

                await persist(deleteInsightOperations)
            } catch (error) {
                // TODO [VK] Improve error UI for deleting
                console.error(error)
            }
        },
        [persist, settingsCascade]
    )

    return { handleDelete }
}
