import * as React from 'react'
import { Omit } from 'utility-types'
import * as GQL from '../../../../shared/src/graphql/schema'
import { getModeFromPath } from '../../../../shared/src/languages'
import { ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import { Connection, FilteredConnection } from '../FilteredConnection'
import { FileDiffNodeProps } from './FileDiffNode'

class FilteredFileDiffConnection extends FilteredConnection<
    GQL.IFileDiff | GQL.IPreviewFileDiff,
    Omit<FileDiffNodeProps, 'node'>
> {}

type Props = FilteredFileDiffConnection['props']

/**
 * Displays a list of file diffs.
 */
export class FileDiffConnection extends React.PureComponent<Props> {
    public render(): JSX.Element | null {
        return <FilteredFileDiffConnection {...this.props} onUpdate={this.onUpdate} />
    }

    private onUpdate = (
        fileDiffsOrError: Connection<GQL.IFileDiff | GQL.IPreviewFileDiff> | ErrorLike | undefined
    ): void => {
        const nodeProps = this.props.nodeComponentProps!

        // TODO(sqs): This reports to extensions that these files are empty. This is wrong, but we don't have any
        // easy way to get the files' full contents here (and doing so would be very slow). Improve the extension
        // API's support for diffs.
        const dummyText = ''

        if (nodeProps.hovers) {
            nodeProps.hovers.extensionsController.services.editor.removeAllEditors()

            if (fileDiffsOrError && !isErrorLike(fileDiffsOrError)) {
                for (const fileDiff of fileDiffsOrError.nodes) {
                    if (fileDiff.oldPath && nodeProps.hovers) {
                        const uri = `git://${nodeProps.hovers.base.repoName}?${nodeProps.hovers.base.commitID}#${fileDiff.oldPath}`
                        if (!nodeProps.hovers.extensionsController.services.model.hasModel(uri)) {
                            nodeProps.hovers.extensionsController.services.model.addModel({
                                uri,
                                languageId: getModeFromPath(fileDiff.oldPath),
                                text: dummyText,
                            })
                        }
                        nodeProps.hovers.extensionsController.services.editor.addEditor({
                            type: 'CodeEditor',
                            resource: uri,
                            selections: [],
                            isActive: false, // HACK: arbitrarily say that the base is inactive. TODO: support diffs first-class
                        })
                    }
                    if (fileDiff.newPath && nodeProps.hovers) {
                        const uri = `git://${nodeProps.hovers.head.repoName}?${nodeProps.hovers.head.commitID}#${fileDiff.newPath}`
                        if (!nodeProps.hovers.extensionsController.services.model.hasModel(uri)) {
                            nodeProps.hovers.extensionsController.services.model.addModel({
                                uri,
                                languageId: getModeFromPath(fileDiff.newPath),
                                text: dummyText,
                            })
                        }
                        nodeProps.hovers.extensionsController.services.editor.addEditor({
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
