import * as H from 'history'
import * as React from 'react'
import { Controller as ClientController } from '../../../../shared/src/extensions/controller'
import { ExtensionStatusPopover } from '../../../../shared/src/extensions/ExtensionStatus'
import { PlatformContextProps } from '../../../../shared/src/platform/context'
import { ShortcutProvider } from './ShortcutProvider'

interface Props extends PlatformContextProps<'sideloadedExtensionURL'> {
    location: H.Location
    extensionsController: ClientController
    sourcegraphURL: string
}

const makeExtensionLink = (sourcegraphURL: string): React.FunctionComponent<{ id: string }> => props => {
    const extensionURL = new URL(sourcegraphURL)
    extensionURL.pathname = `extensions/${props.id}`
    return <a href={extensionURL.href}>{props.id}</a>
}

/**
 * A global debug toolbar shown in the bottom right of the window.
 */
export const GlobalDebug: React.FunctionComponent<Props> = props => (
    <div className="global-debug navbar navbar-expand">
        <div className="navbar-nav align-items-center">
            <div className="nav-item">
                <ShortcutProvider>
                    <ExtensionStatusPopover
                        extensionsController={props.extensionsController}
                        link={makeExtensionLink(props.sourcegraphURL)}
                        platformContext={props.platformContext}
                    />
                </ShortcutProvider>
            </div>
        </div>
    </div>
)
