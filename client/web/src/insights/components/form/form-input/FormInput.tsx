import classnames from 'classnames'
import React, { forwardRef, InputHTMLAttributes, ReactNode } from 'react'

import { LoaderInput } from '@sourcegraph/branded/src/components/LoaderInput'

import styles from './FormInput.module.scss'
import { ForwardReferenceComponent } from './types'

interface FormInputProps extends InputHTMLAttributes<HTMLInputElement> {
    /** Title of input. */
    title?: string
    /** Description block for field. */
    description?: ReactNode
    /** Custom class name for root label element. */
    className?: string
    /** Error massage for input. */
    error?: string
    /** Prop to control error input element state. */
    errorInputState?: boolean
    /** Valid sign to show valid state on input. */
    valid?: boolean
    /** Turn on loading state (visually this is an input with loader) */
    loading?: boolean
    /** Turn on or turn off autofocus for input. */
    autofocus?: boolean
    /** Custom class name for input element. */
    inputClassName?: string
    /** Input icon (symbol) which render right after the input element. */
    inputSymbol?: ReactNode
}

/**
 * Displays the input with description, error message, visual invalid and valid states.
 */
export const FormInput = forwardRef((props, reference) => {
    const {
        as: Component = 'input',
        type = 'text',
        title,
        description,
        className,
        inputClassName,
        inputSymbol,
        valid,
        error,
        loading = false,
        errorInputState,
        ...otherProps
    } = props

    return (
        <label className={classnames('w-100', className)}>
            {title && <div className="mb-2">{title}</div>}

            <LoaderInput className="d-flex" loading={loading}>
                <Component
                    type={type}
                    className={classnames(styles.input, inputClassName, 'form-control', 'with-invalid-icon', {
                        'is-valid': valid,
                        'is-invalid': !!error || errorInputState,
                    })}
                    {...otherProps}
                    ref={reference}
                />

                {inputSymbol}
            </LoaderInput>

            {error && (
                <small role="alert" className="text-danger form-text">
                    {error}
                </small>
            )}
            {!error && description && (
                <small className={classnames('text-muted', 'form-text', styles.description)}>{description}</small>
            )}
        </label>
    )
}) as ForwardReferenceComponent<'input', FormInputProps>
