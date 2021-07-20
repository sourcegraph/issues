import classnames from 'classnames'
import React, { FormEventHandler, RefObject } from 'react'

import { ErrorAlert } from '../../../../../../../components/alerts'
import { LoaderButton } from '../../../../../../../components/LoaderButton'
import { FormInput } from '../../../../../../components/form/form-input/FormInput'
import { useFieldAPI } from '../../../../../../components/form/hooks/useField'
import { FORM_ERROR, SubmissionErrors } from '../../../../../../components/form/hooks/useForm'
import { RepositoryField } from '../../../../../../components/form/repositories-field/RepositoryField'
import { VisibilityPicker } from '../../../../../../components/visibility-picker/VisibilityPicker'
import { SupportedInsightSubject } from '../../../../../../core/types/subjects'
import { LangStatsCreationFormFields } from '../../types'

import styles from './LangStatsInsightCreationForm.module.scss'

export interface LangStatsInsightCreationFormProps {
    mode?: 'creation' | 'edit'
    innerRef: RefObject<any>
    handleSubmit: FormEventHandler
    submitErrors: SubmissionErrors
    submitting: boolean
    className?: string
    isFormClearActive?: boolean

    title: useFieldAPI<LangStatsCreationFormFields['title']>
    repository: useFieldAPI<LangStatsCreationFormFields['repository']>
    threshold: useFieldAPI<LangStatsCreationFormFields['threshold']>
    visibility: useFieldAPI<LangStatsCreationFormFields['visibility']>
    subjects: SupportedInsightSubject[]

    onCancel: () => void
    onFormReset: () => void
}

export const LangStatsInsightCreationForm: React.FunctionComponent<LangStatsInsightCreationFormProps> = props => {
    const {
        mode = 'creation',
        innerRef,
        handleSubmit,
        submitErrors,
        submitting,
        className,
        title,
        repository,
        threshold,
        visibility,
        subjects,
        onCancel,
        onFormReset,
        isFormClearActive,
    } = props

    const isEditMode = mode === 'edit'

    return (
        // eslint-disable-next-line react/forbid-elements
        <form
            ref={innerRef}
            noValidate={true}
            className={classnames(className, 'd-flex flex-column')}
            onSubmit={handleSubmit}
            onReset={onFormReset}
        >
            <FormInput
                as={RepositoryField}
                required={true}
                autoFocus={true}
                title="Repository"
                description="This insight is limited to one repository. You can set up multiple language usage charts for analyzing other repositories."
                placeholder="Example: github.com/sourcegraph/sourcegraph"
                loading={repository.meta.validState === 'CHECKING'}
                valid={repository.meta.touched && repository.meta.validState === 'VALID'}
                error={repository.meta.touched && repository.meta.error}
                {...repository.input}
                className="mb-0"
            />

            <FormInput
                required={true}
                title="Title"
                description="Shown as the title for your insight."
                placeholder="Example: Language Usage in RepositoryName"
                valid={title.meta.touched && title.meta.validState === 'VALID'}
                error={title.meta.touched && title.meta.error}
                {...title.input}
                className="mb-0 mt-4"
            />

            <FormInput
                required={true}
                min={1}
                max={100}
                type="number"
                title="Threshold of ‘Other’ category"
                description="Languages with usage lower than the threshold are grouped into an 'other' category."
                valid={threshold.meta.touched && threshold.meta.validState === 'VALID'}
                error={threshold.meta.touched && threshold.meta.error}
                {...threshold.input}
                className="mb-0 mt-4"
                inputClassName={styles.formThresholdInput}
                inputSymbol={<span className={styles.formThresholdInputSymbol}>%</span>}
            />

            <VisibilityPicker subjects={subjects} value={visibility.input.value} onChange={visibility.input.onChange} />

            <hr className={styles.formSeparator} />

            <div className="d-flex flex-wrap align-items-baseline">
                {submitErrors?.[FORM_ERROR] && <ErrorAlert error={submitErrors[FORM_ERROR]} />}

                <LoaderButton
                    alwaysShowLabel={true}
                    data-testid="insight-save-button"
                    loading={submitting}
                    label={submitting ? 'Submitting' : isEditMode ? 'Edit insight' : 'Create code insight'}
                    type="submit"
                    disabled={submitting}
                    className="btn btn-primary mr-2 mb-2"
                />

                <button type="button" className="btn btn-outline-secondary mb-2 mr-auto" onClick={onCancel}>
                    Cancel
                </button>

                <button type="reset" disabled={!isFormClearActive} className="btn btn-outline-secondary border-0">
                    Clear all fields
                </button>
            </div>
        </form>
    )
}
