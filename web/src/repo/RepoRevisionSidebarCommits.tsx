import * as H from 'history'
import FileIcon from 'mdi-react/FileIcon'
import * as React from 'react'
import { Link } from 'react-router-dom'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { createInvalidGraphQLQueryResponseError, dataOrThrowErrors, gql } from '../../../shared/src/graphql/graphql'
import * as GQL from '../../../shared/src/graphql/schema'
import { queryGraphQL } from '../backend/graphql'
import { FilteredConnection, FilteredConnectionQueryArgs } from '../components/FilteredConnection'
import { replaceRevisionInURL } from '../util/url'
import { GitCommitNode } from './commits/GitCommitNode'
import { gitCommitFragment } from './commits/RepositoryCommitsPage'
import { RevisionSpec, FileSpec } from '../../../shared/src/util/url'
import { FetchCommitsResult, FetchCommitsVariables, GitCommitFields } from '../graphql-operations'

type GraphQlCommitConnection = NonNullable<
    NonNullable<FetchCommitsResult['node'] & { __typename: 'Repository' }>['commit']
>['ancestors']

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
    repoID: GQL.Scalars['ID']
    history: H.History
    location: H.Location
}

export class RepoRevisionSidebarCommits extends React.PureComponent<Props> {
    public render(): JSX.Element | null {
        return (
            <FilteredConnection<GitCommitFields, Pick<CommitNodeProps, 'location'>>
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

    private fetchCommits = (args: FilteredConnectionQueryArgs): Observable<GraphQlCommitConnection> =>
        fetchCommits(this.props.repoID, this.props.revision || '', { ...args, currentPath: this.props.filePath || '' })
}

function fetchCommits(
    repo: GQL.Scalars['ID'],
    revision: string,
    args: Omit<FetchCommitsVariables, 'repo' | 'revision'>
): Observable<GraphQlCommitConnection> {
    return queryGraphQL<FetchCommitsResult>(
        gql`
            query FetchCommits($repo: ID!, $revision: String!, $first: Int, $currentPath: String, $query: String) {
                node(id: $repo) {
                    __typename
                    ... on Repository {
                        commit(rev: $revision) {
                            ancestors(first: $first, query: $query, path: $currentPath) {
                                nodes {
                                    ...GitCommitFields
                                }
                            }
                        }
                    }
                }
            }
            ${gitCommitFragment}
        `,
        { ...args, repo, revision }
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
