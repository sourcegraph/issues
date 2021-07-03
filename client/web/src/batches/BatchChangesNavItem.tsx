import React from 'react'

import { NavItem, NavLink } from '@sourcegraph/wildcard/src/components/NavBar'

import { FeatureFlagProps } from '../featureFlags/featureFlags'

import { BatchChangesIconNav } from './icons'

interface Props extends FeatureFlagProps {
    isSourcegraphDotCom: boolean
}

/**
 * An item in {@link GlobalNavbar} that links to the batch changes area.
 */
export const BatchChangesNavItem: React.FunctionComponent<Props> = ({ isSourcegraphDotCom, featureFlags }) => {
    const shouldRedirect = isSourcegraphDotCom && !featureFlags.has('w1-signup-optimisation')
    return (
        <NavItem icon={BatchChangesIconNav}>
            <NavLink
                to={shouldRedirect ? 'https://about.sourcegraph.com/batch-changes' : '/batch-changes'}
                external={shouldRedirect}
            >
                Batch Changes
            </NavLink>
        </NavItem>
    )
}
