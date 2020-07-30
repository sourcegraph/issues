import ChevronRightIcon from 'mdi-react/ChevronRightIcon'
import React from 'react'
import { Link } from 'react-router-dom'
import { Observable, Subscription } from 'rxjs'
import { catchError, map } from 'rxjs/operators'
import { RepoLink } from '../../../../shared/src/components/RepoLink'
import { gql } from '../../../../shared/src/graphql/graphql'
import { asError, createAggregateError, ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import { buildSearchURLQuery } from '../../../../shared/src/util/url'
import { queryGraphQL } from '../../backend/graphql'
import { PatternTypeProps } from '../../search'
import { ErrorAlert } from '../../components/alerts'
import * as H from 'history'
import { ExploreRepositoriesResult, ExploreRepositoriesVariables } from '../../graphql-operations'

const LOADING = 'loading' as const

interface Props extends Omit<PatternTypeProps, 'setPatternType'> {
    history: H.History
}

interface State {
    /** The repositories, loading, or an error. */
    repositoriesOrError: typeof LOADING | ExploreRepositoriesResult['repositories'] | ErrorLike
}

/**
 * An explore section that shows a few repositories and a link to all.
 */
export class RepositoriesExploreSection extends React.PureComponent<Props, State> {
    private static QUERY_REPOSITORIES_ARGS: ExploreRepositoriesVariables & { first: number } = {
        // Show sample repositories on Sourcegraph.com.
        names: window.context.sourcegraphDotComMode
            ? [
                  'github.com/sourcegraph/sourcegraph',
                  'github.com/theupdateframework/notary',
                  'github.com/pallets/flask',
                  'github.com/ReactiveX/rxjs',
              ]
            : null,
        first: 4,
    }

    public state: State = { repositoriesOrError: LOADING }

    private subscriptions = new Subscription()

    public componentDidMount(): void {
        this.subscriptions.add(
            queryRepositories(RepositoriesExploreSection.QUERY_REPOSITORIES_ARGS)
                .pipe(catchError(error => [asError(error)]))
                .subscribe(repositoriesOrError => this.setState({ repositoriesOrError }))
        )
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        const repositoriesOrError:
            | (typeof LOADING | ExploreRepositoriesResult['repositories']['nodes'][number])[]
            | ErrorLike =
            this.state.repositoriesOrError === LOADING
                ? new Array(RepositoriesExploreSection.QUERY_REPOSITORIES_ARGS.first).fill(LOADING)
                : isErrorLike(this.state.repositoriesOrError)
                ? this.state.repositoriesOrError
                : this.state.repositoriesOrError.nodes

        const itemClass = 'py-2'

        return (
            <div className="card">
                <h3 className="card-header">Repositories</h3>
                <div className="list-group list-group-flush">
                    {isErrorLike(repositoriesOrError) ? (
                        <ErrorAlert error={repositoriesOrError} history={this.props.history} />
                    ) : repositoriesOrError.length === 0 ? (
                        <p>No repositories.</p>
                    ) : (
                        repositoriesOrError.map((repo /* or loading */, index) =>
                            repo === LOADING ? (
                                <div key={index} className={`${itemClass} list-group-item`}>
                                    <h4 className="text-muted mb-0">⋯</h4>&nbsp;
                                </div>
                            ) : (
                                <Link
                                    key={index}
                                    className={`${itemClass} list-group-item list-group-item-action text-truncate`}
                                    to={repo.url}
                                >
                                    <h4 className="mb-0 text-truncate">
                                        <RepoLink to={null} repoName={repo.name} />
                                    </h4>
                                    <span>{repo.description || <>&nbsp;</>}</span>
                                </Link>
                            )
                        )
                    )}
                </div>
                <div className="card-footer">
                    <Link to={`/search?${buildSearchURLQuery('repo:', this.props.patternType, false)}`}>
                        View all repositories
                        <ChevronRightIcon className="icon-inline" />
                    </Link>
                </div>
            </div>
        )
    }
}

function queryRepositories(args: ExploreRepositoriesVariables): Observable<ExploreRepositoriesResult['repositories']> {
    return queryGraphQL<ExploreRepositoriesResult>(
        gql`
            query ExploreRepositories($first: Int, $names: [String!]) {
                repositories(first: $first, names: $names) {
                    nodes {
                        name
                        description
                        url
                    }
                }
            }
        `,
        args
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.repositories || (errors && errors.length > 0)) {
                throw createAggregateError(errors)
            }
            return data.repositories
        })
    )
}
