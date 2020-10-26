import * as H from 'history'
import React from 'react'
import { combineLatest, merge, Observable, Subject, Subscription } from 'rxjs'
import { distinctUntilChanged, first, map, scan, startWith, switchMap, tap } from 'rxjs/operators'
import { Subtract } from 'utility-types'
import {
    ActivationCompletionStatus,
    ActivationProps,
    ActivationStep,
} from '../../../shared/src/components/activation/Activation'
import { dataOrThrowErrors, gql } from '../../../shared/src/graphql/graphql'
import { queryGraphQL } from '../backend/graphql'
import { logUserEvent, logEvent } from '../user/settings/backend'
import { AuthenticatedUser } from '../auth'
import { UserEvent } from '../../../shared/src/graphql-operations'

/**
 * Fetches activation status from server.
 */
const fetchActivationStatus = (isSiteAdmin: boolean): Observable<ActivationCompletionStatus> =>
    queryGraphQL(
        isSiteAdmin
            ? gql`
                  query SiteAdminActivationStatus {
                      externalServices {
                          totalCount
                      }
                      repositories {
                          totalCount
                      }
                      viewerSettings {
                          final
                      }
                      users {
                          totalCount
                      }
                      currentUser {
                          usageStatistics {
                              searchQueries
                              findReferencesActions
                              codeIntelligenceActions
                          }
                      }
                  }
              `
            : gql`
                  query ActivationStatus {
                      currentUser {
                          usageStatistics {
                              searchQueries
                              findReferencesActions
                              codeIntelligenceActions
                          }
                      }
                  }
              `
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            const authProviders = window.context.authProviders
            const usageStats = !!data.currentUser && data.currentUser.usageStatistics
            const completed: ActivationCompletionStatus = {
                DidSearch: usageStats && usageStats.searchQueries > 0,
                FoundReferences:
                    // TODO(beyang): revert this to usageStats.findReferencesActions > 0 in 3.3 or later.
                    // Remove codeIntelligenceActions from the GraphQL query above, as well.
                    usageStats && (usageStats.findReferencesActions > 0 || usageStats.codeIntelligenceActions > 10),
            }
            if (isSiteAdmin) {
                completed.ConnectedCodeHost = data.externalServices && data.externalServices.totalCount > 0
                if (authProviders) {
                    completed.EnabledSharing =
                        data.users.totalCount > 1 || authProviders.filter(provider => !provider.isBuiltin).length > 0
                }
            }
            return completed
        })
    )

/**
 * Returns the link a user should go to when they click on the uncompleted find-references
 * activation step. For now, this links to root page of a repository, but we could improve
 * this by linking to a code file or actual symbol.
 */
const fetchReferencesLink = (): Observable<string | null> =>
    queryGraphQL(gql`
        query LinksForRepositories {
            repositories(cloned: true, first: 100, indexed: true) {
                nodes {
                    url
                    gitRefs {
                        totalCount
                    }
                }
            }
        }
    `).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.repositories.nodes) {
                return null
            }
            const repositoryURLs = data.repositories.nodes
                .filter(repository => repository.gitRefs && repository.gitRefs.totalCount > 0)
                .sort((repository1, repository2) => repository2.gitRefs.totalCount - repository1.gitRefs.totalCount)
                .map(repository => repository.url)
            if (repositoryURLs.length === 0) {
                return null
            }
            return repositoryURLs[0]
        })
    )

/**
 * Gets the activation steps that need to be completed for a given user.
 */
const getActivationSteps = (authenticatedUser: AuthenticatedUser): ActivationStep[] => {
    const sources: (ActivationStep & { siteAdminOnly?: boolean })[] = [
        {
            id: 'ConnectedCodeHost',
            title: 'Add repositories',
            detail: 'Configure Sourcegraph to talk to your code host and fetch a list of your repositories.',
            onClick: (event: React.MouseEvent<HTMLElement>, history: H.History) => {
                event.preventDefault()
                history.push('/site-admin/external-services/new')
            },
            siteAdminOnly: true,
        },
        {
            id: 'DidSearch',
            title: 'Search your code',
            detail: (
                <span>
                    Head to the <a href="/search">homepage</a> and perform a search query on your code.{' '}
                    <strong>Example:</strong> type 'lang:' and select a language
                </span>
            ),
        },
        {
            id: 'FoundReferences',
            title: 'Find some references',
            detail:
                'To find references of a token, navigate to a code file in one of your repositories, hover over a token to activate the tooltip, and then click "Find references".',
            onClick: (event: React.MouseEvent<HTMLElement>, history: H.History) =>
                fetchReferencesLink()
                    .pipe(first())
                    .subscribe(link => {
                        if (link) {
                            history.push(link)
                        } else {
                            alert('Must add repositories before finding references')
                        }
                    }),
        },
        {
            id: 'EnabledSharing',
            title: 'Configure SSO or share with teammates',
            detail: 'Configure a single-sign on (SSO) provider or have at least one other teammate sign up.',
            onClick: () => {
                window.open('https://docs.sourcegraph.com/admin/auth', '_blank', 'height=200,width=200')
                window.open('/site-admin/configuration', '_self')
            },
            siteAdminOnly: true,
        },
    ]
    return sources
        .filter(source => authenticatedUser.siteAdmin || !source.siteAdminOnly)
        .map(({ siteAdminOnly, ...step }) => step)
}

/**
 * Sends update to server for events that don't themselves trigger
 * an update.
 */
const recordUpdate = (update: Partial<ActivationCompletionStatus>): void => {
    if (update.FoundReferences) {
        logUserEvent(UserEvent.CODEINTELREFS)
        logEvent('CodeIntelRefs')
    }
}

interface WithActivationProps {
    authenticatedUser: AuthenticatedUser | null
}

interface WithActivationState {
    completed?: ActivationCompletionStatus
}

/**
 * Modifies the input component to return a component that includes the activation status in the
 * `activation` field of its props.
 */
export const withActivation = <P extends ActivationProps>(
    Component: React.ComponentType<P>
): React.ComponentType<WithActivationProps & Subtract<P, ActivationProps>> =>
    class WithActivation extends React.Component<
        WithActivationProps & Subtract<P, ActivationProps>,
        WithActivationState
    > {
        private subscriptions = new Subscription()
        private componentUpdates = new Subject<Readonly<WithActivationProps & Subtract<P, ActivationProps>>>()
        public state: WithActivationState = {}

        /**
         * Calling `next` triggers refetches. This ensures at most one refetch request is outstanding
         * at any given time.
         */
        private refetches = new Subject<void>()

        private updates = new Subject<Partial<ActivationCompletionStatus>>()

        public componentDidMount(): void {
            const authenticatedUser: Observable<AuthenticatedUser | null> = this.componentUpdates.pipe(
                startWith(this.props),
                map(props => props.authenticatedUser),
                distinctUntilChanged()
            )
            const serverCompletionStatus: Observable<ActivationCompletionStatus> = combineLatest([
                authenticatedUser,
                this.refetches.pipe(startWith(undefined)),
            ]).pipe(
                switchMap(([authenticatedUser]) =>
                    authenticatedUser ? fetchActivationStatus(authenticatedUser.siteAdmin) : []
                )
            )
            const localCompletionStatus: Observable<Partial<ActivationCompletionStatus>> = merge(
                authenticatedUser.pipe(map(() => null)), // reset on new authenticated user
                this.updates
            ).pipe(
                tap(update => update && recordUpdate(update)),
                scan<Partial<ActivationCompletionStatus> | null, Partial<ActivationCompletionStatus>>(
                    (previous, next) => (next ? { ...previous, ...next } : {}),
                    {}
                )
            )
            this.subscriptions.add(
                combineLatest([serverCompletionStatus, localCompletionStatus])
                    .pipe(
                        map(([serverCompletionStatus, localCompletionStatus]) => ({
                            ...serverCompletionStatus,
                            ...localCompletionStatus,
                        }))
                    )
                    .subscribe(completed => this.setState({ completed }))
            )
        }

        public componentWillUnmount(): void {
            this.subscriptions.unsubscribe()
        }

        public componentDidUpdate(): void {
            this.componentUpdates.next(this.props)
        }

        private steps(): ActivationStep[] | undefined {
            const user: AuthenticatedUser | null = this.props.authenticatedUser
            if (user) {
                return getActivationSteps(user)
            }
            return undefined
        }

        private refetchCompletionStatus = (): void => this.refetches.next()

        private updateCompletionStatus = (update: Partial<ActivationCompletionStatus>): void =>
            this.updates.next(update)

        public render(): React.ReactFragment | null {
            const steps = this.steps()
            const activationProps: ActivationProps = {
                activation: steps && {
                    steps,
                    completed: this.state.completed,
                    update: this.updateCompletionStatus,
                    refetch: this.refetchCompletionStatus,
                },
            }

            // Pass component props and activation props through to wrapped component.
            const props: Readonly<Subtract<P, ActivationProps>> = this.props
            const props2: Subtract<P, ActivationProps> = props
            const combinedProps = { ...props2, ...activationProps }
            // This is safe to cast to P, because props2 has everything in P *except*
            // the properties in ActivationProps
            const combinedProps2 = combinedProps as P
            return <Component {...combinedProps2} />
        }
    }
