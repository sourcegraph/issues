import AddIcon from 'mdi-react/AddIcon'
import React, { useCallback, useEffect, useMemo } from 'react'
import { RouteComponentProps } from 'react-router'
import { Link } from 'react-router-dom'
import { Observable, Subject } from 'rxjs'
import { map } from 'rxjs/operators'

import { dataOrThrowErrors, gql } from '@sourcegraph/shared/src/graphql/graphql'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { Container, PageHeader } from '@sourcegraph/wildcard'

import { watchQuery } from '../../../backend/graphql'
import { FilteredConnection } from '../../../components/FilteredConnection'
import { PageTitle } from '../../../components/PageTitle'
import {
    AccessTokenFields,
    AccessTokensConnectionFields,
    AccessTokensResult,
    AccessTokensVariables,
    CreateAccessTokenResult,
} from '../../../graphql-operations'
import { accessTokenFragment, AccessTokenNode, AccessTokenNodeProps } from '../../../settings/tokens/AccessTokenNode'
import { UserSettingsAreaRouteContext } from '../UserSettingsArea'

interface Props
    extends Pick<UserSettingsAreaRouteContext, 'user'>,
        Pick<RouteComponentProps<{}>, 'history' | 'location' | 'match'>,
        TelemetryProps {
    /**
     * The newly created token, if any. This component must call onDidPresentNewToken
     * when it is finished presenting the token secret to the user.
     */
    newToken?: CreateAccessTokenResult['createAccessToken']

    /**
     * Called when the newly created access token has been presented to the user and may be purged
     * from all state (and not displayed to the user anymore).
     */
    onDidPresentNewToken: () => void
}

/**
 * Displays access tokens whose subject is a specific user.
 */
export const UserSettingsTokensPage: React.FunctionComponent<Props> = ({
    telemetryService,
    history,
    location,
    match,
    user,
    newToken,
    onDidPresentNewToken,
}) => {
    useEffect(() => {
        telemetryService.logViewEvent('UserSettingsTokens')
    }, [telemetryService])

    useEffect(
        () => () => {
            // Clear the newly created access token value from our application state; we assume the user
            // has already stored it elsewhere.
            onDidPresentNewToken()
        },
        [onDidPresentNewToken]
    )

    const accessTokenUpdates = useMemo(() => new Subject<void>(), [])
    const onDeleteAccessToken = useCallback(() => {
        accessTokenUpdates.next()
    }, [accessTokenUpdates])

    const queryUserAccessTokens = useCallback(
        (args: { first?: number }) => queryAccessTokens({ first: args.first ?? null, user: user.id }),
        [user.id]
    )

    return (
        <div className="user-settings-tokens-page">
            <PageTitle title="Access tokens" />
            <PageHeader
                headingElement="h2"
                path={[{ text: 'Access tokens' }]}
                description="Access tokens may be used to access the Sourcegraph API."
                actions={
                    <Link className="btn btn-primary" to={`${match.url}/new`}>
                        <AddIcon className="icon-inline" /> Generate new token
                    </Link>
                }
                className="mb-3"
            />
            <Container>
                <FilteredConnection<AccessTokenFields, Omit<AccessTokenNodeProps, 'node'>>
                    listClassName="list-group list-group-flush"
                    noun="access token"
                    pluralNoun="access tokens"
                    queryConnection={queryUserAccessTokens}
                    nodeComponent={AccessTokenNode}
                    nodeComponentProps={{
                        afterDelete: onDeleteAccessToken,
                        showSubject: false,
                        newToken,
                    }}
                    updates={accessTokenUpdates}
                    hideSearch={true}
                    noSummaryIfAllNodesVisible={true}
                    history={history}
                    location={location}
                    emptyElement={
                        <p className="text-muted text-center w-100 mb-0">You don't have any access tokens.</p>
                    }
                />
            </Container>
        </div>
    )
}

const queryAccessTokens = (variables: AccessTokensVariables): Observable<AccessTokensConnectionFields> =>
    watchQuery<AccessTokensResult, AccessTokensVariables>(
        gql`
            query AccessTokens($user: ID!, $first: Int) {
                node(id: $user) {
                    __typename
                    ... on User {
                        id
                        accessTokens(first: $first) {
                            ...AccessTokensConnectionFields
                        }
                    }
                }
            }
            fragment AccessTokensConnectionFields on AccessTokenConnection {
                nodes {
                    ...AccessTokenFields
                }
                totalCount
                pageInfo {
                    hasNextPage
                }
            }
            ${accessTokenFragment}
        `,
        variables
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.node) {
                throw new Error('User not found')
            }
            if (data.node.__typename !== 'User') {
                throw new Error(`Mode is a ${data.node.__typename}, not a User`)
            }
            return data.node.accessTokens
        })
    )
