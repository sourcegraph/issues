import * as React from 'react'
import { CardElement, ReactStripeElements } from 'react-stripe-elements'
import { ThemeProps } from '../../../../../shared/src/theme'

interface Props extends ThemeProps {
    disabled?: boolean
}

// Workaround for @types/stripe-v3 missing the new disabled attribute. See
// https://github.com/stripe/react-stripe-elements/issues/136#issuecomment-424984951.
type PatchedElementProps = ReactStripeElements.ElementProps & { disabled?: boolean }
const PatchedCardElement: React.FunctionComponent<PatchedElementProps> = props => <CardElement {...props} />

/**
 * Displays a payment form control for the user to enter payment information to purchase a product subscription.
 */
export const PaymentTokenFormControl: React.FunctionComponent<Props> = props => {
    const textColor = props.isLightTheme ? '#2b3750' : 'white'

    return (
        <div className="payment-token-form-control">
            <PatchedCardElement
                className={`payment-token-form-control__card payment-token-form-control__card--${
                    props.disabled ? 'disabled' : ''
                }`}
                disabled={props.disabled}
                style={{
                    base: {
                        fontFamily:
                            '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
                        color: textColor,
                        ':-webkit-autofill': {
                            color: textColor,
                        },
                    },
                }}
            />
        </div>
    )
}
