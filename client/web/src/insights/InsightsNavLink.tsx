import React from 'react'
import { LinkWithIconOnlyTooltip } from '../components/LinkWithIconOnlyTooltip'
import { InsightsIcon } from './icon'

export const InsightsNavItem: React.FunctionComponent = () => (
    <LinkWithIconOnlyTooltip
        to="/insights"
        text="Insights"
        icon={InsightsIcon}
        className="nav-link btn btn-link px-1 text-decoration-none"
        activeClassName="active"
    />
)
