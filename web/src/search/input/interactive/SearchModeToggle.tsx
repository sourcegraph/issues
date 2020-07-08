import KeyboardIcon from 'mdi-react/KeyboardIcon'
import ViewQuiltIcon from 'mdi-react/ViewQuiltIcon'
import * as React from 'react'
import { Menu, MenuButton, MenuList, MenuItem, MenuLink } from '@reach/menu-button'
import classNames from 'classnames'
import { Link } from 'react-router-dom'

interface Props {
    interactiveSearchMode: boolean
    toggleSearchMode: () => void
}

const noop: () => void = () => {}

export const SearchModeToggle: React.FunctionComponent<Props> = props => (
    <Menu>
        {({ isExpanded }) => (
            <div className="search-mode-toggle dropdown">
                <MenuButton
                    className="search-mode-toggle__button btn btn-secondary e2e-search-mode-toggle dropdown-toggle"
                    aria-label="Toggle search mode"
                >
                    {props.interactiveSearchMode ? (
                        <ViewQuiltIcon className="icon-inline" size={8} />
                    ) : (
                        <KeyboardIcon className="icon-inline" size={8} />
                    )}
                </MenuButton>
                <MenuList className={classNames('search-mode-toggle dropdown-menu', { show: isExpanded })}>
                    <MenuItem
                        onSelect={!props.interactiveSearchMode ? props.toggleSearchMode : noop}
                        className={classNames(
                            'dropdown-item search-mode-toggle__item e2e-search-mode-toggle__interactive-mode',
                            {
                                active: props.interactiveSearchMode,
                            }
                        )}
                    >
                        <ViewQuiltIcon className="icon-inline" size={8} />
                        <span className="ml-1">Interactive mode</span>
                    </MenuItem>
                    <MenuItem
                        onSelect={props.interactiveSearchMode ? props.toggleSearchMode : noop}
                        className={classNames(
                            'dropdown-item search-mode-toggle__item  e2e-search-mode-toggle__plain-text-mode',
                            {
                                active: !props.interactiveSearchMode,
                            }
                        )}
                    >
                        <KeyboardIcon className="icon-inline" />
                        <span className="ml-1">Plain text mode</span>
                    </MenuItem>
                    <MenuLink as={Link} to="/search/query-builder" className="dropdown-item search-mode-toggle__item ">
                        Query builder&hellip;
                    </MenuLink>
                </MenuList>
            </div>
        )}
    </Menu>
)
