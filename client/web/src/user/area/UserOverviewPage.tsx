import React, { useEffect } from 'react'
import { RouteComponentProps } from 'react-router'
import { PageTitle } from '../../components/PageTitle'
import { Timestamp } from '../../components/time/Timestamp'
import { eventLogger } from '../../tracking/eventLogger'
import { UserAreaRouteContext } from './UserArea'

interface Props extends UserAreaRouteContext, RouteComponentProps<{}> {}

/**
 * The user overview page.
 */
export const UserOverviewPage: React.FunctionComponent<Props> = props => {
    useEffect(() => eventLogger.logViewEvent('UserOverview'), [])

    return (
        <div className="user-page user-overview-page">
            <PageTitle title={props.user.username} />
            <p>
                {props.user.displayName ? (
                    <>
                        {props.user.displayName} ({props.user.username})
                    </>
                ) : (
                    props.user.username
                )}{' '}
                started using Sourcegraph <Timestamp date={props.user.createdAt} />.
            </p>
        </div>
    )
}
