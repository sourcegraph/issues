import React from 'react'
import CloseIcon from 'mdi-react/CloseIcon'
import ExportIcon from 'mdi-react/ExportIcon'
import * as GQL from '../../../../shared/src/graphql/schema'
import { serviceTypeDisplayNameAndIcon } from './GoToCodeHostAction'

interface Props {
    onAlertDismissed: () => void
    externalURLs: GQL.IExternalLink[]
    isChrome: boolean
    codeHostIntegrationMessaging: 'browser-extension' | 'native-integration'
}

// TODO(tj): Add Firefox once the Firefox extension is back
const CHROME_EXTENSION_STORE_LINK = 'https://chrome.google.com/webstore/detail/dgjhfomjieaadpoljlnidmbgkdffpack'

/** Code hosts the browser extension supports */
const supportedServiceTypes = new Set<string>(['github', 'gitlab', 'phabricator', 'bitbucketServer'])

export const InstallBrowserExtensionAlert: React.FunctionComponent<Props> = ({
    onAlertDismissed,
    externalURLs,
    isChrome,
    codeHostIntegrationMessaging,
}) => {
    const externalLink = externalURLs.find(link => link.serviceType && supportedServiceTypes.has(link.serviceType))
    if (!externalLink) {
        return null
    }

    const { serviceType } = externalLink
    const { displayName, icon } = serviceTypeDisplayNameAndIcon(serviceType)

    const Icon = icon || ExportIcon

    return (
        <div className="alert alert-info m-2 d-flex justify-content-between install-browser-extension-alert">
            <div className="d-flex align-items-center">
                <div className="position-relative">
                    <div className="install-browser-extension-alert__icon-flash" />
                    <Icon className="install-browser-extension-alert__icon" />
                </div>
                <p className="install-browser-extension-alert__text my-0 mr-3">
                    {codeHostIntegrationMessaging === 'native-integration' ? (
                        <>
                            Sourcegraph's code intelligence will follow you to your code host. Your site admin set up
                            the Sourcegraph native integration for {displayName}.{' '}
                            <a
                                className="alert-link"
                                href="https://docs.sourcegraph.com/integration/browser_extension"
                                target="_blank"
                                rel="noopener"
                            >
                                Learn more
                            </a>{' '}
                            or{' '}
                            <a className="alert-link" href={externalLink.url} target="_blank" rel="noopener">
                                try it out
                            </a>
                        </>
                    ) : isChrome ? (
                        <>
                            <a
                                href={CHROME_EXTENSION_STORE_LINK}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="alert-link"
                            >
                                Install the Sourcegraph browser extension
                            </a>{' '}
                            to add code intelligence{' '}
                            {serviceType === 'github' ||
                            serviceType === 'bitbucketServer' ||
                            serviceType === 'gitlab' ? (
                                <>to {serviceType === 'gitlab' ? 'merge requests' : 'pull requests'} and file views</>
                            ) : (
                                <>while browsing and reviewing code</>
                            )}{' '}
                            on {displayName}.
                        </>
                    ) : (
                        <>
                            Get code intelligence{' '}
                            {serviceType === 'github' ||
                            serviceType === 'bitbucketServer' ||
                            serviceType === 'gitlab' ? (
                                <>
                                    while browsing files and reviewing{' '}
                                    {serviceType === 'gitlab' ? 'merge requests' : 'pull requests'}
                                </>
                            ) : (
                                <>while browsing and reviewing code</>
                            )}{' '}
                            on {displayName}.{' '}
                            <a
                                href="/help/integration/browser_extension"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="alert-link"
                            >
                                Learn more about Sourcegraph Chrome and Firefox extensions
                            </a>
                        </>
                    )}
                </p>
            </div>
            <button
                type="button"
                onClick={onAlertDismissed}
                aria-label="Close alert"
                className="btn btn-icon test-close-alert"
            >
                <CloseIcon className="icon-inline" />
            </button>
        </div>
    )
}
