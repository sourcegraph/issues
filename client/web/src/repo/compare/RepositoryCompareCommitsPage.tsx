import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { Observable, Subject, Subscription } from 'rxjs'
import { distinctUntilChanged, map, startWith } from 'rxjs/operators'
import { gql } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { createAggregateError } from '../../../../shared/src/util/errors'
import { queryGraphQL } from '../../backend/graphql'
import { FilteredConnection } from '../../components/FilteredConnection'
import { GitCommitNode, GitCommitNodeProps } from '../commits/GitCommitNode'
import { gitCommitFragment } from '../commits/RepositoryCommitsPage'
import { RepositoryCompareAreaPageProps } from './RepositoryCompareArea'
import { GitCommitFields } from '../../graphql-operations'

function queryRepositoryComparisonCommits(args: {
    repo: GQL.ID
    base: string | null
    head: string | null
    first?: number
}): Observable<GQL.IGitCommitConnection> {
    return queryGraphQL(
        gql`
            query RepositoryComparisonCommits($repo: ID!, $base: String, $head: String, $first: Int) {
                node(id: $repo) {
                    ... on Repository {
                        comparison(base: $base, head: $head) {
                            commits(first: $first) {
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
            const repo = data.node as GQL.IRepository
            if (!repo.comparison || !repo.comparison.commits || errors) {
                throw createAggregateError(errors)
            }
            return repo.comparison.commits
        })
    )
}

interface Props extends RepositoryCompareAreaPageProps, RouteComponentProps<{}> {}

/** A page with a list of commits in the comparison. */
export class RepositoryCompareCommitsPage extends React.PureComponent<Props> {
    private componentUpdates = new Subject<Props>()
    private updates = new Subject<void>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    startWith(this.props),
                    distinctUntilChanged(
                        (a, b) =>
                            a.repo.id === b.repo.id &&
                            a.base.revision === b.base.revision &&
                            a.head.revision === b.head.revision
                    )
                )
                .subscribe(() => this.updates.next())
        )
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return (
            <div className="repository-compare-page">
                <div className="card">
                    <div className="card-header">Commits</div>
                    <FilteredConnection<GitCommitFields, Pick<GitCommitNodeProps, 'className' | 'compact'>>
                        listClassName="list-group list-group-flush"
                        noun="commit"
                        pluralNoun="commits"
                        compact={true}
                        queryConnection={this.queryCommits}
                        nodeComponent={GitCommitNode}
                        nodeComponentProps={{
                            className: 'list-group-item',
                            compact: true,
                        }}
                        defaultFirst={50}
                        hideSearch={true}
                        noSummaryIfAllNodesVisible={true}
                        updates={this.updates}
                        history={this.props.history}
                        location={this.props.location}
                    />
                </div>
            </div>
        )
    }

    private queryCommits = (args: { first?: number }): Observable<GQL.IGitCommitConnection> =>
        queryRepositoryComparisonCommits({
            ...args,
            repo: this.props.repo.id,
            base: this.props.base.revision || null,
            head: this.props.head.revision || null,
        })
}
