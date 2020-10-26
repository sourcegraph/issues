import * as React from 'react'
import { Link } from 'react-router-dom'
import SourceRepositoryIcon from 'mdi-react/SourceRepositoryIcon'
import { TreeLayerProps } from './TreeLayer'
import { maxEntries, treePadding } from './util'

interface FileProps extends TreeLayerProps {
    className: string
    maxEntries: number
    handleTreeClick: () => void
    noopRowClick: (event: React.MouseEvent<HTMLAnchorElement>) => void
    linkRowClick: (event: React.MouseEvent<HTMLAnchorElement>) => void
}

export const File: React.FunctionComponent<FileProps> = props => (
    <tr key={props.entryInfo.path} className={props.className}>
        <td className="tree__cell">
            {props.entryInfo.submodule ? (
                props.entryInfo.url ? (
                    <Link
                        to={props.entryInfo.url}
                        onClick={props.linkRowClick}
                        draggable={false}
                        title={'Submodule: ' + props.entryInfo.submodule.url}
                        className="tree__row-contents"
                        data-tree-path={props.entryInfo.path}
                    >
                        <div className="tree__row-contents-text">
                            <span
                                // needed because of dynamic styling
                                // eslint-disable-next-line react/forbid-dom-props
                                style={treePadding(props.depth, true)}
                                className="tree__row-icon"
                                onClick={props.noopRowClick}
                                tabIndex={-1}
                            >
                                <SourceRepositoryIcon className="icon-inline" />
                            </span>
                            <span className="tree__row-label">
                                {props.entryInfo.name} @ {props.entryInfo.submodule.commit.slice(0, 7)}
                            </span>
                        </div>
                    </Link>
                ) : (
                    <div className="tree__row-contents" title={'Submodule: ' + props.entryInfo.submodule.url}>
                        <div className="tree__row-contents-text">
                            <span
                                className="tree__row-icon"
                                // needed because of dynamic styling
                                // eslint-disable-next-line react/forbid-dom-props
                                style={treePadding(props.depth, true)}
                            >
                                <SourceRepositoryIcon className="icon-inline" />
                            </span>
                            <span className="tree__row-label">
                                {props.entryInfo.name} @ {props.entryInfo.submodule.commit.slice(0, 7)}
                            </span>
                        </div>
                    </div>
                )
            ) : (
                <Link
                    className="tree__row-contents test-tree-file-link"
                    to={props.entryInfo.url}
                    onClick={props.linkRowClick}
                    data-tree-path={props.entryInfo.path}
                    draggable={false}
                    title={props.entryInfo.path}
                    // needed because of dynamic styling
                    style={treePadding(props.depth, false)}
                    tabIndex={-1}
                >
                    {props.entryInfo.name}
                </Link>
            )}
            {props.index === maxEntries - 1 && (
                <div
                    className="tree__row-alert alert alert-warning"
                    // needed because of dynamic styling
                    // eslint-disable-next-line react/forbid-dom-props
                    style={treePadding(props.depth, true)}
                >
                    Too many entries. Use search to find a specific file.
                </div>
            )}
        </td>
    </tr>
)
