import classnames from 'classnames'
import React from 'react'
import { noop } from 'rxjs'

import { Settings } from '@sourcegraph/shared/src/settings/settings'

import { FormChangeEvent, SubmissionErrors } from '../../../../../../components/form/hooks/useForm'
import { SupportedInsightSubject } from '../../../../../../core/types/subjects'
import { getExperimentalFeatures } from '../../../../../../utils/get-experimental-features'
import { CreateInsightFormFields } from '../../types'
import { getSanitizedRepositories } from '../../utils/insight-sanitizer'
import { SearchInsightLivePreview } from '../live-preview-chart/SearchInsightLivePreview'
import { SearchInsightCreationForm } from '../search-insight-creation-form/SearchInsightCreationForm'

import { useEditableSeries, createDefaultEditSeries } from './hooks/use-editable-series'
import { useInsightCreationForm } from './hooks/use-insight-creation-form/use-insight-creation-form'
import styles from './SearchInsightCreationContent.module.scss'

export interface SearchInsightCreationContentProps {
    /** This component might be used in edit or creation insight case. */
    mode?: 'creation' | 'edit'
    /** Final settings cascade. Used for title field validation. */
    settings?: Settings | null

    /** List of all supportable insight subjects */
    subjects?: SupportedInsightSubject[]

    /** Initial value for all form fields. */
    initialValue?: CreateInsightFormFields
    /** Custom class name for root form element. */
    className?: string
    /** Test id for the root content element (form element). */
    dataTestId?: string
    /** Submit handler for form element. */
    onSubmit: (values: CreateInsightFormFields) => SubmissionErrors | Promise<SubmissionErrors> | void
    /** Cancel handler. */
    onCancel?: () => void
    /** Change handlers is called every time when user changed any field within the form. */
    onChange?: (event: FormChangeEvent<CreateInsightFormFields>) => void
}

export const SearchInsightCreationContent: React.FunctionComponent<SearchInsightCreationContentProps> = props => {
    const {
        mode = 'creation',
        subjects = [],
        settings,
        initialValue,
        className,
        dataTestId,
        onSubmit,
        onCancel = noop,
        onChange = noop,
    } = props

    const isEditMode = mode === 'edit'

    const { codeInsightsAllRepos } = getExperimentalFeatures(settings)

    const {
        form: { values, formAPI, ref, handleSubmit },
        title,
        repositories,
        series,
        visibility,
        step,
        stepValue,
        allReposMode,
    } = useInsightCreationForm({
        settings,
        subjects,
        initialValue,
        touched: isEditMode,
        onChange,
        onSubmit,
    })

    const { editSeries, listen, editRequest, editCommit, cancelEdit, deleteSeries } = useEditableSeries({
        series,
    })

    const handleFormReset = (): void => {
        // TODO [VK] Change useForm API in order to implement form.reset method.
        title.input.onChange('')
        repositories.input.onChange('')
        // Focus first element of the form
        repositories.input.ref.current?.focus()
        visibility.input.onChange('personal')
        series.input.onChange([createDefaultEditSeries({ edit: true })])
        stepValue.input.onChange('2')
        step.input.onChange('months')
    }

    const validEditSeries = editSeries.filter(series => series.valid)
    const repositoriesList = getSanitizedRepositories(repositories.input.value)

    // If some fields that needed to run live preview  are invalid
    // we should disabled live chart preview
    const allFieldsForPreviewAreValid =
        (repositories.meta.validState === 'VALID' || repositories.meta.validState === 'CHECKING') &&
        repositoriesList.length > 0 &&
        (series.meta.validState === 'VALID' || validEditSeries.length) &&
        stepValue.meta.validState === 'VALID' &&
        // For all repos mode we are not able to show the live preview chart
        !allReposMode.input.value

    const hasFilledValue =
        values.series?.some(line => line.name !== '' || line.query !== '') ||
        values.repositories !== '' ||
        values.title !== ''

    return (
        <div data-testid={dataTestId} className={classnames(styles.content, className)}>
            <SearchInsightCreationForm
                mode={mode}
                className={styles.contentForm}
                innerRef={ref}
                handleSubmit={handleSubmit}
                submitErrors={formAPI.submitErrors}
                submitting={formAPI.submitting}
                submitted={formAPI.submitted}
                title={title}
                repositories={repositories}
                allReposMode={allReposMode}
                visibility={visibility}
                subjects={subjects}
                series={series}
                step={step}
                stepValue={stepValue}
                isFormClearActive={hasFilledValue}
                hasAllReposUI={codeInsightsAllRepos}
                onSeriesLiveChange={listen}
                onCancel={onCancel}
                onEditSeriesRequest={editRequest}
                onEditSeriesCancel={cancelEdit}
                onEditSeriesCommit={editCommit}
                onSeriesRemove={deleteSeries}
                onFormReset={handleFormReset}
            />

            <SearchInsightLivePreview
                disabled={!allFieldsForPreviewAreValid}
                repositories={repositories.meta.value}
                series={editSeries}
                step={step.meta.value}
                stepValue={stepValue.meta.value}
                className={styles.contentLivePreview}
            />
        </div>
    )
}
