import * as React from 'react'
import { Link } from 'react-router-dom'
import { CircleChevronRightIcon } from '../../../shared/src/components/icons' // TODO: Switch to mdi icon
import { DismissibleAlert } from '../components/DismissibleAlert'
import { eventLogger } from '../tracking/eventLogger'

const onClickCTA = () => {
    eventLogger.log('AlertNeedsRepoConfigCTAClicked')
}

/**
 * A global alert telling the site admin that they need to configure repositories
 * on this site.
 */
export const NeedsRepositoryConfigurationAlert: React.FunctionComponent<{ className?: string }> = ({
    className = '',
}) => (
    <DismissibleAlert
        partialStorageKey="needsRepositoryConfiguration"
        className={`alert alert-success alert-animated-bg d-flex align-items-center ${className}`}
    >
        <Link className="site-alert__link" to="/site-admin/external-services" onClick={onClickCTA}>
            <CircleChevronRightIcon className="icon-inline site-alert__link-icon" />{' '}
            <span className="underline">Configure external services</span>
        </Link>
        &nbsp;to connect to Sourcegraph.
    </DismissibleAlert>
)
