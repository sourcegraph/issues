import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import * as React from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import { Subscription } from 'rxjs'
import * as GQL from '../../../../shared/src/graphql/schema'
import { HeroPage } from '../../components/HeroPage'
import { RepoContainerContext } from '../RepoContainer'
import { RepoHeaderBreadcrumbNavItem } from '../RepoHeaderBreadcrumbNavItem'
import { RepoHeaderContributionPortal } from '../RepoHeaderContributionPortal'
import { RepositoryReleasesTagsPage } from './RepositoryReleasesTagsPage'
import { ErrorMessage } from '../../components/alerts'
import * as H from 'history'

const NotFoundPage: React.FunctionComponent = () => (
    <HeroPage
        icon={MapSearchIcon}
        title="404: Not Found"
        subtitle="Sorry, the requested repository tags page was not found."
    />
)

interface Props
    extends RouteComponentProps<{}>,
        Pick<RepoContainerContext, 'repo' | 'routePrefix' | 'repoHeaderContributionsLifecycleProps'> {
    repo: GQL.Repository
    history: H.History
}

interface State {
    error?: string
}

/**
 * Properties passed to all page components in the repository branches area.
 */
export interface RepositoryReleasesAreaPageProps {
    /**
     * The active repository.
     */
    repo: GQL.Repository
}

/**
 * Renders pages related to repository branches.
 */
export class RepositoryReleasesArea extends React.Component<Props> {
    public state: State = {}

    private subscriptions = new Subscription()

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        if (this.state.error) {
            return (
                <HeroPage
                    icon={AlertCircleIcon}
                    title="Error"
                    subtitle={<ErrorMessage error={this.state.error} history={this.props.history} />}
                />
            )
        }

        const transferProps: { repo: GQL.Repository } = {
            repo: this.props.repo,
        }

        return (
            <div className="repository-graph-area">
                <RepoHeaderContributionPortal
                    position="nav"
                    element={<RepoHeaderBreadcrumbNavItem key="tags">Tags</RepoHeaderBreadcrumbNavItem>}
                    repoHeaderContributionsLifecycleProps={this.props.repoHeaderContributionsLifecycleProps}
                />
                <div className="container">
                    <div className="container-inner">
                        <Switch>
                            {/* eslint-disable react/jsx-no-bind */}
                            <Route
                                path={`${this.props.routePrefix}/-/tags`}
                                key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                                exact={true}
                                render={routeComponentProps => (
                                    <RepositoryReleasesTagsPage {...routeComponentProps} {...transferProps} />
                                )}
                            />
                            <Route key="hardcoded-key" component={NotFoundPage} />
                            {/* eslint-enable react/jsx-no-bind */}
                        </Switch>
                    </div>
                </div>
            </div>
        )
    }
}
