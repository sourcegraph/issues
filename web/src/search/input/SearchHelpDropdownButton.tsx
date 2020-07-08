import ExternalLinkIcon from 'mdi-react/ExternalLinkIcon'
import HelpCircleOutlineIcon from 'mdi-react/HelpCircleOutlineIcon'
import React, { useCallback, useState } from 'react'
import { Menu, MenuButton, MenuPopover } from '@reach/menu-button'
import classNames from 'classnames'

/**
 * A dropdown button that shows a menu with reference documentation for Sourcegraph search query
 * syntax.
 */
export const SearchHelpDropdownButton: React.FunctionComponent<{ left?: boolean }> = left => {
    const [isOpen, setIsOpen] = useState(false)
    const toggleIsOpen = useCallback(() => setIsOpen(!isOpen), [isOpen])
    const documentationUrlPrefix = window.context?.sourcegraphDotComMode ? 'https://docs.sourcegraph.com' : '/help'
    return (
        <Menu>
            {({ isExpanded }) => (
                <>
                    <MenuButton
                        className="px-2 btn btn-link d-flex align-items-center cursor-pointer"
                        aria-label="Quick help for search"
                    >
                        <HelpCircleOutlineIcon className="icon-inline small" aria-hidden="true" />
                    </MenuButton>
                    <MenuPopover
                        className={classNames('dropdown', {
                            'd-flex': isExpanded,
                            show: isExpanded,
                        })}
                        portal={false}
                    >
                        <div
                            className={classNames('pb-0 dropdown-menu', {
                                'dropdown-menu-right': !left,
                                show: isExpanded,
                            })}
                        >
                            <div className="dropdown-header">
                                <strong>Search reference</strong>
                            </div>
                            <div className="dropdown-divider" />
                            <div className="dropdown-header">Finding matches:</div>
                            <ul className="list-unstyled px-2 mb-2">
                                <li>
                                    <span className="text-muted small">Regexp:</span>{' '}
                                    <code>
                                        <strong>(read|write)File</strong>
                                    </code>
                                </li>
                                <li>
                                    <span className="text-muted small">Exact:</span>{' '}
                                    <code>
                                        "<strong>fs.open(f)</strong>"
                                    </code>
                                </li>
                            </ul>
                            <div className="dropdown-divider" />
                            <div className="dropdown-header">Common search keywords:</div>
                            <ul className="list-unstyled px-2 mb-2">
                                <li>
                                    <code>
                                        repo:<strong>my/repo</strong>
                                    </code>
                                </li>
                                {window.context?.sourcegraphDotComMode && (
                                    <li>
                                        <code>
                                            repo:<strong>github.com/myorg/</strong>
                                        </code>
                                    </li>
                                )}
                                <li>
                                    <code>
                                        file:<strong>my/file</strong>
                                    </code>
                                </li>
                                <li>
                                    <code>
                                        lang:<strong>javascript</strong>
                                    </code>
                                </li>
                            </ul>
                            <div className="dropdown-divider" />
                            <div className="dropdown-header">Diff/commit search keywords:</div>
                            <ul className="list-unstyled px-2 mb-2">
                                <li>
                                    <code>type:diff</code> <em className="text-muted small">or</em>{' '}
                                    <code>type:commit</code>
                                </li>
                                <li>
                                    <code>
                                        after:<strong>"2 weeks ago"</strong>
                                    </code>
                                </li>
                                <li>
                                    <code>
                                        author:<strong>alice@example.com</strong>
                                    </code>
                                </li>
                                <li className="text-nowrap">
                                    <code>
                                        repo:<strong>r@*refs/heads/</strong>
                                    </code>{' '}
                                    <span className="text-muted small">(all branches)</span>
                                </li>
                            </ul>
                            <div className="dropdown-divider mb-0" />
                            <a
                                // eslint-disable-next-line react/jsx-no-target-blank
                                target="_blank"
                                rel="noopener"
                                href={`${documentationUrlPrefix}/user/search/queries`}
                                className="dropdown-item"
                                onClick={toggleIsOpen}
                            >
                                <ExternalLinkIcon className="icon-inline small" /> All search keywords
                            </a>
                            {window.context?.sourcegraphDotComMode && (
                                <div className="p-2 alert alert-info small rounded-0 mb-0 mt-1">
                                    On Sourcegraph.com, use a <code>repo:</code> filter to narrow your search to &le;500
                                    repositories.
                                </div>
                            )}
                        </div>
                    </MenuPopover>
                </>
            )}
        </Menu>
    )
}
