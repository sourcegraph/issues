import * as React from 'react'
import { Link } from 'react-router-dom'
import { CircleChevronRightIcon } from '../../../shared/src/components/icons'
import { DismissibleAlert } from '../components/DismissibleAlert'
import { eventLogger } from '../tracking/eventLogger'

const onClickCTA = (): void => {
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
        className={`alert alert-success d-flex align-items-center ${className}`}
    >
        <Link className="site-alert__link" to="/site-admin/external-services" onClick={onClickCTA}>
            <CircleChevronRightIcon className="icon-inline site-alert__link-icon" />{' '}
            <span className="underline">Connect a code host</span>
        </Link>
        &nbsp;to connect repositories to Sourcegraph.
    </DismissibleAlert>
)
