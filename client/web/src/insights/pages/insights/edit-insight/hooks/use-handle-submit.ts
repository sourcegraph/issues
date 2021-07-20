import { useHistory } from 'react-router-dom'

import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { asError } from '@sourcegraph/shared/src/util/errors'

import { Settings } from '../../../../../schema/settings.schema'
import { FORM_ERROR, SubmissionErrors } from '../../../../components/form/hooks/useForm'
import { Insight } from '../../../../core/types'
import { usePersistEditOperations } from '../../../../hooks/use-persist-edit-operations'

import { getUpdatedSubjectSettings } from './utils'

export interface UseHandleSubmitProps extends PlatformContextProps<'updateSettings'>, SettingsCascadeProps<Settings> {
    originalInsight: Insight | null
}

export interface useHandleSubmitOutput {
    handleEditInsightSubmit: (newInsight: Insight) => Promise<SubmissionErrors>
}

/**
 * Returns submit handler for edit submit form. Updates all related to edit-update
 * settings subjects.
 */
export function useHandleSubmit(props: UseHandleSubmitProps): useHandleSubmitOutput {
    const { originalInsight, platformContext, settingsCascade } = props
    const history = useHistory()
    const { persist } = usePersistEditOperations({ platformContext })

    const handleEditInsightSubmit = async (newInsight: Insight): Promise<SubmissionErrors> => {
        if (!originalInsight) {
            return
        }

        try {
            const editOperations = getUpdatedSubjectSettings({
                oldInsight: originalInsight,
                newInsight,
                settingsCascade,
            })

            await persist(editOperations)

            // Navigate user to the dashboard page with new created dashboard
            history.push(`/insights/dashboards/${newInsight.visibility}`)
        } catch (error) {
            return { [FORM_ERROR]: asError(error) }
        }

        return
    }

    return { handleEditInsightSubmit }
}
