import classNames from 'classnames'
import ChevronDownIcon from 'mdi-react/ChevronDownIcon'
import ChevronLeftIcon from 'mdi-react/ChevronLeftIcon'
import React, { useEffect, useState } from 'react'
import { Collapse } from 'reactstrap'

import { FilterLink, FilterLinkProps } from './FilterLink'
import styles from './SearchSidebarSection.module.scss'

export const SearchSidebarSection: React.FunctionComponent<{
    header: string
    children?: React.ReactElement[] | ((filter: string) => React.ReactElement)
    className?: string
    showSearch?: boolean // Search only works if children are FilterLink
    onToggle?: (open: boolean) => void
    open?: boolean
}> = ({ header, children = [], className, showSearch = false, onToggle, open }) => {
    const [filter, setFilter] = useState('')

    // Clear filter when children change
    useEffect(() => setFilter(''), [children])

    let body
    let searchVisible = showSearch
    let visible = typeof children === 'function'

    if (typeof children === 'function') {
        body = children(filter)
    } else {
        visible = children.length > 0
        searchVisible = searchVisible && children.length > 1

        const filteredChildren = searchVisible
            ? children.filter(child => {
                  if (child.type === FilterLink) {
                      const props: FilterLinkProps = child.props as FilterLinkProps
                      return (
                          (props?.label).toLowerCase().includes(filter.toLowerCase()) ||
                          (props?.value).toLowerCase().includes(filter.toLowerCase())
                      )
                  }
                  return true
              })
            : children

        body = (
            <>
                <ul className={styles.sidebarSectionList}>
                    {filteredChildren.map((child, index) => (
                        <li key={child.key || index}>{child}</li>
                    ))}
                    {filteredChildren.length === 0 && (
                        <li className={classNames('text-muted', styles.sidebarSectionNoResults)}>No results</li>
                    )}
                </ul>
            </>
        )
    }

    const [collapsed, setCollapsed] = useState(!open)

    return visible ? (
        <div className={classNames(styles.sidebarSection, className)}>
            <button
                type="button"
                className={classNames('btn btn-outline-secondary', styles.sidebarSectionCollapseButton)}
                onClick={() =>
                    setCollapsed(collapsed => {
                        if (onToggle) {
                            onToggle(collapsed)
                        }
                        return !collapsed
                    })
                }
                aria-label={collapsed ? 'Expand' : 'Collapse'}
            >
                <h5 className="flex-grow-1">{header}</h5>
                {collapsed ? (
                    <ChevronLeftIcon className="icon-inline mr-1" />
                ) : (
                    <ChevronDownIcon className="icon-inline mr-1" />
                )}
            </button>

            <Collapse isOpen={!collapsed}>
                <div className={classNames('pb-4', !searchVisible && 'border-top')}>
                    {searchVisible && (
                        <input
                            type="search"
                            placeholder="Find..."
                            aria-label="Find filters"
                            value={filter}
                            onChange={event => setFilter(event.currentTarget.value)}
                            data-testid="sidebar-section-search-box"
                            className={classNames('form-control form-control-sm', styles.sidebarSectionSearchBox)}
                        />
                    )}
                    {body}
                </div>
            </Collapse>
        </div>
    ) : null
}
