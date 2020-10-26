import React from 'react'
import { RouteComponentProps } from 'react-router'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { Link } from '../../../shared/src/components/Link'
import { dataOrThrowErrors, gql } from '../../../shared/src/graphql/graphql'
import { queryGraphQL } from '../backend/graphql'
import * as GQL from '../../../shared/src/graphql/schema'
import { FilteredConnection } from '../components/FilteredConnection'
import { PageTitle } from '../components/PageTitle'
import { Timestamp } from '../components/time/Timestamp'
import { eventLogger } from '../tracking/eventLogger'
import { UserAreaRouteContext } from './area/UserArea'

interface UserEventNodeProps {
    /**
     * The user to display in this list item.
     */
    node: GQL.IEventLog
}

export const UserEventNode: React.FunctionComponent<UserEventNodeProps> = ({ node }: UserEventNodeProps) => (
    <li className="list-group-item py-2">
        <div className="d-flex align-items-center justify-content-between">
            <code>{node.name}</code>
            <div>
                <Timestamp date={node.timestamp} />
            </div>
        </div>
        <div className="text-break">
            <small>
                From: {node.source}{' '}
                {node.url && (
                    <span>
                        (<Link to={node.url}>{node.url}</Link>)
                    </span>
                )}
            </small>
        </div>
    </li>
)

interface UserEventLogsPageProps extends UserAreaRouteContext, RouteComponentProps {
    isLightTheme: boolean
}

/**
 * A page displaying usage statistics for the site.
 */
export class UserEventLogsPage extends React.PureComponent<UserEventLogsPageProps> {
    public componentDidMount(): void {
        eventLogger.logViewEvent('UserEventLogPage')
    }

    public render(): JSX.Element | null {
        return (
            <div>
                <PageTitle title="User event log" />
                <FilteredConnection<GQL.IEventLog, {}>
                    key="chronological"
                    defaultFirst={50}
                    className="list-group list-group-flush"
                    hideSearch={true}
                    noun="user event"
                    pluralNoun="user events"
                    queryConnection={this.queryUserEventLogs}
                    nodeComponent={UserEventNode}
                    history={this.props.history}
                    location={this.props.location}
                />
            </div>
        )
    }

    private queryUserEventLogs = (args: { first?: number }): Observable<GQL.IEventLogsConnection> =>
        queryGraphQL(
            gql`
                query UserEventLogs($user: ID!, $first: Int) {
                    node(id: $user) {
                        ... on User {
                            eventLogs(first: $first) {
                                nodes {
                                    name
                                    source
                                    url
                                    timestamp
                                }
                                totalCount
                                pageInfo {
                                    hasNextPage
                                }
                            }
                        }
                    }
                }
            `,
            { ...args, user: this.props.user.id }
        ).pipe(
            map(dataOrThrowErrors),
            map(data => (data.node as GQL.IUser).eventLogs)
        )
}
