import React, { useState, useEffect } from 'react'
import { Elements, injectStripe, StripeProvider, ReactStripeElements } from 'react-stripe-elements'
import { billingPublishableKey } from '../productSubscriptions/features'

type Props<P> = P & { component: React.ComponentType<P & ReactStripeElements.InjectedStripeProps> }

/**
 * Wraps a React tree (of elements) and injects the Stripe API.
 */
export const StripeWrapper = <P extends object>(props: Props<P>): JSX.Element | null => {
    const [stripe, setStripe] = useState<stripe.Stripe | null>(null)
    useEffect(() => {
        if (stripe) {
            return
        }

        const initStripe = (): void => setStripe((window as any).Stripe(billingPublishableKey))

        const id = 'stripe-script'
        if (document.querySelector(`#${id}`)) {
            initStripe()
            return // already loaded
        }

        const script = document.createElement('script')
        script.id = id
        script.src = 'https://js.stripe.com/v3/'
        script.async = true
        script.addEventListener('load', initStripe)
        document.body.append(script)
    }, [stripe])

    // Ensure that injectStripe gets called exactly once for each props.component, or else there
    // will be a re-render loop.
    const [lastComponent, setLastComponent] = useState<
        React.ComponentType<P & ReactStripeElements.InjectedStripeProps>
    >(() => props.component)
    const [InjectedComponent, setInjectedComponent] = useState<React.ComponentType<P>>(() =>
        injectStripe<P>(props.component)
    )
    useEffect(() => {
        if (props.component !== lastComponent) {
            setLastComponent(() => props.component)
            setInjectedComponent(() => injectStripe<P>(props.component))
        }
    }, [lastComponent, props.component])

    if (!stripe || !InjectedComponent) {
        return null
    }
    return (
        <StripeProvider stripe={stripe}>
            <Elements>
                <InjectedComponent {...props} />
            </Elements>
        </StripeProvider>
    )
}
