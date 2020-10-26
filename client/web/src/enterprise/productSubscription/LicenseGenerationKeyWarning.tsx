import React from 'react'

/**
 * Displays a warning in debug mode (which is on for local dev) that generated license keys aren't
 * actually valid. Local dev (in dev/start.sh) uses a test private key that does NOT correspond to
 * the license validation public key shipped with Sourcegraph instances.
 *
 * Technically it's possible to generate valid license keys in debug mode (if you manually configure
 * the right license generation private key), but it's not worth the complexity to make this alert
 * precise.
 */
export const LicenseGenerationKeyWarning: React.FunctionComponent<{ className?: string }> = ({ className = '' }) =>
    window.context?.debug ? (
        <div className={`alert alert-warning ${className}`}>
            License keys generated in dev mode are <strong>NOT VALID</strong>.{' '}
            <a href="https://sourcegraph.com/site-admin/dotcom/product/subscriptions">
                Use Sourcegraph.com to generate valid license keys.
            </a>
        </div>
    ) : null
