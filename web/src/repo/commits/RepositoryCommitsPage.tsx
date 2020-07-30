import * as H from 'history'
import * as React from 'react'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { gql } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { createAggregateError } from '../../../../shared/src/util/errors'
import { queryGraphQL } from '../../backend/graphql'
import { FilteredConnection, FilteredConnectionQueryArgs } from '../../components/FilteredConnection'
import { PageTitle } from '../../components/PageTitle'
import { eventLogger } from '../../tracking/eventLogger'
import { RepoHeaderContributionsLifecycleProps } from '../RepoHeader'
import { RepoHeaderBreadcrumbNavItem } from '../RepoHeaderBreadcrumbNavItem'
import { RepoHeaderContributionPortal } from '../RepoHeaderContributionPortal'
import { GitCommitNode, GitCommitNodeProps } from './GitCommitNode'
import { RevisionSpec, ResolvedRevisionSpec } from '../../../../shared/src/util/url'
import { RepositoryGitCommitsResult } from '../../graphql-operations'

export const gitCommitFragment = gql`
    fragment GitCommitFields on GitCommit {
        id
        oid
        abbreviatedOID
        message
        subject
        body
        author {
            ...SignatureFields
        }
        committer {
            ...SignatureFields
        }
        parents {
            oid
            abbreviatedOID
            url
        }
        url
        canonicalURL
        externalURLs {
            url
            serviceType
        }
        tree(path: "") {
            canonicalURL
        }
    }

    fragment SignatureFields on Signature {
        person {
            avatarURL
            name
            email
            displayName
            user {
                id
                username
                displayName
                url
            }
        }
        date
    }
`

const fetchGitCommits = (args: {
    repo: GQL.Scalars['ID']
    revspec: string
    first?: number
    query?: string
}): Observable<GQL.GitCommitConnection> =>
    queryGraphQL<RepositoryGitCommitsResult>(
        gql`
            query RepositoryGitCommits($repo: ID!, $revspec: String!, $first: Int, $query: String) {
                node(id: $repo) {
                    ... on Repository {
                        commit(rev: $revspec) {
                            ancestors(first: $first, query: $query) {
                                nodes {
                                    ...GitCommitFields
                                }
                                pageInfo {
                                    hasNextPage
                                }
                            }
                        }
                    }
                }
            }
            ${gitCommitFragment}
        `,
        args
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.node) {
                throw createAggregateError(errors)
            }
            const repo = data.node as GQL.Repository
            if (!repo.commit || !repo.commit.ancestors) {
                throw createAggregateError(errors)
            }
            return repo.commit.ancestors
        })
    )

interface Props extends RepoHeaderContributionsLifecycleProps, Partial<RevisionSpec>, ResolvedRevisionSpec {
    repo: GQL.Repository

    history: H.History
    location: H.Location
}

/** A page that shows a repository's commits at the current revision. */
export class RepositoryCommitsPage extends React.PureComponent<Props> {
    public componentDidMount(): void {
        eventLogger.logViewEvent('RepositoryCommits')
    }

    public render(): JSX.Element | null {
        return (
            <div className="repository-commits-page">
                <PageTitle title="Commits" />
                <RepoHeaderContributionPortal
                    position="nav"
                    element={<RepoHeaderBreadcrumbNavItem key="commits">Commits</RepoHeaderBreadcrumbNavItem>}
                    repoHeaderContributionsLifecycleProps={this.props.repoHeaderContributionsLifecycleProps}
                />
                <FilteredConnection<GQL.GitCommit, Pick<GitCommitNodeProps, 'className' | 'compact'>>
                    className="repository-commits-page__content"
                    listClassName="list-group list-group-flush"
                    noun="commit"
                    pluralNoun="commits"
                    queryConnection={this.queryCommits}
                    nodeComponent={GitCommitNode}
                    nodeComponentProps={{ className: 'list-group-item' }}
                    defaultFirst={20}
                    autoFocus={true}
                    history={this.props.history}
                    hideSearch={true}
                    location={this.props.location}
                />
            </div>
        )
    }

    private queryCommits = (args: FilteredConnectionQueryArgs): Observable<GQL.GitCommitConnection> =>
        fetchGitCommits({ ...args, repo: this.props.repo.id, revspec: this.props.commitID })
}
