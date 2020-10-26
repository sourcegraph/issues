/**
 * The publishable (non-secret) API key for the billing system.
 */
export const billingPublishableKey: string | undefined = window.context?.billingPublishableKey

/**
 * Feature flag for showing Sourcegraph.com subscriptions, licensing, and billing features.
 */
export const SHOW_BUSINESS_FEATURES = Boolean(window.context?.sourcegraphDotComMode || billingPublishableKey)
