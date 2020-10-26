import AddIcon from 'mdi-react/AddIcon'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { Observable, Subject } from 'rxjs'
import { map } from 'rxjs/operators'
import { gql } from '../../../shared/src/graphql/graphql'
import * as GQL from '../../../shared/src/graphql/schema'
import { createAggregateError } from '../../../shared/src/util/errors'
import { queryGraphQL } from '../backend/graphql'
import { PageTitle } from '../components/PageTitle'
import { accessTokenFragment, AccessTokenNode, AccessTokenNodeProps } from '../settings/tokens/AccessTokenNode'
import { eventLogger } from '../tracking/eventLogger'
import { LinkOrSpan } from '../../../shared/src/components/LinkOrSpan'
import { FilteredConnection } from '../components/FilteredConnection'
import { AuthenticatedUser } from '../auth'

interface Props extends RouteComponentProps<{}> {
    authenticatedUser: AuthenticatedUser
}

interface State {}

/**
 * Displays a list of all access tokens on the site.
 */
export class SiteAdminTokensPage extends React.PureComponent<Props, State> {
    public state: State = {}

    private accessTokenUpdates = new Subject<void>()

    public componentDidMount(): void {
        eventLogger.logViewEvent('SiteAdminTokens')
    }

    public render(): JSX.Element | null {
        const nodeProps: Omit<AccessTokenNodeProps, 'node'> = {
            showSubject: true,
            onDidUpdate: this.onDidUpdateAccessToken,
            history: this.props.history,
        }

        const accessTokensEnabled = window.context.accessTokensAllow !== 'none'
        return (
            <div className="user-settings-tokens-page">
                <PageTitle title="Access tokens - Admin" />
                <div className="d-flex justify-content-between align-items-center mb-3">
                    <h2 className="mb-0">Access tokens</h2>
                    <LinkOrSpan
                        title={accessTokensEnabled ? '' : 'Access token creation is disabled in site configuration'}
                        className={`btn btn-primary ml-2 ${accessTokensEnabled ? '' : 'disabled'}`}
                        to={accessTokensEnabled ? `${this.props.authenticatedUser.settingsURL!}/tokens/new` : null}
                    >
                        <AddIcon className="icon-inline" /> Generate access token
                    </LinkOrSpan>
                </div>
                <p>Tokens may be used to access the Sourcegraph API with the full privileges of the token's creator.</p>
                <FilteredConnection<GQL.IAccessToken, Omit<AccessTokenNodeProps, 'node'>>
                    className="list-group list-group-flush mt-3"
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
                query SiteAdminAccessTokens($first: Int) {
                    site {
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
                ${accessTokenFragment}
            `,
            args
        ).pipe(
            map(({ data, errors }) => {
                if (!data || !data.site || !data.site.accessTokens || errors) {
                    throw createAggregateError(errors)
                }
                return data.site.accessTokens
            })
        )

    private onDidUpdateAccessToken = (): void => this.accessTokenUpdates.next()
}
