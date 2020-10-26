import FileDownloadIcon from 'mdi-react/FileDownloadIcon'
import * as React from 'react'
import { encodeRepoRevision, RepoSpec, RevisionSpec, FileSpec } from '../../../../shared/src/util/url'

interface Props extends RepoSpec, Partial<RevisionSpec>, FileSpec {}

/**
 * A repository header action that replaces the blob in the URL with the raw URL.
 */
export class GoToRawAction extends React.PureComponent<Props> {
    public render(): JSX.Element {
        const to = `/${encodeRepoRevision(this.props)}/-/raw/${this.props.filePath}`
        return (
            <a href={to} className="nav-link" data-tooltip="Raw (download file)" download={true}>
                <FileDownloadIcon className="icon-inline" />
            </a>
        )
    }
}
