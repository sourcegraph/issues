import * as H from 'history'
import FileIcon from 'mdi-react/FileIcon'
import * as React from 'react'
import { Link } from 'react-router-dom'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { createInvalidGraphQLQueryResponseError, dataOrThrowErrors, gql } from '../../../shared/src/graphql/graphql'
import { requestGraphQL } from '../backend/graphql'
import { FilteredConnection } from '../components/FilteredConnection'
import { replaceRevisionInURL } from '../util/url'
import { GitCommitNode } from './commits/GitCommitNode'
import { gitCommitFragment } from './commits/RepositoryCommitsPage'
import { RevisionSpec, FileSpec } from '../../../shared/src/util/url'
import {
    CommitAncestorsConnectionFields,
    FetchCommitsResult,
    FetchCommitsVariables,
    GitCommitFields,
    Scalars,
} from '../graphql-operations'

interface CommitNodeProps {
    node: GitCommitFields
    location: H.Location
}

const CommitNode: React.FunctionComponent<CommitNodeProps> = ({ node, location }) => (
    <li className="list-group-item p-0">
        <GitCommitNode
            compact={true}
            node={node}
            hideExpandCommitMessageBody={true}
            afterElement={
                <Link
                    to={replaceRevisionInURL(location.pathname + location.search + location.hash, node.oid)}
                    className="ml-2"
                    title="View current file at this commit"
                >
                    <FileIcon className="icon-inline" />
                </Link>
            }
        />
    </li>
)

interface Props extends Partial<RevisionSpec>, FileSpec {
    repoID: Scalars['ID']
    history: H.History
    location: H.Location
}

export class RepoRevisionSidebarCommits extends React.PureComponent<Props> {
    public render(): JSX.Element | null {
        return (
            <FilteredConnection<GitCommitFields, Pick<CommitNodeProps, 'location'>, CommitAncestorsConnectionFields>
                className="list-group list-group-flush"
                compact={true}
                noun="commit"
                pluralNoun="commits"
                queryConnection={this.fetchCommits}
                nodeComponent={CommitNode}
                nodeComponentProps={{ location: this.props.location }}
                defaultFirst={100}
                hideSearch={true}
                useURLQuery={false}
                history={this.props.history}
                location={this.props.location}
            />
        )
    }

    private fetchCommits = (args: { query?: string }): Observable<CommitAncestorsConnectionFields> =>
        fetchCommits(this.props.repoID, this.props.revision || '', { ...args, currentPath: this.props.filePath || '' })
}

function fetchCommits(
    repo: Scalars['ID'],
    revision: string,
    args: { first?: number; currentPath?: string; query?: string }
): Observable<CommitAncestorsConnectionFields> {
    return requestGraphQL<FetchCommitsResult, FetchCommitsVariables>(
        gql`
            query FetchCommits($repo: ID!, $revision: String!, $first: Int, $currentPath: String, $query: String) {
                node(id: $repo) {
                    __typename
                    ... on Repository {
                        commit(rev: $revision) {
                            ancestors(first: $first, query: $query, path: $currentPath) {
                                ...CommitAncestorsConnectionFields
                            }
                        }
                    }
                }
            }

            ${gitCommitFragment}

            fragment CommitAncestorsConnectionFields on GitCommitConnection {
                nodes {
                    ...GitCommitFields
                }
                pageInfo {
                    hasNextPage
                }
            }
        `,
        {
            currentPath: args.currentPath ?? null,
            first: args.first ?? null,
            query: args.query ?? null,
            repo,
            revision,
        }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.node || data.node.__typename !== 'Repository' || !data.node.commit) {
                throw createInvalidGraphQLQueryResponseError('FetchCommits')
            }
            return data.node.commit.ancestors
        })
    )
}
