import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import ErrorIcon from 'mdi-react/ErrorIcon'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React from 'react'
import { Link, RouteComponentProps } from 'react-router-dom'
import { Observable, Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, map, startWith, switchMap, tap } from 'rxjs/operators'
import { isError } from 'util'
import { gql } from '../../../shared/src/graphql'
import * as GQL from '../../../shared/src/graphqlschema'
import { LinkOrSpan } from '../../../shared/src/ui/generic/LinkOrSpan'
import { queryGraphQL } from '../backend/graphql'
import { HeroPage } from '../components/HeroPage'
import { Markdown } from '../components/Markdown'
import { PageTitle } from '../components/PageTitle'
import { eventLogger } from '../tracking/eventLogger'
import { asError, createAggregateError, ErrorLike, isErrorLike } from '../util/errors'
import { createLinkClickHandler } from '../util/linkClickHandler'
import { memoizeObservable } from '../util/memoize'

const queryDocPage = memoizeObservable(
    (path: string): Observable<GQL.IDocSitePage | null> =>
        queryGraphQL(
            gql`
                query DocSitePage($path: String!) {
                    docSitePage(path: $path) {
                        title
                        contentHTML
                        indexHTML
                        filePath
                    }
                }
            `,
            { path }
        ).pipe(
            map(({ data, errors }) => {
                if (!data || (errors && errors.length > 0)) {
                    throw createAggregateError(errors)
                }
                return data.docSitePage
            })
        ),
    path => path
)

interface Props extends RouteComponentProps<{}> {
    /** The path of the documentation page to display. */
    path: string
}

const LOADING: 'loading' = 'loading'

interface State {
    /** The documentation page, null for not found, loading, or an error. */
    pageOrError: GQL.IDocSitePage | null | typeof LOADING | ErrorLike
}

/**
 * A Markdown-rendered Sourcegraph documentation page.
 */
export class DocSitePage extends React.PureComponent<Props, State> {
    public state: State = { pageOrError: LOADING }

    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        const pathChanges = this.componentUpdates.pipe(
            map(({ path }) => path),
            distinctUntilChanged()
        )

        this.subscriptions.add(
            pathChanges
                .pipe(
                    switchMap(path =>
                        queryDocPage(path).pipe(
                            catchError(err => [asError(err)]),
                            startWith(LOADING)
                        )
                    ),
                    tap(result => {
                        if (result !== null && result !== LOADING && !isError(result)) {
                            eventLogger.logViewEvent('Docs', { docs_title: result.title })
                        }
                    }),
                    map(result => ({ pageOrError: result }))
                )
                .subscribe(stateUpdate => this.setState(stateUpdate))
        )

        this.componentUpdates.next(this.props)
    }

    public componentDidUpdate(prevProps: Props, prevState: State): void {
        this.componentUpdates.next(this.props)

        if (
            this.props.location.hash &&
            (this.props.location.hash !== prevProps.location.hash || this.state.pageOrError !== prevState.pageOrError)
        ) {
            setTimeout(() => (window.location.href = this.props.location.hash))
        }
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element {
        const pathParts = this.props.path ? this.props.path.split('/') : []
        const breadcrumb = (
            <div className="breadcrumb mb-3">
                <LinkOrSpan
                    to={pathParts.length === 0 ? undefined : this.props.match.path}
                    className={`breadcrumb-item ${pathParts.length === 0 ? 'active' : ''}`}
                >
                    Help
                </LinkOrSpan>
                {pathParts.map((path, i) => (
                    <Link
                        key={i}
                        to={`${this.props.match.path}/${pathParts.slice(0, i + 1).join('/')}`}
                        className={`breadcrumb-item ${i === pathParts.length - 1 ? 'active' : ''}`}
                    >
                        {path}
                    </Link>
                ))}
            </div>
        )

        return (
            <>
                {this.state.pageOrError === null ? (
                    <>
                        <PageTitle title="Help" />
                        <HeroPage
                            icon={MapSearchIcon}
                            title="404: Not Found"
                            subtitle="The requested documentation page was not found."
                        />
                    </>
                ) : isErrorLike(this.state.pageOrError) ? (
                    <>
                        <PageTitle title="Help" />
                        <HeroPage icon={ErrorIcon} title="Error" subtitle={this.state.pageOrError.message} />
                    </>
                ) : (
                    <div className="doc-site-page container px-0 d-flex flex-column flex-grow-1">
                        <div
                            onClick={createLinkClickHandler(this.props.history)}
                            className="doc-site-page__page flex-grow-1 d-flex flex-wrap flex-row-reverse"
                        >
                            <PageTitle
                                title={
                                    this.state.pageOrError === LOADING
                                        ? 'Help'
                                        : `${this.state.pageOrError.title} - Help`
                                }
                            />
                            <div
                                className="doc-site-page__index p-4"
                                dangerouslySetInnerHTML={
                                    this.state.pageOrError !== LOADING && this.state.pageOrError.indexHTML
                                        ? { __html: this.state.pageOrError.indexHTML }
                                        : undefined
                                }
                            />
                            <div className="doc-site-page__content p-4">
                                {breadcrumb}
                                {this.state.pageOrError === LOADING ? (
                                    <LoadingSpinner className="icon-inline" />
                                ) : (
                                    <Markdown dangerousInnerHTML={this.state.pageOrError.contentHTML} />
                                )}
                            </div>
                        </div>
                    </div>
                )}
            </>
        )
    }
}
