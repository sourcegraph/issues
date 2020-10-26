import React from 'react'
import { CampaignFields } from '../../../graphql-operations'
import { Link } from '../../../../../shared/src/components/Link'
import { Timestamp } from '../../../components/time/Timestamp'
import classNames from 'classnames'

interface Props extends Pick<CampaignFields, 'createdAt' | 'initialApplier' | 'lastAppliedAt' | 'lastApplier'> {
    className?: string
}

/**
 * The created/updated byline in the campaign header.
 */
export const CampaignInfoByline: React.FunctionComponent<Props> = ({
    className,
    createdAt,
    initialApplier,
    lastAppliedAt,
    lastApplier,
}) => (
    <div className={classNames('text-muted', className)}>
        {initialApplier ? <Link to={initialApplier.url}>{initialApplier.username}</Link> : 'A deleted user'}{' '}
        <Timestamp date={createdAt} />
        {lastAppliedAt !== createdAt && (
            <>
                <span className="mx-2">•</span>
                {lastApplier?.username !== initialApplier?.username && (
                    <>{lastApplier ? <Link to={lastApplier.url}>{lastApplier.username}</Link> : 'A deleted user'} </>
                )}
                updated <Timestamp date={lastAppliedAt} />
            </>
        )}
    </div>
)
