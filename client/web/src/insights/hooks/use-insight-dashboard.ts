import { useMemo } from 'react'

import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'

import { Settings } from '../../schema/settings.schema'
import { isRealDashboard, RealInsightDashboard } from '../core/types'

import { useDashboards } from './use-dashboards/use-dashboards'

export interface UseInsightDashboardsProps extends SettingsCascadeProps<Settings> {
    insightId: string
}

/**
 * Returns all dashboards (built-in - personal or org) that have insight Id
 */
export function useInsightDashboards(props: UseInsightDashboardsProps): RealInsightDashboard[] {
    const { settingsCascade, insightId } = props

    const dashboards = useDashboards(settingsCascade)

    return useMemo(
        () => dashboards.filter(isRealDashboard).filter(dashboard => dashboard.insightIds?.includes(insightId)),
        [dashboards, insightId]
    )
}
