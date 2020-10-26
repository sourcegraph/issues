import React from 'react'
import { Dialog } from '@reach/dialog'
import { splitExtensionID } from './extension/extension'

/**
 * A modal confirmation prompt to the user confirming whether to add an extension.
 */
export const ExtensionPermissionModal: React.FunctionComponent<{
    extensionID: string
    givePermission: () => void
    denyPermission: () => void
}> = ({ extensionID, denyPermission, givePermission }) => {
    const { name } = splitExtensionID(extensionID)
    const labelId = `label--permission-${extensionID}`

    return (
        <Dialog
            className="modal-body modal-body--centered p-4 rounded border"
            onDismiss={denyPermission}
            aria-labelledBy={labelId}
        >
            <div className="web-content">
                <h3 id={labelId}>Add {name || extensionID} Sourcegraph extension?</h3>
                <p className="mb-0 mt-3">It will be able to:</p>
                <ul className="list-dashed">
                    <li className="m-0">read repositories and files you view using Sourcegraph</li>
                    <li className="m-0">read and change your Sourcegraph settings</li>
                </ul>
                <div className="d-flex justify-content-end pt-5">
                    <button type="button" className="btn btn-outline-secondary mr-2" onClick={denyPermission}>
                        Cancel
                    </button>
                    <button type="button" className="btn btn-primary" onClick={givePermission}>
                        Yes, add {name || extensionID}!
                    </button>
                </div>
            </div>
        </Dialog>
    )
}
