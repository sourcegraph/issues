import { ChangeEvent, FocusEventHandler, useEffect, useRef, useState } from 'react'

import { FormAPI } from './useForm'

export interface UseCheckboxAPI<FieldValue> {
    input: {
        isChecked: (value: FieldValue) => boolean
        onChange: (event: ChangeEvent<HTMLInputElement>) => void
        onBlur: FocusEventHandler<HTMLInputElement>
    }
    values: string[]
}

interface CheckboxesState<FieldValue> {
    value: FieldValue[]
    touched: boolean
}

export function useCheckboxes<FormValues, FieldValueKey extends keyof FormAPI<FormValues>['initialValues']>(
    name: FieldValueKey,
    formApi: Omit<FormAPI<FormValues>, 'initialValues'> & { initialValues: { [_ in FieldValueKey]: string[] } }
): UseCheckboxAPI<string> {
    const { setFieldState, initialValues } = formApi
    const initialCheckboxesValue = initialValues[name]

    const [state, setState] = useState<CheckboxesState<string>>({
        value: initialCheckboxesValue,
        touched: false,
    })

    // Use useRef for form api handler in order to avoid unnecessary
    // calls if API handler has been changed.
    const setFieldStateReference = useRef<FormAPI<FormValues>['setFieldState']>(setFieldState)
    setFieldStateReference.current = setFieldState

    // Sync field state with the state on form level - useForm hook will use this state to run
    // onSubmit handler and track validation state to prevent onSubmit run when async
    // validation is going.
    useEffect(() => setFieldStateReference.current(name, { ...state, validState: 'VALID', validity: null }), [
        name,
        state,
    ])

    return {
        input: {
            isChecked: (value: string) => state.value.includes(value),
            onBlur: () => setState(state => ({ ...state, touched: true })),
            onChange: (event: ChangeEvent<HTMLInputElement>) => {
                const checkboxValue = event.target.value

                if (event.target.checked) {
                    setState(state => ({ ...state, value: [...state.value, checkboxValue] }))
                } else {
                    setState(state => ({ ...state, value: state.value.filter(value => value !== checkboxValue) }))
                }
            },
        },
        values: state.value,
    }
}
