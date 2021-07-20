import { ListboxButton } from '@reach/listbox'
import classnames from 'classnames'
import ChevronDownIcon from 'mdi-react/ChevronDownIcon'
import ChevronUpIcon from 'mdi-react/ChevronUpIcon'
import React from 'react'

import { InsightDashboard, InsightsDashboardType } from '../../../../../../../core/types'
import { getDashboardOwnerName, getDashboardTitle } from '../../helpers/get-dashboard-title'
import { Badge } from '../badge/Badge'
import { TruncatedText } from '../trancated-text/TrancatedText'

import styles from './MenuButton.module.scss'

interface MenuButtonProps {
    dashboards: InsightDashboard[]
    className?: string
}

/**
 * Renders ListBox menu button for dashboard select component.
 */
export const MenuButton: React.FunctionComponent<MenuButtonProps> = props => {
    const { dashboards, className } = props

    return (
        <ListboxButton className={classnames(styles.button, className)}>
            {({ value, isExpanded }) => {
                const dashboard = dashboards.find(dashboard => dashboard.id === value)

                if (!dashboard) {
                    return <MenuButtonContent title="Unknown dashboard" isExpanded={isExpanded} />
                }

                if (dashboard.type === InsightsDashboardType.All) {
                    return <MenuButtonContent title="All Insights" isExpanded={isExpanded} />
                }

                return (
                    <MenuButtonContent
                        title={getDashboardTitle(dashboard)}
                        badge={getDashboardOwnerName(dashboard)}
                        isExpanded={isExpanded}
                    />
                )
            }}
        </ListboxButton>
    )
}

interface MenuButtonContentProps {
    title: string
    isExpanded: boolean
    badge?: string
}

const MenuButtonContent: React.FunctionComponent<MenuButtonContentProps> = props => {
    const { title, isExpanded, badge } = props
    const ListboxButtonIcon = isExpanded ? ChevronUpIcon : ChevronDownIcon

    return (
        <>
            <span className={styles.text}>
                <TruncatedText title={title}>{title}</TruncatedText>
                {badge && <Badge value={badge} className={classnames('ml-1 mr-1', styles.badge)} />}
            </span>

            <ListboxButtonIcon className={styles.expandedIcon} />
        </>
    )
}
