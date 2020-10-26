import * as React from 'react'
import { Omit } from 'utility-types'
import { getModeFromPath } from '../../../../shared/src/languages'
import { ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import { Connection, FilteredConnection } from '../FilteredConnection'
import { FileDiffNodeProps } from './FileDiffNode'
import { FileDiffFields } from '../../graphql-operations'

class FilteredFileDiffConnection extends FilteredConnection<FileDiffFields, Omit<FileDiffNodeProps, 'node'>> {}

type Props = FilteredFileDiffConnection['props']

/**
 * Displays a list of file diffs.
 */
export class FileDiffConnection extends React.PureComponent<Props> {
    public render(): JSX.Element | null {
        return <FilteredFileDiffConnection {...this.props} onUpdate={this.onUpdate} />
    }

    private onUpdate = (fileDiffsOrError: Connection<FileDiffFields> | ErrorLike | undefined): void => {
        const nodeProps = this.props.nodeComponentProps!

        // TODO(sqs): This reports to extensions that these files are empty. This is wrong, but we don't have any
        // easy way to get the files' full contents here (and doing so would be very slow). Improve the extension
        // API's support for diffs.
        const dummyText = ''

        if (nodeProps.extensionInfo) {
            nodeProps.extensionInfo.extensionsController.services.viewer.removeAllViewers()

            if (fileDiffsOrError && !isErrorLike(fileDiffsOrError)) {
                for (const fileDiff of fileDiffsOrError.nodes) {
                    if (fileDiff.oldPath && nodeProps.extensionInfo) {
                        const uri = `git://${nodeProps.extensionInfo.base.repoName}?${nodeProps.extensionInfo.base.commitID}#${fileDiff.oldPath}`
                        if (!nodeProps.extensionInfo.extensionsController.services.model.hasModel(uri)) {
                            nodeProps.extensionInfo.extensionsController.services.model.addModel({
                                uri,
                                languageId: getModeFromPath(fileDiff.oldPath),
                                text: dummyText,
                            })
                        }
                        nodeProps.extensionInfo.extensionsController.services.viewer.addViewer({
                            type: 'CodeEditor',
                            resource: uri,
                            selections: [],
                            isActive: false, // HACK: arbitrarily say that the base is inactive. TODO: support diffs first-class
                        })
                    }
                    if (fileDiff.newPath && nodeProps.extensionInfo) {
                        const uri = `git://${nodeProps.extensionInfo.head.repoName}?${nodeProps.extensionInfo.head.commitID}#${fileDiff.newPath}`
                        if (!nodeProps.extensionInfo.extensionsController.services.model.hasModel(uri)) {
                            nodeProps.extensionInfo.extensionsController.services.model.addModel({
                                uri,
                                languageId: getModeFromPath(fileDiff.newPath),
                                text: dummyText,
                            })
                        }
                        nodeProps.extensionInfo.extensionsController.services.viewer.addViewer({
                            type: 'CodeEditor',
                            resource: uri,
                            selections: [],
                            isActive: true,
                        })
                    }
                }
            }
        }
    }
}
