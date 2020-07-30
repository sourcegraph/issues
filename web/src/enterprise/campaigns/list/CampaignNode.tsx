import React from 'react'
import { Markdown } from '../../../../../shared/src/components/Markdown'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { renderMarkdown } from '../../../../../shared/src/util/markdown'
import { CampaignsIcon } from '../icons'
import { Link } from '../../../../../shared/src/components/Link'
import classNames from 'classnames'
import formatDistance from 'date-fns/formatDistance'
import parseISO from 'date-fns/parseISO'
import * as H from 'history'
import { changesetExternalStateIcons, changesetExternalStateColorClasses } from '../detail/changesets/presentation'
import { Timestamp } from '../../../components/time/Timestamp'
import { ListCampaign } from '../../../graphql-operations'

export interface CampaignNodeProps {
    node: ListCampaign
    /** Used for testing purposes. Sets the current date */
    now?: Date
    history: H.History
}

/**
 * An item in the list of campaigns.
 */
export const CampaignNode: React.FunctionComponent<CampaignNodeProps> = ({ node, history, now = new Date() }) => {
    const campaignIconClass = node.closedAt ? 'text-danger' : 'text-success'
    const OpenChangesetIcon = changesetExternalStateIcons[GQL.ChangesetExternalState.OPEN]
    const ClosedChangesetIcon = changesetExternalStateIcons[GQL.ChangesetExternalState.CLOSED]
    const MergedChangesetIcon = changesetExternalStateIcons[GQL.ChangesetExternalState.MERGED]
    return (
        <li className="list-group-item">
            <div className="d-flex align-items-center p-2">
                <CampaignsIcon
                    className={classNames('icon-inline mr-2 flex-shrink-0 align-self-stretch', campaignIconClass)}
                    data-tooltip={node.closedAt ? 'Closed' : 'Open'}
                />
                <div className="flex-grow-1 campaign-node__content">
                    <div className="m-0 d-flex align-items-baseline">
                        <h3 className="m-0 d-inline-block">
                            <Link to={`/campaigns/${node.id}`}>{node.name}</Link>
                        </h3>
                        <small className="ml-2 text-muted">
                            created{' '}
                            <span data-tooltip={<Timestamp date={node.createdAt} />}>
                                {formatDistance(parseISO(node.createdAt), now)} ago
                            </span>{' '}
                            by <span className="badge badge-secondary">{node.author.username}</span>
                        </small>
                    </div>
                    <Markdown
                        className={classNames('text-truncate', !node.description && 'text-muted font-italic')}
                        dangerousInnerHTML={
                            node.description ? renderMarkdown(node.description, { plainText: true }) : 'No description'
                        }
                        history={history}
                    />
                </div>
                <div className="flex-shrink-0" data-tooltip="Open changesets">
                    {node.changesets.stats.open}{' '}
                    <OpenChangesetIcon className={`text-${changesetExternalStateColorClasses.OPEN} ml-1 mr-2`} />
                </div>
                <div className="flex-shrink-0" data-tooltip="Closed changesets">
                    {node.changesets.stats.closed}{' '}
                    <ClosedChangesetIcon className={`text-${changesetExternalStateColorClasses.CLOSED} ml-1 mr-2`} />
                </div>
                <div className="flex-shrink-0" data-tooltip="Merged changesets">
                    {node.changesets.stats.merged}{' '}
                    <MergedChangesetIcon className={`text-${changesetExternalStateColorClasses.MERGED} ml-1`} />
                </div>
            </div>
        </li>
    )
}
