import * as React from 'react'
import Close from 'mdi-react/CloseIcon'
import Check from 'mdi-react/CheckIcon'
import classnames from 'classnames'

interface Props {
    /** The initial value. */
    value?: boolean

    /** The DOM ID of the element. */
    id?: string

    /**
     * Called when the user changes the input's value.
     */
    onToggle?: (value: boolean) => void

    /**
     * Called when the user hovers over the toggle.
     */
    onHover?: (value: boolean) => void

    /** The title attribute (tooltip). */
    title?: string

    disabled?: boolean
    tabIndex?: number
    className?: string

    /** Data attribute for testing */
    dataTest?: string
}

/** A big toggle switch input component. */
export const ToggleBig: React.FunctionComponent<Props> = ({
    disabled,
    className,
    id,
    title,
    value,
    tabIndex,
    onToggle,
    onHover,
    dataTest,
}) => {
    function onClick(): void {
        if (!disabled && onToggle) {
            onToggle(!value)
        }
    }

    function onMouseOver(): void {
        if (onHover) {
            onHover(!value)
        }
    }

    return (
        <button
            type="button"
            className={classnames('toggle-big toggle-big__container', className)}
            id={id}
            title={title}
            value={value ? 1 : 0}
            onClick={onClick}
            tabIndex={tabIndex}
            onMouseOver={onMouseOver}
            disabled={!!disabled}
            role="switch"
            aria-checked={value}
            data-test={dataTest}
        >
            <span
                className={classnames('toggle-big__bar', {
                    'toggle-big__bar--on': value,
                })}
            />
            <span
                className={classnames('toggle-big__bar-shadow', {
                    'toggle-big__bar-shadow--on': value,
                })}
            />
            <span
                className={classnames('toggle-big__knob d-flex flex-column justify-content-center align-items-center', {
                    'toggle-big__knob--on': value,
                })}
            >
                {value ? (
                    <Check size={16} className="toggle-big__icon--on" />
                ) : (
                    <Close size={16} className="toggle-big__icon" />
                )}
            </span>
            <span className={classnames('toggle-big__text', { 'toggle-big__text--on': value })}>
                {value ? 'Enabled' : 'Disabled'}
            </span>
        </button>
    )
}
