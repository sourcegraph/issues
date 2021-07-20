import classnames from 'classnames'
import PlusIcon from 'mdi-react/PlusIcon'
import React from 'react'
import { Link } from 'react-router-dom'

import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'

import { Settings } from '../../../../../../../../schema/settings.schema'
import { InsightDashboard } from '../../../../../../../core/types'
import { getTooltipMessage, useDashboardPermissions } from '../../../../hooks/use-dashboard-permissions'
import { isDashboardConfigurable } from '../../utils/is-dashboard-configurable'

import styles from './EmptyInsightDashboard.module.scss'

interface EmptyInsightDashboardProps extends SettingsCascadeProps<Settings> {
    dashboard: InsightDashboard
    onAddInsight: () => void
}

export const EmptyInsightDashboard: React.FunctionComponent<EmptyInsightDashboardProps> = props => {
    const { onAddInsight, dashboard, settingsCascade } = props

    return isDashboardConfigurable(dashboard) ? (
        <EmptySettingsBasedDashboard
            dashboard={dashboard}
            settingsCascade={settingsCascade}
            onAddInsight={onAddInsight}
        />
    ) : (
        <EmptyBuiltInDashboard />
    )
}

/**
 * Built-in empty dashboard state provides link to create a new code insight via creation UI.
 * Since all insights within built-in dashboards are calculated there's no ability to add insight to
 * this type of dashboard.
 */
export const EmptyBuiltInDashboard: React.FunctionComponent = () => (
    <section className={styles.emptySection}>
        <Link to="/insights/create" className={classnames(styles.itemCard, 'card')}>
            <PlusIcon size="2rem" />
            <span>Create new insight</span>
        </Link>
        <span className="d-flex justify-content-center mt-3">
            <span>
                ...or add existing insights from <Link to="/insights/dashboards/all">All Insights</Link>
            </span>
        </span>
    </section>
)

/**
 * Settings based empty dashboard state provides button for adding existing insights to the dashboard.
 * Since it is possible with settings based dashboard to add existing insights to it.
 */
export const EmptySettingsBasedDashboard: React.FunctionComponent<EmptyInsightDashboardProps> = props => {
    const { onAddInsight, settingsCascade, dashboard } = props
    const permissions = useDashboardPermissions(dashboard, settingsCascade)

    return (
        <section className={styles.emptySection}>
            <button
                type="button"
                disabled={!permissions.isConfigurable}
                onClick={onAddInsight}
                className="btn btn-secondary p-0 w-100 border-0"
            >
                <div
                    data-tooltip={!permissions.isConfigurable ? getTooltipMessage(permissions) : undefined}
                    data-placement="right"
                    className={classnames(styles.itemCard, 'card')}
                >
                    <PlusIcon size="2rem" />
                    <span>Add insight</span>
                </div>
            </button>
            <span className="d-flex justify-content-center mt-3">
                <Link to="/insights/create">...or create a new insight</Link>
            </span>
        </section>
    )
}
