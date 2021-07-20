import React, { ReactNode } from 'react'

import { ErrorAlert } from '../../../../../../components/alerts'
import { InsightDashboard } from '../../../../../../schema/settings.schema'
import { FormGroup } from '../../../../../components/form/form-group/FormGroup'
import { FormInput } from '../../../../../components/form/form-input/FormInput'
import { FormRadioInput } from '../../../../../components/form/form-radio-input/FormRadioInput'
import { useField } from '../../../../../components/form/hooks/useField'
import { FORM_ERROR, FormAPI, SubmissionErrors, useForm } from '../../../../../components/form/hooks/useForm'
import { getUserSubject } from '../../../../../components/visibility-picker/VisibilityPicker'
import {
    isGlobalSubject,
    isOrganizationSubject,
    isUserSubject,
    SupportedInsightSubject,
} from '../../../../../core/types/subjects'

import { useDashboardNameValidator } from './hooks/useDashboardNameValidator'
import { getGlobalSubjectTooltipText } from './utils/get-global-subject-tooltip-text'

const DASHBOARD_INITIAL_VALUES: DashboardCreationFields = {
    name: '',
    visibility: 'personal',
}

export interface DashboardCreationFields {
    name: string
    visibility: string
}

export interface InsightsDashboardCreationContentProps {
    /**
     * Initial values for the dashboard creation form.
     */
    initialValues?: DashboardCreationFields

    /**
     * Organizations list used in the creation form for dashboard visibility setting.
     */
    subjects: SupportedInsightSubject[]

    dashboardsSettings: {
        [k: string]: InsightDashboard
    }

    onSubmit: (values: DashboardCreationFields) => SubmissionErrors | Promise<SubmissionErrors> | void
    children: (formAPI: FormAPI<DashboardCreationFields>) => ReactNode
}

/**
 * Renders creation UI form content (fields, submit and cancel buttons).
 */
export const InsightsDashboardCreationContent: React.FunctionComponent<InsightsDashboardCreationContentProps> = props => {
    const { initialValues, subjects, dashboardsSettings, onSubmit, children } = props

    // Calculate initial value for the visibility settings
    const userSubjectID = subjects.find(isUserSubject)?.id ?? ''

    const { ref, handleSubmit, formAPI } = useForm<DashboardCreationFields>({
        initialValues: initialValues ?? { ...DASHBOARD_INITIAL_VALUES, visibility: userSubjectID },
        onSubmit,
    })

    const nameValidator = useDashboardNameValidator({ settings: dashboardsSettings })

    const name = useField({
        name: 'name',
        formApi: formAPI,
        validators: { sync: nameValidator },
    })

    const visibility = useField({
        name: 'visibility',
        formApi: formAPI,
    })

    // We always have user subject in our settings cascade
    const userSubject = getUserSubject(subjects)
    const organizationSubjects = subjects.filter(isOrganizationSubject)

    // We always have global subject in our settings cascade
    const globalSubject = subjects.find(isGlobalSubject)
    const canGlobalSubjectBeEdited = globalSubject?.allowSiteSettingsEdits && globalSubject?.viewerCanAdminister

    return (
        // eslint-disable-next-line react/forbid-elements
        <form noValidate={true} ref={ref} onSubmit={handleSubmit}>
            <FormInput
                required={true}
                autoFocus={true}
                title="Name"
                placeholder="Example: My personal code insight dashboard"
                description="Shown as the title for your dashboard"
                valid={name.meta.touched && name.meta.validState === 'VALID'}
                error={name.meta.touched && name.meta.error}
                {...name.input}
            />

            <FormGroup name="visibility" title="Visibility" contentClassName="d-flex flex-column" className="mb-0 mt-4">
                <FormRadioInput
                    name="visibility"
                    value={userSubject.id}
                    title="Private"
                    description="visible only to you"
                    checked={visibility.input.value === userSubject.id}
                    className="mr-3"
                    onChange={visibility.input.onChange}
                />

                <hr className="mt-2 mb-3" />

                <small className="d-block text-muted mb-3">
                    Shared - visible to everyone is the chosen Organisation
                </small>

                {organizationSubjects.map(org => (
                    <FormRadioInput
                        key={org.id}
                        name="visibility"
                        value={org.id}
                        title={org.displayName ?? org.name}
                        checked={visibility.input.value === org.id}
                        onChange={visibility.input.onChange}
                        className="mr-3"
                    />
                ))}

                {organizationSubjects.length === 0 && (
                    <FormRadioInput
                        name="visibility"
                        value="organization"
                        disabled={true}
                        title="Organization"
                        description="all users in your organization"
                        labelTooltipPosition="right"
                        className="d-inline-block mr-3"
                        labelTooltipText="Create or join an organization to share the dashboard with others!"
                    />
                )}

                <FormRadioInput
                    name="visibility"
                    value={globalSubject?.id}
                    title="Global"
                    description="visible to everyone on your Sourcegraph instance"
                    checked={visibility.input.value === globalSubject?.id}
                    className="mr-3 flex-grow-0"
                    labelTooltipText={getGlobalSubjectTooltipText(globalSubject)}
                    labelTooltipPosition="bottom"
                    disabled={!canGlobalSubjectBeEdited}
                    onChange={visibility.input.onChange}
                />
            </FormGroup>

            {formAPI.submitErrors?.[FORM_ERROR] && (
                <ErrorAlert error={formAPI.submitErrors[FORM_ERROR]} className="mt-2 mb-2" />
            )}

            <div className="d-flex flex-wrap justify-content-end mt-3">{children(formAPI)}</div>
        </form>
    )
}
