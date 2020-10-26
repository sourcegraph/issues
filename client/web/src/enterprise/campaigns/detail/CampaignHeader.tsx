import React from 'react'
import { CampaignsIconFlushLeft } from '../icons'
import { CampaignFields } from '../../../graphql-operations'
import { Link } from '../../../../../shared/src/components/Link'
import { PageHeader } from '../../../components/PageHeader'
import classNames from 'classnames'

interface Props {
    className?: string
    name: string
    namespace: Pick<CampaignFields['namespace'], 'namespaceName' | 'url'>
    actionSection?: JSX.Element
}

/**
 * The header bar for campaigns pages.
 */
export const CampaignHeader: React.FunctionComponent<Props> = ({ className, name, namespace, actionSection }) => (
    <PageHeader
        icon={CampaignsIconFlushLeft}
        title={
            <>
                <Link to={namespace.url + '/campaigns'}>{namespace.namespaceName}</Link> / {name}
            </>
        }
        actions={actionSection}
        className={classNames('justify-content-end', className)}
    />
)
