import { camelCase } from 'lodash'
import { useContext } from 'react'
import { useHistory } from 'react-router-dom'

import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { asError } from '@sourcegraph/shared/src/util/errors'

import { AuthenticatedUser } from '../../../../../auth'
import { InsightDashboard } from '../../../../../schema/settings.schema'
import { FORM_ERROR, SubmissionErrors } from '../../../../components/form/hooks/useForm'
import { InsightsApiContext } from '../../../../core/backend/api-provider'
import { addDashboardToSettings, removeDashboardFromSettings } from '../../../../core/settings-action/dashboards'
import { SettingsBasedInsightDashboard } from '../../../../core/types'
import { DashboardCreationFields } from '../../creation/components/insights-dashboard-creation-content/InsightsDashboardCreationContent'
import { createSanitizedDashboard } from '../../creation/utils/dashboard-sanitizer'

interface useUpdateDashboardProps extends PlatformContextProps<'updateSettings'> {
    authenticatedUser: Pick<AuthenticatedUser, 'id' | 'organizations' | 'username'>

    /**
     * Old version of dashboard before edit operation.
     */
    previousDashboard: SettingsBasedInsightDashboard | undefined
}

export type DashboardUpdateHandler = (dashboardValues: DashboardCreationFields) => Promise<void | SubmissionErrors>

/**
 * Returns an update-callback to update (remove old and add new one) dashboard.
 */
export function useUpdateDashboardCallback(props: useUpdateDashboardProps): DashboardUpdateHandler {
    const { platformContext, previousDashboard } = props
    const { updateSubjectSettings, getSubjectSettings } = useContext(InsightsApiContext)
    const history = useHistory()

    return async dashboardValues => {
        if (!previousDashboard) {
            return
        }

        try {
            if (previousDashboard.owner.id !== dashboardValues.visibility) {
                const settings = await getSubjectSettings(previousDashboard.owner.id).toPromise()
                const editedSettings = removeDashboardFromSettings(settings.contents, previousDashboard.settingsKey)

                await updateSubjectSettings(platformContext, previousDashboard.owner.id, editedSettings).toPromise()
            }

            const subjectID = dashboardValues.visibility

            const settings = await getSubjectSettings(subjectID).toPromise()

            let settingsContent = settings.contents

            // Since id (settings key) of insight dashboard is based on its title
            // if title was changed we need remove old dashboard object from the settings
            // by dashboard's old id
            if (previousDashboard.title !== dashboardValues.name) {
                settingsContent = removeDashboardFromSettings(settingsContent, previousDashboard.settingsKey)
            }

            const updatedDashboard: InsightDashboard = {
                ...createSanitizedDashboard(dashboardValues),
                // We have to preserve id and insights IDs value since edit UI
                // doesn't have these options.
                id: previousDashboard.id,
                insightIds: previousDashboard.insightIds,
            }

            settingsContent = addDashboardToSettings(settingsContent, updatedDashboard)

            await updateSubjectSettings(platformContext, subjectID, settingsContent).toPromise()

            history.push(`/insights/dashboards/${camelCase(updatedDashboard.title)}`)
        } catch (error) {
            return { [FORM_ERROR]: asError(error) }
        }

        return
    }
}
