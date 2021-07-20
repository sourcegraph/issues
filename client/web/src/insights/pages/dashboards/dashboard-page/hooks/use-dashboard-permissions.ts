import { SettingsCascadeOrError } from '@sourcegraph/shared/src/settings/settings'

import { Settings } from '../../../../../schema/settings.schema'
import { InsightDashboard, isRealDashboard, isVirtualDashboard } from '../../../../core/types'
import { isSettingsBasedInsightsDashboard } from '../../../../core/types/dashboard/real-dashboard'
import { isGlobalSubject } from '../../../../core/types/subjects'
import { useInsightSubjects } from '../../../../hooks/use-insight-subjects/use-insight-subjects'

enum DashboardReasonDenied {
    BuiltInCantBeEdited,
    PermissionDenied,
    UnknownDashboard,
}

type DashboardPermissions =
    | {
          isConfigurable: false
          reason: DashboardReasonDenied
      }
    | {
          isConfigurable: true
      }

const DEFAULT_DASHBOARD_PERMISSIONS: DashboardPermissions = {
    isConfigurable: false,
    reason: DashboardReasonDenied.UnknownDashboard,
}

export function useDashboardPermissions(
    dashboard: InsightDashboard | undefined,
    settingsCascade: SettingsCascadeOrError<Settings>
): DashboardPermissions {
    const supportedSubject = useInsightSubjects({ settingsCascade })

    if (isVirtualDashboard(dashboard)) {
        return {
            isConfigurable: false,
            reason: DashboardReasonDenied.BuiltInCantBeEdited,
        }
    }

    const dashboardOwner = supportedSubject.find(subject => subject.id === dashboard?.owner?.id)

    // No dashboard can't be modified
    if (!dashboard || !dashboardOwner) {
        return DEFAULT_DASHBOARD_PERMISSIONS
    }

    if (isRealDashboard(dashboard)) {
        // Settings based insights dashboards (custom dashboards created by users)
        if (isSettingsBasedInsightsDashboard(dashboard)) {
            // Global scope permission handling
            if (isGlobalSubject(dashboardOwner)) {
                const canBeEdited = dashboardOwner.viewerCanAdminister && dashboardOwner.allowSiteSettingsEdits

                if (!canBeEdited) {
                    return {
                        isConfigurable: false,
                        reason: DashboardReasonDenied.PermissionDenied,
                    }
                }
            }

            return {
                isConfigurable: true,
            }
        }

        // Not settings based dashboard (built-in-dashboard case)
        return {
            isConfigurable: false,
            reason: DashboardReasonDenied.BuiltInCantBeEdited,
        }
    }

    return DEFAULT_DASHBOARD_PERMISSIONS
}

export function getTooltipMessage(permissions: DashboardPermissions): string | undefined {
    if (permissions.isConfigurable) {
        return
    }

    switch (permissions.reason) {
        case DashboardReasonDenied.UnknownDashboard:
            return 'Dashboard not found'
        case DashboardReasonDenied.PermissionDenied:
            return "You don't a permission to edit this dashboard"
        case DashboardReasonDenied.BuiltInCantBeEdited:
            return "Built-in dashboards can't be edited"
    }
}
