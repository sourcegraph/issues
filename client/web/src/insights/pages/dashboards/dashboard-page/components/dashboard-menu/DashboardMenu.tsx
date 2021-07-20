import { Menu, MenuButton, MenuItem, MenuList } from '@reach/menu-button'
import classnames from 'classnames'
import DotsVerticalIcon from 'mdi-react/DotsVerticalIcon'
import React from 'react'

import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'

import { Settings } from '../../../../../../schema/settings.schema'
import { InsightDashboard } from '../../../../../core/types'
import { getTooltipMessage, useDashboardPermissions } from '../../hooks/use-dashboard-permissions'

import styles from './DashboardMenu.module.scss'

export enum DashboardMenuAction {
    CopyLink,
    Delete,
    Configure,
    AddRemoveInsights,
}

export interface DashboardMenuProps extends SettingsCascadeProps<Settings> {
    innerRef: React.Ref<HTMLButtonElement>
    dashboard?: InsightDashboard
    onSelect?: (action: DashboardMenuAction) => void
    tooltipText?: string
}

export const DashboardMenu: React.FunctionComponent<DashboardMenuProps> = props => {
    const { innerRef, dashboard, settingsCascade, onSelect = () => {}, tooltipText } = props

    const hasDashboard = dashboard !== undefined
    const permissions = useDashboardPermissions(dashboard, settingsCascade)

    return (
        <Menu>
            <MenuButton
                ref={innerRef}
                data-tooltip={tooltipText}
                data-placement="right"
                className={classnames(styles.triggerButton, 'btn btn-icon')}
            >
                <DotsVerticalIcon size={16} />
            </MenuButton>

            <MenuList className={classnames(styles.menuList, 'dropdown-menu')}>
                <MenuItem
                    as="button"
                    disabled={!permissions.isConfigurable}
                    data-tooltip={getTooltipMessage(permissions)}
                    data-placement="right"
                    className={classnames(styles.menuItem, 'btn btn-outline')}
                    onSelect={() => onSelect(DashboardMenuAction.AddRemoveInsights)}
                >
                    Add insights
                </MenuItem>

                <MenuItem
                    as="button"
                    disabled={!permissions.isConfigurable}
                    data-tooltip={getTooltipMessage(permissions)}
                    data-placement="right"
                    className={classnames(styles.menuItem, 'btn btn-outline')}
                    onSelect={() => onSelect(DashboardMenuAction.Configure)}
                >
                    Configure dashboard
                </MenuItem>

                <MenuItem
                    as="button"
                    disabled={!hasDashboard}
                    className={classnames(styles.menuItem, 'btn btn-outline')}
                    onSelect={() => onSelect(DashboardMenuAction.CopyLink)}
                >
                    Copy link
                </MenuItem>

                <hr />

                <MenuItem
                    as="button"
                    disabled={!permissions.isConfigurable}
                    data-tooltip={getTooltipMessage(permissions)}
                    data-placement="right"
                    className={classnames(styles.menuItem, 'btn btn-outline', styles.menuItemDanger)}
                    onSelect={() => onSelect(DashboardMenuAction.Delete)}
                >
                    Delete
                </MenuItem>
            </MenuList>
        </Menu>
    )
}
