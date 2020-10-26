import MapSearchIcon from 'mdi-react/MapSearchIcon'
import * as React from 'react'
import { merge, of, Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, map, switchMap } from 'rxjs/operators'
import { asError, ErrorLike, isErrorLike } from '../../../shared/src/util/errors'
import { HeroPage } from '../components/HeroPage'
import { checkMirrorRepositoryConnection } from '../site-admin/backend'
import { eventLogger } from '../tracking/eventLogger'

interface Props {
    /** The name of the repository. */
    repo: string

    /** Whether the viewer is a site admin. */
    viewerCanAdminister: boolean
}

interface State {
    /**
     * Whether the option to add the repository should be shown.
     */
    showAdd: boolean

    /**
     * Whether the site admin can add this repository. undefined while loading.
     */
    canAddOrError?: boolean | ErrorLike
}

/**
 * A page informing the user that an error occurred while trying to display the repository. It
 * attempts to present the user with actions to solve the problem.
 */
export class RepositoryNotFoundPage extends React.PureComponent<Props, State> {
    public state: State = {
        showAdd: false,
    }

    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        eventLogger.logViewEvent('RepositoryError')

        // Show/hide add.
        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    distinctUntilChanged(
                        (a, b) => a.repo === b.repo && a.viewerCanAdminister === b.viewerCanAdminister
                    ),
                    switchMap(({ repo, viewerCanAdminister }) => {
                        type PartialStateUpdate = Pick<State, 'showAdd' | 'canAddOrError'>
                        if (!viewerCanAdminister) {
                            return of({ showAdd: false, canAddOrError: undefined })
                        }
                        return merge<PartialStateUpdate>(
                            of({ showAdd: true, canAddOrError: undefined }),
                            checkMirrorRepositoryConnection({ name: repo }).pipe(
                                map(result => result.error === null),
                                catchError(error => [asError(error)]),
                                map((canAddOrError): PartialStateUpdate => ({ showAdd: true, canAddOrError }))
                            )
                        )
                    })
                )
                .subscribe(
                    stateUpdate => this.setState(stateUpdate),
                    error => console.error(error)
                )
        )

        this.componentUpdates.next(this.props)
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return (
            <HeroPage
                icon={MapSearchIcon}
                title="Repository not found"
                subtitle={
                    <div className="repository-not-found-page">
                        {this.state.showAdd && (
                            <div className="repository-not-found-page__section mt-3">
                                <div className="repository-not-found-page__section-inner">
                                    <div className="repository-not-found-page__section-description">
                                        {this.state.canAddOrError === undefined && (
                                            <>Checking whether this repository can be added...</>
                                        )}
                                        {(this.state.canAddOrError === false ||
                                            isErrorLike(this.state.canAddOrError)) && (
                                            <>
                                                <p>
                                                    If this is a private repository, check that this site is configured
                                                    with a token that has access to this repository.
                                                </p>

                                                <p>
                                                    If this is a public repository, check that this repository is
                                                    explicitly listed in an{' '}
                                                    <a href="/site-admin/external-services">
                                                        external service configuration
                                                    </a>
                                                    .
                                                </p>
                                            </>
                                        )}
                                        {this.state.canAddOrError === true && (
                                            <>
                                                As a site admin, you can add this repository to Sourcegraph to allow
                                                users to search and view it by{' '}
                                                <a href="/site-admin/external-services">
                                                    connecting an external service
                                                </a>
                                                .
                                            </>
                                        )}
                                    </div>
                                </div>
                            </div>
                        )}
                        {!this.state.showAdd && <p>To access this repository, contact the Sourcegraph admin.</p>}
                    </div>
                }
            />
        )
    }
}
