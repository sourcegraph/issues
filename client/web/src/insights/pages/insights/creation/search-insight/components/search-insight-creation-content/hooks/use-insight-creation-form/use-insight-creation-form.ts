import { Settings } from '../../../../../../../../../schema/settings.schema'
import { useField, useFieldAPI } from '../../../../../../../../components/form/hooks/useField'
import { Form, FormChangeEvent, SubmissionErrors, useForm } from '../../../../../../../../components/form/hooks/useForm'
import { useInsightTitleValidator } from '../../../../../../../../components/form/hooks/useInsightTitleValidator'
import { InsightTypePrefix } from '../../../../../../../../core/types'
import { isUserSubject, SupportedInsightSubject } from '../../../../../../../../core/types/subjects'
import { CreateInsightFormFields, EditableDataSeries, InsightStep } from '../../../../types'
import { INITIAL_INSIGHT_VALUES } from '../../initial-insight-values'
import {
    repositoriesExistValidator,
    repositoriesFieldValidator,
    requiredStepValueField,
    seriesRequired,
} from '../../validators'

export interface UseInsightCreationFormProps {
    /**
     * Final (merged) settings cascade  objects
     */
    settings?: Settings | null

    /**
     * List of all supportable insight subjects
     */
    subjects?: SupportedInsightSubject[]

    /**
     * Initial value for all form fields
     */
    initialValue?: CreateInsightFormFields

    /**
     * Set initial touched state for all form fields.
     */
    touched?: boolean

    /**
     * Submit handler for form element.
     */
    onSubmit: (values: CreateInsightFormFields) => SubmissionErrors | Promise<SubmissionErrors> | void

    /**
     * Change handlers is called every time when user changed any field within the form.
     */
    onChange?: (event: FormChangeEvent<CreateInsightFormFields>) => void
}

export interface InsightCreationForm {
    form: Form<CreateInsightFormFields>

    title: useFieldAPI<string>
    repositories: useFieldAPI<string>
    visibility: useFieldAPI<string>
    series: useFieldAPI<EditableDataSeries[]>
    step: useFieldAPI<InsightStep>
    stepValue: useFieldAPI<string>
    allReposMode: useFieldAPI<boolean>
}

/**
 * Hooks absorbs all insight creation form logic (field state managements, validations, fields dependencies)
 */
export function useInsightCreationForm(props: UseInsightCreationFormProps): InsightCreationForm {
    const { subjects = [], initialValue, touched, settings, onSubmit, onChange } = props

    // Calculate initial value for visibility settings
    const userSubjectID = subjects.find(isUserSubject)?.id ?? ''

    const form = useForm<CreateInsightFormFields>({
        initialValues: initialValue ?? { ...INITIAL_INSIGHT_VALUES, visibility: userSubjectID },
        onSubmit,
        onChange,
        touched,
    })

    const allReposMode = useField({
        name: 'allRepos',
        formApi: form.formAPI,
        onChange: (checked: boolean) => {
            // Reset form values in case if All repos mode was activated
            if (checked) {
                repositories.input.onChange('')
                step.input.onChange('weeks')
                stepValue.input.onChange('2')
            }
        },
    })

    const isAllReposMode = allReposMode.input.value

    // We can't have two or more insights with the same name, since we rely on name as on id of insights.
    const titleValidator = useInsightTitleValidator({ settings, insightType: InsightTypePrefix.search })
    const title = useField({
        name: 'title',
        formApi: form.formAPI,
        validators: { sync: titleValidator },
    })

    const repositories = useField({
        name: 'repositories',
        formApi: form.formAPI,
        validators: {
            // Turn off any validations for the repositories field in we are in all repos mode
            sync: !isAllReposMode ? repositoriesFieldValidator : undefined,
            async: !isAllReposMode ? repositoriesExistValidator : undefined,
        },
        disabled: isAllReposMode,
    })

    const visibility = useField({
        name: 'visibility',
        formApi: form.formAPI,
    })

    const series = useField({
        name: 'series',
        formApi: form.formAPI,
        validators: { sync: seriesRequired },
    })

    const step = useField({
        name: 'step',
        formApi: form.formAPI,
        disabled: isAllReposMode,
    })
    const stepValue = useField({
        name: 'stepValue',
        formApi: form.formAPI,
        validators: {
            // Turn off any validations if we are in all repos mode
            sync: !isAllReposMode ? requiredStepValueField : undefined,
        },
        disabled: isAllReposMode,
    })

    return {
        form,
        title,
        repositories,
        visibility,
        series,
        step,
        stepValue,
        allReposMode,
    }
}
