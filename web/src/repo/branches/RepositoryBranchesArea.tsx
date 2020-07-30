import MapSearchIcon from 'mdi-react/MapSearchIcon'
import * as React from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import { Subscription } from 'rxjs'
import * as GQL from '../../../../shared/src/graphql/schema'
import { HeroPage } from '../../components/HeroPage'
import { RepoHeaderContributionsLifecycleProps } from '../RepoHeader'
import { RepoHeaderBreadcrumbNavItem } from '../RepoHeaderBreadcrumbNavItem'
import { RepoHeaderContributionPortal } from '../RepoHeaderContributionPortal'
import { RepositoryBranchesAllPage } from './RepositoryBranchesAllPage'
import { RepositoryBranchesNavbar } from './RepositoryBranchesNavbar'
import { RepositoryBranchesOverviewPage } from './RepositoryBranchesOverviewPage'

const NotFoundPage: React.FunctionComponent = () => (
    <HeroPage
        icon={MapSearchIcon}
        title="404: Not Found"
        subtitle="Sorry, the requested repository branches page was not found."
    />
)

interface Props extends RouteComponentProps<{}>, RepoHeaderContributionsLifecycleProps {
    repo: GQL.Repository
}

/**
 * Properties passed to all page components in the repository branches area.
 */
export interface RepositoryBranchesAreaPageProps {
    /**
     * The active repository.
     */
    repo: GQL.Repository
}

/**
 * Renders pages related to repository branches.
 */
export class RepositoryBranchesArea extends React.Component<Props> {
    private subscriptions = new Subscription()

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        const transferProps: { repo: GQL.Repository } = {
            repo: this.props.repo,
        }

        return (
            <div className="repository-branches-area container">
                <RepoHeaderContributionPortal
                    position="nav"
                    element={<RepoHeaderBreadcrumbNavItem key="branches">Branches</RepoHeaderBreadcrumbNavItem>}
                    repoHeaderContributionsLifecycleProps={this.props.repoHeaderContributionsLifecycleProps}
                />
                <RepositoryBranchesNavbar className="my-3" repo={this.props.repo.name} />
                <Switch>
                    {/* eslint-disable react/jsx-no-bind */}
                    <Route
                        path={`${this.props.match.url}`}
                        key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                        exact={true}
                        render={routeComponentProps => (
                            <RepositoryBranchesOverviewPage {...routeComponentProps} {...transferProps} />
                        )}
                    />
                    <Route
                        path={`${this.props.match.url}/all`}
                        key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                        exact={true}
                        render={routeComponentProps => (
                            <RepositoryBranchesAllPage {...routeComponentProps} {...transferProps} />
                        )}
                    />
                    <Route key="hardcoded-key" component={NotFoundPage} />
                    {/* eslint-enable react/jsx-no-bind */}
                </Switch>
            </div>
        )
    }
}
