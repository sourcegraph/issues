import React, { useCallback } from 'react'
import { IProductPlan } from '../../../../../shared/src/graphql/schema'

interface Props {
    /** The user count input by the user. */
    value: number | null

    /** Called when the user count value changes. */
    onChange: (value: number | null) => void

    disabled?: boolean
    className?: string
    selectedPlan?: IProductPlan
}

/**
 * The minimum user count.
 */
export const MIN_USER_COUNT = 25

/**
 * The step size by which to increment/decrement user count.
 */
const USER_COUNT_STEP = 25

/**
 * Displays a form control for inputting the user count for a product subscription.
 */
export const ProductSubscriptionUserCountFormControl: React.FunctionComponent<Props> = ({
    value,
    onChange,
    disabled,
    className = '',
    selectedPlan,
}) => {
    const onUserCountChange = useCallback<React.ChangeEventHandler<HTMLSelectElement | HTMLInputElement>>(
        event => {
            // Check for NaN (which is the value if the user deletes the input's value).
            onChange(Number.isNaN(event.currentTarget.value) ? null : Number(event.currentTarget.value))
        },
        [onChange]
    )

    if (!selectedPlan) {
        return null
    }

    return (
        <div className={`product-subscription-user-count-control form-group align-items-center ${className}`}>
            <label htmlFor="product-subscription-user-count-control__userCount" className="mb-0 mr-2 font-weight-bold">
                Users
            </label>
            <div className="d-flex align-items-center">
                {selectedPlan.tiersMode === 'graduated' ? (
                    <select className="form-control w-auto" disabled={disabled} onChange={onUserCountChange}>
                        {selectedPlan.planTiers
                            .map(({ upTo }) => upTo)
                            // Currently filters out upTo 0. Only supporting fixed tiers
                            .filter(Boolean)
                            .map(userCount => (
                                <option value={userCount}>{userCount}</option>
                            ))}
                    </select>
                ) : (
                    <input
                        id="product-subscription-user-count-control__userCount"
                        type="number"
                        className="form-control w-auto"
                        min={MIN_USER_COUNT}
                        step={USER_COUNT_STEP}
                        max={50000}
                        required={true}
                        disabled={disabled}
                        value={value || ''}
                        onChange={onUserCountChange}
                    />
                )}
            </div>
        </div>
    )
}
