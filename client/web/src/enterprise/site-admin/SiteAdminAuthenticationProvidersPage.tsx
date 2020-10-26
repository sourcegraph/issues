import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { gql } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { createAggregateError } from '../../../../shared/src/util/errors'
import { queryGraphQL } from '../../backend/graphql'
import { FilteredConnection } from '../../components/FilteredConnection'
import { PageTitle } from '../../components/PageTitle'
import { eventLogger } from '../../tracking/eventLogger'

interface AuthProviderNodeProps {
    /** The auth provider to display in this item. */
    node: GQL.IAuthProvider
}

/** Whether to show experimental auth features. */
export const authExp = localStorage.getItem('authExp') !== null

class AuthProviderNode extends React.PureComponent<AuthProviderNodeProps> {
    public render(): JSX.Element | null {
        return (
            <li className="list-group-item py-2">
                <div className="d-flex align-items-center justify-content-between">
                    <div className="mr-2">
                        <strong>{this.props.node.displayName}</strong>{' '}
                        <span className="badge badge-secondary">{this.props.node.serviceType}</span>
                        <br />
                        {(this.props.node.serviceID || this.props.node.clientID) && (
                            <small className="text-muted">
                                {this.props.node.serviceID}
                                {this.props.node.clientID && <> &mdash; {this.props.node.clientID}</>}
                            </small>
                        )}
                    </div>
                    {authExp && (
                        <div className="text-nowrap">
                            {this.props.node.authenticationURL && (
                                <a className="btn btn-secondary" href={this.props.node.authenticationURL}>
                                    Authenticate
                                </a>
                            )}
                        </div>
                    )}
                </div>
            </li>
        )
    }
}

const authProviderFragment = gql`
    fragment AuthProviderFields on AuthProvider {
        serviceType
        serviceID
        clientID
        displayName
        isBuiltin
        authenticationURL
    }
`

interface Props extends RouteComponentProps<{}> {}

class FilteredAuthProviderConnection extends FilteredConnection<GQL.IAuthProvider> {}

/**
 * A page displaying the auth providers in site configuration.
 */
export class SiteAdminAuthenticationProvidersPage extends React.Component<Props> {
    public componentDidMount(): void {
        eventLogger.logViewEvent('SiteAdminAuthentication')
    }

    public render(): JSX.Element | null {
        return (
            <div className="site-admin-authentication-page">
                <PageTitle title="Authentication providers - Admin" />
                <h2>Authentication providers</h2>
                <p>
                    Authentication providers allow users to sign into Sourcegraph. See{' '}
                    <a href="https://docs.sourcegraph.com/admin/auth">authentication documentation</a> about configuring
                    single-sign-on (SSO) via SAML and OpenID Connect. Configure authentication providers in the{' '}
                    <a href="https://docs.sourcegraph.com/admin/config/site_config">site configuration</a>.
                </p>
                <FilteredAuthProviderConnection
                    className="list-group list-group-flush mt-3"
                    noun="authentication provider"
                    pluralNoun="authentication providers"
                    queryConnection={this.queryAuthProviders}
                    nodeComponent={AuthProviderNode}
                    hideSearch={true}
                    history={this.props.history}
                    location={this.props.location}
                />
            </div>
        )
    }

    private queryAuthProviders = (args: {}): Observable<GQL.IAuthProviderConnection> =>
        queryGraphQL(
            gql`
                query AuthProviders {
                    site {
                        authProviders {
                            nodes {
                                ...AuthProviderFields
                            }
                            totalCount
                            pageInfo {
                                hasNextPage
                            }
                        }
                    }
                }
                ${authProviderFragment}
            `,
            args
        ).pipe(
            map(({ data, errors }) => {
                if (!data || !data.site || !data.site.authProviders || errors) {
                    throw createAggregateError(errors)
                }
                return data.site.authProviders
            })
        )
}
