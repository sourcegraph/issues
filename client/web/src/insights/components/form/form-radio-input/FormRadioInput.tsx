import classnames from 'classnames'
import React, { InputHTMLAttributes } from 'react'

interface RadioInputProps extends InputHTMLAttributes<HTMLInputElement> {
    /** Title of radio input. */
    title: string
    /** Description text for radio input. */
    description?: string
    /** Custom class name for root label element. */
    className?: string
    /** Tooltip text for radio label element. */
    labelTooltipText?: string
    /** Tooltip position */
    labelTooltipPosition?: string
}

/** Displays form radio input for code insight creation form. */
export const FormRadioInput: React.FunctionComponent<RadioInputProps> = props => {
    const { title, description, className, labelTooltipText, labelTooltipPosition, ...otherProps } = props

    return (
        <label
            data-placement={labelTooltipPosition}
            data-tooltip={labelTooltipText}
            className={classnames('d-flex flex-wrap align-items-center', className, {
                'text-muted': otherProps.disabled,
            })}
        >
            <input type="radio" {...otherProps} />

            <span className="pl-2">{title}</span>

            {description && (
                <>
                    <span className="pl-2 pr-2">—</span>
                    <span className="text-muted">{description}</span>
                </>
            )}
        </label>
    )
}
