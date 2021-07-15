import { debounce, DebouncedFunc } from 'lodash'
import { EventHandler, FormEventHandler, RefObject, SyntheticEvent, useEffect, useMemo, useRef, useState } from 'react'
import { noop } from 'rxjs'

import { useDistinctValue } from '../../../hooks/use-distinct-value'

// Special key for the submit error store.
export const FORM_ERROR = 'useForm/submissionErrors'

export type SubmissionErrors = Record<string, any> | undefined
export type ValidationResult = string | undefined | void

export interface FormChangeEvent<FormValues> {
    values: FormValues
    valid: boolean
}

type ChangeHandler<FormValues> = (event: FormChangeEvent<FormValues>) => void

interface UseFormProps<FormValues extends object> {
    /**
     * Initial values for form fields.
     * */
    initialValues: FormValues

    /**
     * Mark all fields within the form as touched.
     * */
    touched?: boolean

    /**
     * Submit handler for a form element.
     * */
    onSubmit: (values: FormValues) => SubmissionErrors | Promise<SubmissionErrors> | void

    /**
     * Change handler will be called every time when some field withing the form
     * has been changed with last fields values.
     * */
    onChange?: ChangeHandler<FormValues>
}

/**
 * High level API for form instance. It consists with form api for useField hook,
 * form state, form handlers like handleSubmit and some props which should be
 * passed on root form element.
 *
 * */
export interface Form<FormValues> {
    /**
     * Values of all inputs in the form.
     * */
    values: Partial<FormValues>
    /**
     * State and methods of form, used in consumers to create filed by useField(formAPI)
     * */
    formAPI: FormAPI<FormValues>

    /**
     * Handler for onSubmit form element.
     * */
    handleSubmit: FormEventHandler | EventHandler<SyntheticEvent>

    /**
     * Ref for the root element of form. It might be form or any html element.
     * Used to find first invalid input within this root element and call focus
     * on it.
     * */
    ref: RefObject<any>
}

/**
 * This API should be passed to useField hook for registration they state
 * to the form object from useForm hook. Also this api consists form state
 * like submitting, submitted, validation.
 * */
export interface FormAPI<FormValues> {
    /**
     * Initial values for the form.
     * These values are set for inputs only in first render.
     * Initial values also used as field value for first run
     * of sync and async validators on useField level.
     * */
    initialValues: FormValues

    /**
     * Mark to understand was there an attempt by user to submit the form?
     * Used in useField hook to trigger appearance of error message if
     * user tried submit the form.
     * */
    submitted: boolean

    valid: boolean

    /**
     * State to understand there some field is processing async validations.
     * It might be useful to disable submit button for example if we got async
     * validation for form filed.
     * */
    validating: boolean

    /**
     * State to understand that form submitting is going on.
     * Also might be used as a sign to disable or show loading
     * state for submit button.
     * */
    submitting: boolean

    /** Store for submit errors which we got from onSubmit prop handler */
    submitErrors: SubmissionErrors

    /**
     * This prop marks all fields within the form as touched.
     * This might be useful when you need trigger touched of all fields
     * programmatically (edit mode for forms scenario)
     * */
    touched: boolean

    /**
     * Public api for register fields to the form from useField hook.
     * By this we have field state withing useField hook and in useField.
     * */
    setFieldState: (name: keyof FormValues, state: FieldState<unknown>) => void
}

/**
 * Field state which present public state from useField hook. On order to aggregate
 * state of all fields within the form we store all fields state on form level as well.
 * */
export interface FieldState<Value> extends FieldMetaState {
    /**
     * Field (input) controlled value. This value might be not only some primitive value
     * like string, number but array, object, tuple and other complex types as consumer set.
     * */
    value: Value
}

export interface FieldMetaState {
    /**
     * State to understand when users focused and blurred input element.
     * */
    touched: boolean

    /**
     * Valid state with initial value NOT_VALIDATED, with VALID when all validators
     * didn't return validation error, CHECKING for when async validation is going on,
     * and INVALID when some validator returns validation error.
     * */
    validState: 'VALID' | 'INVALID' | 'NOT_VALIDATED' | 'CHECKING'

    /**
     * Last error value which has been returned from validators.
     * */
    error?: any

    /**
     * Native validity state from native validation API of input element.
     * Null when useField is used for some custom elements instead of native input.
     * */
    validity: ValidityState | null
}

/**
 * Store object of all fields state within the form element.
 * Used below to keep tracking of fields value, touched, validity and other
 * fields state data.
 * */
type FieldsState<FormValues> = Record<keyof FormValues, FieldState<FormValues>>

/**
 * Unified form abstraction to track form state and provide form fields management
 * React hook to have all needed state for building proper UX for forms.
 *
 * useForm is one of two hooks for form management which responsible for
 * form state - submitted, submitting, state of all form fileds from useField
 * hook.
 * */
export function useForm<FormValues extends object>(props: UseFormProps<FormValues>): Form<FormValues> {
    const { onSubmit, initialValues, touched = false, onChange = noop } = props

    const [submitted, setSubmitted] = useState(false)
    const [submitting, setSubmitting] = useState(false)
    const [submitErrors, setSubmitErrors] = useState<SubmissionErrors>()
    const [fields, setFields] = useState<FieldsState<FormValues>>({} as FieldsState<FormValues>)

    const formElementReference = useRef<HTMLFormElement>(null)
    const onSubmitReference = useRef<UseFormProps<FormValues>['onSubmit']>()

    // Debounced onChange handler.
    const onChangeReference = useRef<DebouncedFunc<ChangeHandler<FormValues>>>(debounce(onChange, 0))

    // Track unmounted state to prevent setState if async validation or async submitting
    // will be resolved after component has been unmounted.
    const isUnmounted = useRef<boolean>(false)

    const setFieldState = (name: keyof FormValues, state: FieldState<unknown>): void => {
        setFields(fields => ({ ...fields, [name]: state }))
    }

    const values = useMemo(() => getFormValues<FormValues>(fields), [fields])

    const changeEvent = useDistinctValue(
        useMemo<{ values: FormValues; valid: boolean }>(
            () => ({
                values: getFormValues(fields),
                valid: Object.values<Pick<FieldState<unknown>, 'validState'>>(fields).every(
                    state => state.validState === 'VALID'
                ),
            }),
            [fields]
        )
    )

    useEffect(() => {
        if (Object.keys(changeEvent.values).length === 0) {
            return
        }

        onChangeReference.current?.(changeEvent)
    }, [changeEvent])

    useEffect(
        () => () => {
            onChangeReference.current?.cancel()
            isUnmounted.current = true
        },
        []
    )

    // Mutate local ref for submit handler to allow pass onSubmit
    // handler without memo.
    onSubmitReference.current = onSubmit

    return {
        values,
        formAPI: {
            submitted,
            touched,
            submitting,
            submitErrors,
            initialValues,
            setFieldState,
            valid: Object.values<FieldState<unknown>>(fields).every(state => state.validState === 'VALID'),
            validating: Object.values<FieldState<unknown>>(fields).some(state => state.validState === 'CHECKING'),
        },
        ref: formElementReference,
        handleSubmit: async event => {
            event.preventDefault()

            setSubmitted(true)

            const hasInvalidField = Object.values<FieldState<unknown>>(fields).some(
                state => state.validState === 'INVALID'
            )

            if (!hasInvalidField) {
                setSubmitting(true)

                const submitResult = await onSubmitReference.current?.(values)

                // Check isUnmounted state to prevent calling setState on
                // unmounted components.
                if (!isUnmounted.current) {
                    setSubmitting(false)
                    // eslint-disable-next-line no-unused-expressions
                    submitResult && setSubmitErrors(submitResult)
                }
            } else {
                // Hack to focus first invalid input on submit, since we are not using
                // native behavior in order to avoid poor UX of native validation focus on error
                // we have to find and focus invalid input by ourselves
                formElementReference.current?.querySelector<HTMLInputElement>(':invalid:not(fieldset)')?.focus()
            }
        },
    }
}

/**
 * Creates form values object and omits all other internal states of a form field.
 * Used to form values for onSubmit and onChange handlers.
 * */
export function getFormValues<FormValues>(fields: Record<string, Pick<FieldState<FormValues>, 'value'>>): FormValues {
    return Object.keys(fields).reduce(
        (values, fieldName) => ({ ...values, [fieldName]: fields[fieldName].value }),
        {} as FormValues
    )
}
