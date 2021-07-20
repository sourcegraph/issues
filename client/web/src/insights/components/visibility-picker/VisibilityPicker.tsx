import React, { ChangeEvent } from 'react'
import { Link } from 'react-router-dom'

import { SettingsUserSubject } from '@sourcegraph/shared/src/settings/settings'

import {
    isGlobalSubject,
    isOrganizationSubject,
    isUserSubject,
    SupportedInsightSubject,
} from '../../core/types/subjects'
import { FormGroup } from '../form/form-group/FormGroup'
import { FormRadioInput } from '../form/form-radio-input/FormRadioInput'

export interface VisibilityPickerProps {
    /**
     * Current visibility value.
     */
    value: string

    /**
     * On change handler.
     */
    onChange: (subjectId: string) => void

    /**
     * Supported insight subjects list - to display subjects visibility radio buttons
     */
    subjects: SupportedInsightSubject[]

    /**
     * Custom class name for visibility group label element.
     */
    labelClassName?: string
}

/**
 * Shared component for visibility field for creation UI pages.
 * */
export const VisibilityPicker: React.FunctionComponent<VisibilityPickerProps> = props => {
    const { value, subjects, onChange, labelClassName } = props

    const handleChange = (event: ChangeEvent<HTMLInputElement>): void => {
        onChange(event.target.value)
    }

    const userSubject = getUserSubject(subjects)
    const organizationSubjects = subjects.filter(isOrganizationSubject)
    const globalSubject = subjects.find(isGlobalSubject)!

    const canGlobalSubjectBeEdited = globalSubject.allowSiteSettingsEdits && globalSubject.viewerCanAdminister

    return (
        <FormGroup
            name="visibility"
            title="Visibility"
            subtitle={
                <span>
                    This insight will be always displayed in the{' '}
                    <Link to="/insights/dashboards/all">‘All Insights’ dashboard</Link> by default
                </span>
            }
            className="mb-0 mt-4"
            labelClassName={labelClassName}
            contentClassName="d-flex flex-wrap mb-n2"
        >
            <FormRadioInput
                name="visibility"
                value={userSubject.id}
                title="Private"
                description="only you"
                checked={value === userSubject.id}
                className="mr-3 w-100"
                onChange={handleChange}
            />

            {organizationSubjects.map(org => (
                <FormRadioInput
                    key={org.id}
                    name="visibility"
                    value={org.id}
                    title={org.displayName ?? org.name}
                    description={`all users in ${org.displayName ?? org.name} organization`}
                    checked={value === org.id}
                    onChange={handleChange}
                    className="mr-3 w-100"
                />
            ))}

            {organizationSubjects.length === 0 && (
                <FormRadioInput
                    name="visibility"
                    value="organization"
                    disabled={true}
                    title="Organization"
                    description="all users in your organization"
                    className="mr-3 w-100"
                    labelTooltipText="Create or join the Organization to share code insights with others!"
                />
            )}

            <FormRadioInput
                name="visibility"
                value={globalSubject.id}
                title="Global"
                description="visible to everyone on your Sourcegraph instance"
                checked={value === globalSubject.id}
                disabled={!canGlobalSubjectBeEdited}
                className="mr-3 w-100"
                onChange={handleChange}
            />
        </FormGroup>
    )
}

/**
 * Returns a user setting subject from the settings cascade subjects.
 *
 * @param subjects - insight supported settings subjects
 */
export function getUserSubject(subjects: SupportedInsightSubject[]): SettingsUserSubject {
    // We always have user subject in our settings cascade
    return subjects.find(isUserSubject)!
}
