import {
    ChangeEvent,
    FocusEventHandler,
    InputHTMLAttributes,
    RefObject,
    useCallback,
    useEffect,
    useLayoutEffect,
    useRef,
    useState,
} from 'react'
import { noop } from 'rxjs'

import { FieldState, FormAPI, ValidationResult } from './useForm'
import { getEventValue } from './utils/get-event-value'
import { AsyncValidator, useAsyncValidation } from './utils/use-async-validation'

export type Validator<FieldValue> = (value: FieldValue | undefined, validity?: ValidityState | null) => ValidationResult

export interface Validators<FieldValue> {
    sync?: Validator<FieldValue>
    async?: AsyncValidator<FieldValue>
}

/**
 * Subset of native input props that useField can set to the native input element.
 */
interface InputProps<Value> extends Omit<InputHTMLAttributes<HTMLInputElement>, 'name' | 'value' | 'onChange'> {
    onChange?: (value: Value) => void
}

/**
 * Public API for input element. Contains all handlers and props for
 * native input element and expose meta state of input like touched,
 * validState and etc.
 * */
export interface useFieldAPI<FieldValue> {
    /**
     * Props and handles which should be passed to input component in order
     * to track change of value, set controlled value and set errors by
     * Constraint validation API.
     * */
    input: {
        ref: RefObject<HTMLInputElement & HTMLFieldSetElement>
        name: string
        value: FieldValue
        onChange: (event: ChangeEvent<HTMLInputElement> | FieldValue) => void
        onBlur: FocusEventHandler<HTMLInputElement>
    } & InputProps<FieldValue>
    /**
     * Meta state of form field - like touched, valid state and last
     * native validity state.
     * */
    meta: FieldState<FieldValue> & {
        /**
         * Set state handler gives access to set inner state of useField.
         * Useful for complicated cases when need to deal with custom react input
         * component.
         * */
        setState: (dispatch: (previousState: FieldState<FieldValue>) => FieldState<FieldValue>) => void
    }
}

export type UseFieldProps<FormValues, Key, Value> = {
    name: Key
    formApi: FormAPI<FormValues>
    validators?: Validators<Value>
} & InputProps<Value>

/**
 * React hook to manage validation of a single form input field.
 * `useInputValidation` helps with coordinating the constraint validation API
 * and custom synchronous and asynchronous validators.
 *
 * Should be used with useForm hook to connect field and form component's states.
 * */
export function useField<FormValues, Key extends keyof FormAPI<FormValues>['initialValues']>(
    props: UseFieldProps<FormValues, Key, FormValues[Key]>
): useFieldAPI<FormValues[Key]> {
    const { formApi, name, validators, onChange = noop, ...inputProps } = props
    const { setFieldState, initialValues, submitted, touched: formTouched } = formApi
    const { sync = noop, async } = validators ?? {}
    const inputReference = useRef<HTMLInputElement & HTMLFieldSetElement>(null)

    const [state, setState] = useState<FieldState<FormValues[Key]>>({
        value: initialValues[name],
        touched: false,
        validState: 'NOT_VALIDATED',
        error: '',
        validity: null,
    })

    const { start: startAsyncValidation, cancel: cancelAsyncValidation } = useAsyncValidation({
        inputReference,
        asyncValidator: async,
        onValidationChange: asyncState => setState(previousState => ({ ...previousState, ...asyncState })),
    })

    // Use useRef for form api handler in order to avoid unnecessary
    // calls if API handler has been changed.
    const setFieldStateReference = useRef<FormAPI<FormValues>['setFieldState']>(setFieldState)
    setFieldStateReference.current = setFieldState

    // Since validation logic wants to use sync state update we use `useLayoutEffect` instead of
    // `useEffect` in order to synchronously re-render after value setState updates, but before
    // the browser has painted DOM updates. This prevents users from seeing inconsistent states
    // where changes handled by React have been painted, but DOM manipulation handled by these
    // effects are painted on the next tick.
    useLayoutEffect(() => {
        const inputElement = inputReference.current

        // Clear custom validity from the last validation call.
        inputElement?.setCustomValidity?.('')

        const nativeAttributeValidation = inputElement?.checkValidity?.() ?? true
        const validity = inputElement?.validity ?? null

        // If we got error from native attr validation (required, pattern, type)
        // we still run validator in order to get some custom error message for
        // standard validation error if validator doesn't provide message we fallback
        // on standard validationMessage string [1] (ex. Please fill in input.)
        const nativeErrorMessage = inputElement?.validationMessage ?? ''
        const customValidation = sync(state.value, validity)

        if (customValidation || !nativeAttributeValidation) {
            // We have to cancel async validation from previous call
            // if we got sync validation native or custom.
            cancelAsyncValidation()

            // [1] Custom error message or fallback on native error message
            const validationMessage = customValidation || nativeErrorMessage

            inputElement?.setCustomValidity?.(validationMessage)

            return setState(state => ({
                ...state,
                validState: 'INVALID',
                error: validationMessage,
                validity,
            }))
        }

        if (async) {
            // Due the call of start async validation in useLayoutEffect we have to
            // schedule the async validation event in the next tick to be able run
            // observable pipeline validation since useAsyncValidation hook use
            // useObservable hook internally which calls '.subscribe' in useEffect.
            requestAnimationFrame(() => {
                startAsyncValidation({ value: state.value, validity })
            })

            return
        }

        return setState(state => ({
            ...state,
            validState: 'VALID' as const,
            error: '',
            validity,
        }))
    }, [state.value, sync, startAsyncValidation, async, cancelAsyncValidation])

    // Sync field state with state on form level - useForm hook will used this state to run
    // onSubmit handler and track validation state to prevent onSubmit run when async
    // validation is going.
    useEffect(() => setFieldStateReference.current(name, state), [name, state])

    const handleBlur = useCallback(() => setState(state => ({ ...state, touched: true })), [])
    const handleChange = useCallback(
        (event: ChangeEvent<HTMLInputElement> | FormValues[Key]) => {
            const value = getEventValue(event)

            setState(state => ({ ...state, value }))
            onChange(value)
        },
        [onChange]
    )

    return {
        input: {
            name: name.toString(),
            ref: inputReference,
            value: state.value,
            onBlur: handleBlur,
            onChange: handleChange,
            ...inputProps,
        },
        meta: {
            ...state,
            touched: state.touched || submitted || formTouched,
            /**
             * Set state dispatcher gives access to set inner state of useField.
             * Useful for complex cases when you need to deal with custom react input
             * components.
             * */
            setState: dispatch => {
                setState(state => ({ ...dispatch(state) }))
            },
        },
    }
}
