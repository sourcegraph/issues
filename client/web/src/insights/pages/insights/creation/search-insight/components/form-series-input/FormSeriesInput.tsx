import classnames from 'classnames'
import React from 'react'
import { noop } from 'rxjs'

import { FormInput } from '../../../../../../components/form/form-input/FormInput'
import { useField } from '../../../../../../components/form/hooks/useField'
import { useForm } from '../../../../../../components/form/hooks/useForm'
import { createRequiredValidator } from '../../../../../../components/form/validators'
import { DataSeries } from '../../../../../../core/backend/types'
import { DEFAULT_ACTIVE_COLOR, FormColorInput } from '../form-color-input/FormColorInput'

const requiredNameField = createRequiredValidator('Name is a required field for data series.')
const validQuery = createRequiredValidator('Query is a required field for data series.')

interface FormSeriesInputProps {
    /** Series index. */
    index: number
    /**
     * Show all validation error of all fields within the form.
     * */
    showValidationErrorsOnMount?: boolean
    /** Name of series. */
    name?: string
    /** Query value of series. */
    query?: string
    /** Color value for line chart. (series) */
    stroke?: string
    /** Enable autofocus behavior of first input of form. */
    autofocus?: boolean
    /** Enable cancel button. */
    cancel?: boolean
    /** Custom class name for root element of form series. */
    className?: string
    /** On submit handler of series form. */
    onSubmit?: (series: DataSeries) => void
    /** On cancel handler. */
    onCancel?: () => void
    /** Change handler in order to listen last values of series form. */
    onChange?: (formValues: DataSeries, valid: boolean) => void
}

/** Displays form series input (three field - name field, query field and color picker). */
export const FormSeriesInput: React.FunctionComponent<FormSeriesInputProps> = props => {
    const {
        index,
        showValidationErrorsOnMount = false,
        name,
        query,
        stroke: color,
        className,
        cancel = false,
        autofocus = true,
        onCancel = noop,
        onSubmit = noop,
        onChange = noop,
    } = props

    const hasNameControlledValue = !!name
    const hasQueryControlledValue = !!query

    const { formAPI, handleSubmit, ref } = useForm({
        touched: showValidationErrorsOnMount,
        initialValues: {
            seriesName: name ?? '',
            seriesQuery: query ?? '',
            seriesColor: color ?? DEFAULT_ACTIVE_COLOR,
        },
        onSubmit: values =>
            onSubmit({
                name: values.seriesName,
                query: values.seriesQuery,
                stroke: values.seriesColor,
            }),
        onChange: event => {
            const { values } = event

            onChange(
                {
                    name: values.seriesName,
                    query: values.seriesQuery,
                    stroke: values.seriesColor,
                },
                event.valid
            )
        },
    })

    const nameField = useField({
        name: 'seriesName',
        formApi: formAPI,
        validators: { sync: requiredNameField },
    })

    const queryField = useField({
        name: 'seriesQuery',
        formApi: formAPI,
        validators: { sync: validQuery },
    })

    const colorField = useField({
        name: 'seriesColor',
        formApi: formAPI,
    })

    return (
        <div data-testid="series-form" ref={ref} className={classnames('d-flex flex-column', className)}>
            <FormInput
                title="Name"
                required={true}
                autoFocus={autofocus}
                placeholder="Example: Function component"
                description="Name shown in the legend and tooltip"
                valid={(hasNameControlledValue || nameField.meta.touched) && nameField.meta.validState === 'VALID'}
                error={nameField.meta.touched && nameField.meta.error}
                {...nameField.input}
            />

            <FormInput
                title="Search query"
                required={true}
                placeholder="Example: patternType:regexp const\s\w+:\s(React\.)?FunctionComponent"
                description={
                    <span>
                        Do not include the <code>repo:</code> filter as it will be added automatically for the
                        repositories you included above.
                    </span>
                }
                valid={(hasQueryControlledValue || queryField.meta.touched) && queryField.meta.validState === 'VALID'}
                error={queryField.meta.touched && queryField.meta.error}
                className="mt-4"
                {...queryField.input}
            />

            <FormColorInput
                name={`color group of ${index} series`}
                title="Color"
                className="mt-4"
                value={colorField.input.value}
                onChange={colorField.input.onChange}
            />

            <div className="mt-4">
                <button
                    aria-label="Submit button for data series"
                    type="button"
                    onClick={handleSubmit}
                    className="btn btn-secondary"
                >
                    Done
                </button>

                {cancel && (
                    <button type="button" onClick={onCancel} className="btn btn-outline-secondary ml-2">
                        Cancel
                    </button>
                )}
            </div>
        </div>
    )
}
