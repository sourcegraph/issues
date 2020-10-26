import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { concat, of, Subject, Subscription } from 'rxjs'
import {
    catchError,
    delay,
    distinctUntilChanged,
    map,
    mapTo,
    mergeMap,
    startWith,
    switchMap,
    tap,
} from 'rxjs/operators'
import * as GQL from '../../../shared/src/graphql/schema'
import { asError, ErrorLike, isErrorLike } from '../../../shared/src/util/errors'
import { NamespaceProps } from '../namespaces'
import { fetchSavedSearch, updateSavedSearch } from '../search/backend'
import { eventLogger } from '../tracking/eventLogger'
import { SavedQueryFields, SavedSearchForm } from './SavedSearchForm'
import { AuthenticatedUser } from '../auth'

interface Props extends RouteComponentProps<{ id: GQL.ID }>, NamespaceProps {
    authenticatedUser: AuthenticatedUser | null
}

const LOADING = 'loading' as const

interface State {
    savedSearchOrError: typeof LOADING | GQL.ISavedSearch | ErrorLike
    updatedOrError: null | true | typeof LOADING | ErrorLike
}

export class SavedSearchUpdateForm extends React.Component<Props, State> {
    constructor(props: Props) {
        super(props)
        this.state = {
            savedSearchOrError: LOADING,
            updatedOrError: null,
        }
    }

    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()
    private submits = new Subject<SavedQueryFields>()

    public componentDidMount(): void {
        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    map(props => props.match.params.id),
                    distinctUntilChanged(),
                    switchMap(id =>
                        fetchSavedSearch(id).pipe(
                            startWith(LOADING),
                            catchError(error => [asError(error)])
                        )
                    ),
                    map(result => ({ savedSearchOrError: result }))
                )
                .subscribe(stateUpdate => this.setState(stateUpdate))
        )

        this.subscriptions.add(
            this.submits
                .pipe(
                    switchMap(input =>
                        concat(
                            [{ updatedOrError: LOADING }],
                            updateSavedSearch(
                                input.id,
                                input.description,
                                input.query,
                                input.notify,
                                input.notifySlack,
                                this.props.namespace.__typename === 'User' ? this.props.namespace.id : null,
                                this.props.namespace.__typename === 'Org' ? this.props.namespace.id : null
                            ).pipe(
                                mapTo(null),
                                tap(() => eventLogger.log('SavedSearchUpdated')),
                                mergeMap(() =>
                                    concat(
                                        // Flash "updated" text
                                        of({ updatedOrError: true }),
                                        // Hide "updated" text again after 1s
                                        of({ updatedOrError: null }).pipe(delay(1000))
                                    )
                                ),
                                catchError((error: Error) => [{ updatedOrError: asError(error) }])
                            )
                        )
                    )
                )
                .subscribe(stateUpdate => this.setState(stateUpdate as State))
        )

        this.componentUpdates.next(this.props)

        eventLogger.logViewEvent('UpdateSavedSearchPage')
    }

    public render(): JSX.Element | null {
        const savedSearch =
            (!isErrorLike(this.state.savedSearchOrError) &&
                this.state.savedSearchOrError !== LOADING &&
                this.state.savedSearchOrError) ||
            undefined

        return (
            <div>
                {this.state.savedSearchOrError === LOADING && <LoadingSpinner className="icon-inline" />}
                {this.props.authenticatedUser && savedSearch && (
                    /* eslint-disable react/jsx-no-bind */
                    <SavedSearchForm
                        {...this.props}
                        submitLabel="Update saved search"
                        title="Manage saved search"
                        defaultValues={{
                            id: savedSearch.id,
                            description: savedSearch.description,
                            query: savedSearch.query,
                            notify: savedSearch.notify,
                            notifySlack: savedSearch.notifySlack,
                            slackWebhookURL: savedSearch.slackWebhookURL,
                        }}
                        loading={this.state.updatedOrError === LOADING}
                        onSubmit={(fields: Pick<SavedQueryFields, Exclude<keyof SavedQueryFields, 'id'>>): void =>
                            this.onSubmit({ id: savedSearch.id, ...fields })
                        }
                        error={isErrorLike(this.state.updatedOrError) ? this.state.updatedOrError : undefined}
                    />
                    /* eslint-enable react/jsx-no-bind */
                )}
                {this.state.updatedOrError === true && (
                    <p className="alert alert-success user-settings-profile-page__alert">Updated!</p>
                )}
            </div>
        )
    }

    private onSubmit = (fields: SavedQueryFields): void => {
        this.submits.next(fields)
    }
}
