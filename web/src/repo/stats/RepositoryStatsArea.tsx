import MapSearchIcon from 'mdi-react/MapSearchIcon'
import * as React from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import { Subscription } from 'rxjs'
import * as GQL from '../../../../shared/src/graphql/schema'
import { HeroPage } from '../../components/HeroPage'
import { RepoHeaderContributionsLifecycleProps } from '../RepoHeader'
import { RepoHeaderBreadcrumbNavItem } from '../RepoHeaderBreadcrumbNavItem'
import { RepoHeaderContributionPortal } from '../RepoHeaderContributionPortal'
import { RepositoryStatsContributorsPage } from './RepositoryStatsContributorsPage'
import { RepositoryStatsNavbar } from './RepositoryStatsNavbar'
import { PatternTypeProps } from '../../search'

const NotFoundPage: React.FunctionComponent = () => (
    <HeroPage
        icon={MapSearchIcon}
        title="404: Not Found"
        subtitle="Sorry, the requested repository stats page was not found."
    />
)

interface Props
    extends RouteComponentProps<{}>,
        RepoHeaderContributionsLifecycleProps,
        Omit<PatternTypeProps, 'setPatternType'> {
    repo: GQL.Repository
    globbing: boolean
}

/**
 * Properties passed to all page components in the repository stats area.
 */
export interface RepositoryStatsAreaPageProps {
    /**
     * The active repository.
     */
    repo: GQL.Repository
}

const showNavbar = false

/**
 * Renders pages related to repository stats.
 */
export class RepositoryStatsArea extends React.Component<Props> {
    private subscriptions = new Subscription()

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        const transferProps: RepositoryStatsAreaPageProps = {
            repo: this.props.repo,
        }

        return (
            <div className="repository-stats-area container mt-3">
                <RepoHeaderContributionPortal
                    position="nav"
                    element={<RepoHeaderBreadcrumbNavItem key="stats">Contributors</RepoHeaderBreadcrumbNavItem>}
                    repoHeaderContributionsLifecycleProps={this.props.repoHeaderContributionsLifecycleProps}
                />
                {showNavbar && <RepositoryStatsNavbar className="mb-3" repo={this.props.repo.name} />}
                <Switch>
                    {/* eslint-disable react/jsx-no-bind */}
                    <Route
                        path={`${this.props.match.url}/contributors`}
                        key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                        exact={true}
                        render={routeComponentProps => (
                            <RepositoryStatsContributorsPage
                                {...routeComponentProps}
                                {...transferProps}
                                patternType={this.props.patternType}
                                globbing={this.props.globbing}
                            />
                        )}
                    />
                    <Route key="hardcoded-key" component={NotFoundPage} />
                    {/* eslint-enable react/jsx-no-bind */}
                </Switch>
            </div>
        )
    }
}
