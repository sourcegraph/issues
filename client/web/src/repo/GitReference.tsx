import * as React from 'react'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { LinkOrSpan } from '../../../shared/src/components/LinkOrSpan'
import { gql } from '../../../shared/src/graphql/graphql'
import * as GQL from '../../../shared/src/graphql/schema'
import { createAggregateError } from '../../../shared/src/util/errors'
import { memoizeObservable } from '../../../shared/src/util/memoizeObservable'
import { numberWithCommas } from '../../../shared/src/util/strings'
import { queryGraphQL } from '../backend/graphql'
import { Timestamp } from '../components/time/Timestamp'
import { GitRefType } from '../graphql-operations'

interface GitReferenceNodeProps {
    node: GQL.IGitRef

    /** Link URL; if undefined, node.url is used. */
    url?: string

    /** Whether any ancestor element higher up in the tree is an `<a>` element. */
    ancestorIsLink?: boolean

    children?: React.ReactNode
}

export const GitReferenceNode: React.FunctionComponent<GitReferenceNodeProps> = ({
    node,
    url,
    ancestorIsLink,
    children,
}) => {
    const mostRecentSig =
        node.target.commit &&
        (node.target.commit.committer && node.target.commit.committer.date > node.target.commit.author.date
            ? node.target.commit.committer
            : node.target.commit.author)
    const behindAhead = node.target.commit?.behindAhead
    url = url !== undefined ? url : node.url

    return (
        <LinkOrSpan key={node.id} className="git-ref-node list-group-item" to={!ancestorIsLink ? url : undefined}>
            <span>
                <code className="git-ref-tag-2">{node.displayName}</code>
                {mostRecentSig && (
                    <small className="text-muted pl-2">
                        Updated <Timestamp date={mostRecentSig.date} />{' '}
                        {mostRecentSig.person && <>by {mostRecentSig.person.displayName}</>}
                    </small>
                )}
            </span>
            {behindAhead && (
                <small className="text-muted">
                    {numberWithCommas(behindAhead.behind)} behind, {numberWithCommas(behindAhead.ahead)} ahead
                </small>
            )}
            {children}
        </LinkOrSpan>
    )
}

export const gitReferenceFragments = gql`
    fragment GitRefFields on GitRef {
        id
        displayName
        name
        abbrevName
        url
        target {
            commit {
                author {
                    ...SignatureFieldsForReferences
                }
                committer {
                    ...SignatureFieldsForReferences
                }
                behindAhead(revspec: "HEAD") @include(if: $withBehindAhead) {
                    behind
                    ahead
                }
            }
        }
    }

    fragment SignatureFieldsForReferences on Signature {
        person {
            displayName
            user {
                username
            }
        }
        date
    }
`

export const queryGitReferences = memoizeObservable(
    (args: {
        repo: GQL.ID
        first?: number
        query?: string
        type: GitRefType
        withBehindAhead?: boolean
    }): Observable<GQL.IGitRefConnection> =>
        queryGraphQL(
            gql`
                query RepositoryGitRefs(
                    $repo: ID!
                    $first: Int
                    $query: String
                    $type: GitRefType!
                    $withBehindAhead: Boolean!
                ) {
                    node(id: $repo) {
                        ... on Repository {
                            gitRefs(first: $first, query: $query, type: $type, orderBy: AUTHORED_OR_COMMITTED_AT) {
                                nodes {
                                    ...GitRefFields
                                }
                                totalCount
                                pageInfo {
                                    hasNextPage
                                }
                            }
                        }
                    }
                }
                ${gitReferenceFragments}
            `,
            {
                ...args,
                withBehindAhead:
                    args.withBehindAhead !== undefined ? args.withBehindAhead : args.type === GitRefType.GIT_BRANCH,
            }
        ).pipe(
            map(({ data, errors }) => {
                if (!data || !data.node || !(data.node as GQL.IRepository).gitRefs) {
                    throw createAggregateError(errors)
                }
                return (data.node as GQL.IRepository).gitRefs
            })
        ),
    args => `${args.repo}:${String(args.first)}:${String(args.query)}:${args.type}`
)
