import * as React from 'react'
import { withAuthenticatedUser } from '../../../auth/withAuthenticatedUser'
import { PageTitle } from '../../../components/PageTitle'

/** A page for publishing a new release of an extension to the extension registry. */
export const RegistryExtensionNewReleasePage = withAuthenticatedUser(() => (
    <div className="registry-extension-new-release-page">
        <PageTitle title="Publish new release" />
        <h2>Publish new release</h2>
        <p>
            Use the{' '}
            <a href="https://github.com/sourcegraph/src-cli" target="_blank" rel="noopener noreferrer">
                <code>src</code> CLI tool
            </a>{' '}
            to publish a new release:
        </p>
        <pre>
            <code>$ src extensions publish</code>
        </pre>
    </div>
))
