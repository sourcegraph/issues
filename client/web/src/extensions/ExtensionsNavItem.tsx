import React from 'react'
import { LinkWithIconOnlyTooltip } from '../components/LinkWithIconOnlyTooltip'
import Icon from 'mdi-react/PuzzleOutlineIcon'

export const ExtensionsNavItem: React.FunctionComponent = () => (
    <LinkWithIconOnlyTooltip
        to="/extensions"
        text="Extensions"
        icon={Icon}
        className="nav-link btn btn-link px-1 text-decoration-none"
        activeClassName="active"
    />
)
