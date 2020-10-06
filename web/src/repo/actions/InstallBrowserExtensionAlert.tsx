import React from 'react'
import CloseIcon from 'mdi-react/CloseIcon'
import ExportIcon from 'mdi-react/ExportIcon'
import * as GQL from '../../../../shared/src/graphql/schema'
import { serviceTypeDisplayNameAndIcon } from './GoToCodeHostAction'
import { IS_CHROME } from '../../marketing/util'

interface Props {
    onAlertDismissed: () => void
    externalURLs: GQL.IExternalLink[]
}

// TODO(tj): Add Firefox once the Firefox extension is back
const CHROME_EXTENSION_STORE_LINK = 'https://chrome.google.com/webstore/detail/dgjhfomjieaadpoljlnidmbgkdffpack'

export const InstallBrowserExtensionAlert: React.FunctionComponent<Props> = ({ onAlertDismissed, externalURLs }) => {
    const { serviceType } = externalURLs[0]
    const { displayName, icon } = serviceTypeDisplayNameAndIcon(serviceType)

    const Icon = icon || ExportIcon

    return (
        <div className="alert alert-info m-2 d-flex justify-content-between">
            <div className="d-flex align-items-center">
                <div className="position-relative">
                    <div className="install-browser-extension-alert__icon-flash" />
                    <Icon className="install-browser-extension-alert__icon" />
                </div>
                <p className="install-browser-extension-alert__text my-0 mr-3">
                    {IS_CHROME ? (
                        <>
                            <a
                                href={CHROME_EXTENSION_STORE_LINK}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="alert-link"
                            >
                                Install the Sourcegraph browser extension
                            </a>{' '}
                            to add code intelligence to {serviceType === 'gitlab' ? 'MR' : 'PR'}s and file views on{' '}
                            {displayName} or any other connected code host.
                        </>
                    ) : (
                        <>
                            Get code intelligence while browsing files and reading{' '}
                            {serviceType === 'gitlab' ? 'MR' : 'PR'}s on {displayName} or any other connected code host.{' '}
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
            <button type="button" onClick={onAlertDismissed} aria-label="Close alert" className="btn btn-icon">
                <CloseIcon className="icon-inline" />
            </button>
        </div>
    )
}
