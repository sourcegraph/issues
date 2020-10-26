import React from 'react'
import { LinkWithIconOnlyTooltip } from '../../../../components/LinkWithIconOnlyTooltip'
import { CampaignsIconNav } from '../../icons'
import classNames from 'classnames'

interface Props {
    className?: string
}

/**
 * An item in {@link GlobalNavbar} that links to the campaigns area.
 */
export const CampaignsNavItem: React.FunctionComponent<Props> = ({ className }) => (
    <LinkWithIconOnlyTooltip
        to="/campaigns"
        text="Campaigns"
        icon={CampaignsIconNav}
        className={classNames('nav-link btn btn-link px-1 text-decoration-none', className)}
        activeClassName="active"
    />
)
