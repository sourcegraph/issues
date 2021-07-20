import classnames from 'classnames'
import React, { FormEventHandler, RefObject } from 'react'

import { ErrorAlert } from '../../../../../../../components/alerts'
import { LoaderButton } from '../../../../../../../components/LoaderButton'
import { FormGroup } from '../../../../../../components/form/form-group/FormGroup'
import { FormInput } from '../../../../../../components/form/form-input/FormInput'
import { FormRadioInput } from '../../../../../../components/form/form-radio-input/FormRadioInput'
import { useFieldAPI } from '../../../../../../components/form/hooks/useField'
import { FORM_ERROR, SubmissionErrors } from '../../../../../../components/form/hooks/useForm'
import { RepositoriesField } from '../../../../../../components/form/repositories-field/RepositoriesField'
import { VisibilityPicker } from '../../../../../../components/visibility-picker/VisibilityPicker'
import { SupportedInsightSubject } from '../../../../../../core/types/subjects'
import { CreateInsightFormFields, EditableDataSeries } from '../../types'
import { FormSeries } from '../form-series/FormSeries'

import styles from './SearchInsightCreationForm.module.scss'

interface CreationSearchInsightFormProps {
    /** This component might be used in edit or creation insight case. */
    mode?: 'creation' | 'edit'

    innerRef: RefObject<any>
    handleSubmit: FormEventHandler
    submitErrors: SubmissionErrors
    submitting: boolean
    submitted: boolean
    className?: string
    isFormClearActive?: boolean

    /**
     * Enables the experimental insight mode (run insight on all repositories in the instance)
     */
    hasAllReposUI?: boolean

    title: useFieldAPI<CreateInsightFormFields['title']>
    repositories: useFieldAPI<CreateInsightFormFields['repositories']>
    allReposMode: useFieldAPI<CreateInsightFormFields['allRepos']>

    visibility: useFieldAPI<CreateInsightFormFields['visibility']>
    subjects: SupportedInsightSubject[]

    series: useFieldAPI<CreateInsightFormFields['series']>
    step: useFieldAPI<CreateInsightFormFields['step']>
    stepValue: useFieldAPI<CreateInsightFormFields['stepValue']>

    onCancel: () => void

    /**
     * Handler to listen latest value form particular series edit form
     * Used to get information for live preview chart.
     * */
    onSeriesLiveChange: (liveSeries: EditableDataSeries, isValid: boolean, index: number) => void

    /**
     * Handlers for CRUD operation over series. Add, delete, update and cancel
     * series edit form.
     * */
    onEditSeriesRequest: (openedCardIndex: number) => void
    onEditSeriesCommit: (seriesIndex: number, editedSeries: EditableDataSeries) => void
    onEditSeriesCancel: (closedCardIndex: number) => void
    onSeriesRemove: (removedSeriesIndex: number) => void

    onFormReset: () => void
}

/**
 * Displays creation code insight form (title, visibility, series, etc.)
 * UI layer only, all controlled data should be managed by consumer of this component.
 * */
export const SearchInsightCreationForm: React.FunctionComponent<CreationSearchInsightFormProps> = props => {
    const {
        mode,
        innerRef,
        handleSubmit,
        submitErrors,
        submitting,
        submitted,
        title,
        repositories,
        allReposMode,
        visibility,
        subjects,
        series,
        stepValue,
        step,
        className,
        isFormClearActive,
        hasAllReposUI,
        onCancel,
        onSeriesLiveChange,
        onEditSeriesRequest,
        onEditSeriesCommit,
        onEditSeriesCancel,
        onSeriesRemove,
        onFormReset,
    } = props

    const isEditMode = mode === 'edit'

    return (
        // eslint-disable-next-line react/forbid-elements
        <form
            noValidate={true}
            ref={innerRef}
            onSubmit={handleSubmit}
            onReset={onFormReset}
            className={classnames(className, 'd-flex flex-column')}
        >
            <FormGroup
                name="insight repositories"
                title="Repositories"
                subtitle="Create a list of repositories to run your search over"
            >
                <FormInput
                    as={RepositoriesField}
                    autoFocus={true}
                    required={true}
                    description="Separate repositories with commas"
                    placeholder="Example: github.com/sourcegraph/sourcegraph"
                    loading={repositories.meta.validState === 'CHECKING'}
                    valid={repositories.meta.touched && repositories.meta.validState === 'VALID'}
                    error={repositories.meta.touched && repositories.meta.error}
                    {...repositories.input}
                    className="mb-0 d-flex flex-column"
                />

                {hasAllReposUI && (
                    <>
                        <label className="d-flex align-items-center mb-2 mt-2 font-weight-normal">
                            <input
                                type="checkbox"
                                {...allReposMode.input}
                                value="all-repos-mode"
                                checked={allReposMode.input.value}
                            />

                            <span className="pl-2">Run your insight over all your repositories</span>
                        </label>

                        <hr className={styles.creationInsightFormSeparator} />
                    </>
                )}
            </FormGroup>

            <FormGroup
                name="data series group"
                title="Data series"
                subtitle="Add any number of data series to your chart"
                error={series.meta.touched && series.meta.error}
                innerRef={series.input.ref}
                className={!hasAllReposUI ? 'mt-5' : undefined}
            >
                <FormSeries
                    series={series.input.value}
                    showValidationErrorsOnMount={submitted}
                    onLiveChange={onSeriesLiveChange}
                    onEditSeriesRequest={onEditSeriesRequest}
                    onEditSeriesCommit={onEditSeriesCommit}
                    onEditSeriesCancel={onEditSeriesCancel}
                    onSeriesRemove={onSeriesRemove}
                />
            </FormGroup>

            <hr className={styles.creationInsightFormSeparator} />

            <FormGroup name="chart settings group" title="Chart settings">
                <FormInput
                    title="Title"
                    required={true}
                    description="Shown as the title for your insight"
                    placeholder="Example: Migration to React function components"
                    valid={title.meta.touched && title.meta.validState === 'VALID'}
                    error={title.meta.touched && title.meta.error}
                    {...title.input}
                    className="d-flex flex-column"
                />

                <VisibilityPicker
                    subjects={subjects}
                    value={visibility.input.value}
                    labelClassName={styles.creationInsightFormGroupLabel}
                    onChange={visibility.input.onChange}
                />

                <FormGroup
                    name="insight step group"
                    title="Granularity: distance between data points"
                    description="The prototype supports 7 datapoints, so your total x-axis timeframe is 6 times the distance between each point."
                    error={stepValue.meta.touched && stepValue.meta.error}
                    className="mt-4"
                    labelClassName={styles.creationInsightFormGroupLabel}
                    contentClassName="d-flex flex-wrap mb-n2"
                >
                    <FormInput
                        placeholder="ex. 2"
                        required={true}
                        type="number"
                        min={1}
                        {...stepValue.input}
                        valid={stepValue.meta.touched && stepValue.meta.validState === 'VALID'}
                        errorInputState={stepValue.meta.touched && stepValue.meta.validState === 'INVALID'}
                        className={classnames(styles.creationInsightFormStepInput)}
                    />

                    <FormRadioInput
                        title="Hours"
                        name="step"
                        value="hours"
                        checked={step.input.value === 'hours'}
                        onChange={step.input.onChange}
                        disabled={step.input.disabled}
                        className="mr-3"
                    />
                    <FormRadioInput
                        title="Days"
                        name="step"
                        value="days"
                        checked={step.input.value === 'days'}
                        onChange={step.input.onChange}
                        disabled={step.input.disabled}
                        className="mr-3"
                    />
                    <FormRadioInput
                        title="Weeks"
                        name="step"
                        value="weeks"
                        checked={step.input.value === 'weeks'}
                        onChange={step.input.onChange}
                        disabled={step.input.disabled}
                        className="mr-3"
                    />
                    <FormRadioInput
                        title="Months"
                        name="step"
                        value="months"
                        checked={step.input.value === 'months'}
                        onChange={step.input.onChange}
                        disabled={step.input.disabled}
                        className="mr-3"
                    />
                    <FormRadioInput
                        title="Years"
                        name="step"
                        value="years"
                        checked={step.input.value === 'years'}
                        onChange={step.input.onChange}
                        disabled={step.input.disabled}
                        className="mr-3"
                    />
                </FormGroup>
            </FormGroup>

            <hr className={styles.creationInsightFormSeparator} />

            <div className="d-flex flex-wrap align-items-baseline">
                {submitErrors?.[FORM_ERROR] && <ErrorAlert error={submitErrors[FORM_ERROR]} />}

                <LoaderButton
                    alwaysShowLabel={true}
                    loading={submitting}
                    label={submitting ? 'Submitting' : isEditMode ? 'Edit insight' : 'Create code insight'}
                    type="submit"
                    disabled={submitting}
                    data-testid="insight-save-button"
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
