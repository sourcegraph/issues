import AddIcon from 'mdi-react/AddIcon'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { Link } from 'react-router-dom'
import { Observable, Subject } from 'rxjs'
import { map } from 'rxjs/operators'
import { gql } from '../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { createAggregateError } from '../../../../../shared/src/util/errors'
import { queryGraphQL } from '../../../backend/graphql'
import { PageTitle } from '../../../components/PageTitle'
import { accessTokenFragment, AccessTokenNode, AccessTokenNodeProps } from '../../../settings/tokens/AccessTokenNode'
import { eventLogger } from '../../../tracking/eventLogger'
import { UserAreaRouteContext } from '../../area/UserArea'
import { FilteredConnection } from '../../../components/FilteredConnection'
import * as H from 'history'

interface Props extends UserAreaRouteContext, RouteComponentProps<{}> {
    /**
     * The newly created token, if any. This component must call onDidPresentNewToken
     * when it is finished presenting the token secret to the user.
     */
    newToken?: GQL.ICreateAccessTokenResult

    /**
     * Called when the newly created access token has been presented to the user and may be purged
     * from all state (and not displayed to the user anymore).
     */
    onDidPresentNewToken: () => void
    history: H.History
}

interface State {}

/**
 * Displays access tokens whose subject is a specific user.
 */
export class UserSettingsTokensPage extends React.PureComponent<Props, State> {
    private static clearNewTokenTimer: number | undefined = undefined

    public state: State = {}

    private accessTokenUpdates = new Subject<void>()

    public componentDidMount(): void {
        eventLogger.logViewEvent('UserSettingsTokens')

        if (UserSettingsTokensPage.clearNewTokenTimer !== undefined) {
            clearTimeout(UserSettingsTokensPage.clearNewTokenTimer)
        }
    }

    public componentWillUnmount(): void {
        // Clear the newly created access token value from our application state; we assume the user
        // has already stored it elsewhere.
        this.props.onDidPresentNewToken()
    }

    public render(): JSX.Element | null {
        const nodeProps: Omit<AccessTokenNodeProps, 'node'> = {
            onDidUpdate: this.onDidUpdateAccessToken,
            newToken: this.props.newToken,
            history: this.props.history,
        }

        return (
            <div className="user-settings-tokens-page">
                <PageTitle title="Access tokens" />
                <div className="d-flex justify-content-between align-items-center">
                    <h2>Access tokens</h2>
                    <Link className="btn btn-primary ml-2" to={`${this.props.match.url}/new`}>
                        <AddIcon className="icon-inline" /> Generate new token
                    </Link>
                </div>
                <p>Access tokens may be used to access the Sourcegraph API.</p>
                <FilteredConnection<GQL.IAccessToken, Omit<AccessTokenNodeProps, 'node'>>
                    listClassName="list-group list-group-flush"
                    noun="access token"
                    pluralNoun="access tokens"
                    queryConnection={this.queryAccessTokens}
                    nodeComponent={AccessTokenNode}
                    nodeComponentProps={nodeProps}
                    updates={this.accessTokenUpdates}
                    hideSearch={true}
                    noSummaryIfAllNodesVisible={true}
                    history={this.props.history}
                    location={this.props.location}
                />
            </div>
        )
    }

    private queryAccessTokens = (args: { first?: number }): Observable<GQL.IAccessTokenConnection> =>
        queryGraphQL(
            gql`
                query AccessTokens($user: ID!, $first: Int) {
                    node(id: $user) {
                        ... on User {
                            accessTokens(first: $first) {
                                nodes {
                                    ...AccessTokenFields
                                }
                                totalCount
                                pageInfo {
                                    hasNextPage
                                }
                            }
                        }
                    }
                }
                ${accessTokenFragment}
            `,
            { ...args, user: this.props.user.id }
        ).pipe(
            map(({ data, errors }) => {
                if (!data || !data.node) {
                    throw createAggregateError(errors)
                }
                const user = data.node as GQL.IUser
                if (!user.accessTokens) {
                    throw createAggregateError(errors)
                }
                return user.accessTokens
            })
        )

    private onDidUpdateAccessToken = (): void => {
        this.accessTokenUpdates.next()
    }
}
