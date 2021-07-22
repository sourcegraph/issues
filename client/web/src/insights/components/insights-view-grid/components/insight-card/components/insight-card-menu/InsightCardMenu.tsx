import { Menu, MenuButton, MenuItem, MenuItems, MenuLink, MenuPopover } from '@reach/menu-button'
import classnames from 'classnames'
import DotsVerticalIcon from 'mdi-react/DotsVerticalIcon'
import React, { MouseEvent } from 'react'
import { useHistory } from 'react-router'

import { positionRight } from '../../../../../context-menu/utils'

import styles from './InsightCardMenu.module.scss'

export interface InsightCardMenuProps {
    className?: string
    onDelete: (insightID: string) => void
    insightID: string
}

/**
 * Renders context menu (three dots menu) for particular insight card.
 */
export const InsightCardMenu: React.FunctionComponent<InsightCardMenuProps> = props => {
    const { insightID, className, onDelete } = props
    const history = useHistory()

    const handleEditClick = (event: MouseEvent): void => {
        event.preventDefault()
        history.push(`/insights/edit/${insightID}`)
    }

    return (
        <Menu>
            <MenuButton data-testid="InsightContextMenuButton" className={classnames(className, 'btn btn-outline p-1')}>
                <DotsVerticalIcon size={16} />
            </MenuButton>
            <MenuPopover portal={true} position={positionRight}>
                <MenuItems
                    data-testid={`context-menu.${insightID}`}
                    className={classnames(styles.panel, 'dropdown-menu')}
                >
                    <MenuLink
                        data-testid="InsightContextMenuEditLink"
                        className={classnames('btn btn-outline', styles.item)}
                        onClick={handleEditClick}
                    >
                        Edit
                    </MenuLink>

                    <MenuItem
                        data-testid="insight-context-menu-delete-button"
                        onSelect={() => onDelete(insightID)}
                        className={classnames('btn btn-outline-', styles.item)}
                    >
                        Delete
                    </MenuItem>
                </MenuItems>
            </MenuPopover>
        </Menu>
    )
}
