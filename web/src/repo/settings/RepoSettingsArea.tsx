import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import DoNotDisturbIcon from 'mdi-react/DoNotDisturbIcon'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import * as React from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import { Subject, Subscription } from 'rxjs'
import { switchMap, map, distinctUntilChanged } from 'rxjs/operators'
import * as GQL from '../../../../shared/src/graphql/schema'
import { HeroPage } from '../../components/HeroPage'
import { RepoHeaderContributionsLifecycleProps } from '../RepoHeader'
import { RepoHeaderContributionPortal } from '../RepoHeaderContributionPortal'
import { fetchRepository } from './backend'
import { RepoSettingsSidebar, RepoSettingsSideBarGroups } from './RepoSettingsSidebar'
import { RouteDescriptor } from '../../util/contributions'
import { ErrorMessage } from '../../components/alerts'
import { asError } from '../../../../shared/src/util/errors'
import * as H from 'history'
import { OptionalAuthProps } from '../../auth'

const NotFoundPage: React.FunctionComponent = () => (
    <HeroPage
        icon={MapSearchIcon}
        title="404: Not Found"
        subtitle="Sorry, the requested repository page was not found."
    />
)

export interface RepoSettingsAreaRouteContext {
    repo: GQL.Repository
    onDidUpdateRepository: (update: Partial<GQL.Repository>) => void
}

export interface RepoSettingsAreaRoute extends RouteDescriptor<RepoSettingsAreaRouteContext> {}

interface Props extends RouteComponentProps<{}>, RepoHeaderContributionsLifecycleProps, OptionalAuthProps {
    repoSettingsAreaRoutes: readonly RepoSettingsAreaRoute[]
    repoSettingsSidebarGroups: RepoSettingsSideBarGroups
    repo: GQL.Repository
    onDidUpdateRepository: (update: Partial<GQL.Repository>) => void
    history: H.History
}

interface State {
    repo?: GQL.Repository | null
    error?: string
}

/**
 * Renders a layout of a sidebar and a content area to display pages related to
 * a repository's settings.
 */
export class RepoSettingsArea extends React.Component<Props> {
    public state: State = {}

    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    map(props => props.repo.name),
                    distinctUntilChanged(),
                    switchMap(name => fetchRepository(name))
                )
                .subscribe(
                    repo => this.setState({ repo }),
                    error => this.setState({ error: asError(error).message })
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
        if (this.state.error) {
            return (
                <HeroPage
                    icon={AlertCircleIcon}
                    title="Error"
                    subtitle={<ErrorMessage error={this.state.error} history={this.props.history} />}
                />
            )
        }

        if (this.state.repo === undefined) {
            return null
        }
        if (this.state.repo === null) {
            return <NotFoundPage />
        }
        if (!this.state.repo.viewerCanAdminister) {
            return (
                <HeroPage
                    icon={DoNotDisturbIcon}
                    title="Forbidden"
                    subtitle="You are not authorized to view or change this repository's settings."
                />
            )
        }
        if (!this.props.authenticatedUser) {
            return null
        }

        const context: RepoSettingsAreaRouteContext = {
            repo: this.state.repo,
            onDidUpdateRepository: this.props.onDidUpdateRepository,
        }

        return (
            <div className="repo-settings-area container d-flex mt-3">
                <RepoHeaderContributionPortal
                    position="nav"
                    element={
                        <span key="graph" className="repo-settings-area__header-item">
                            Settings
                        </span>
                    }
                    repoHeaderContributionsLifecycleProps={this.props.repoHeaderContributionsLifecycleProps}
                />
                <RepoSettingsSidebar className="flex-0 mr-3" {...this.props} {...context} />
                <div className="flex-1">
                    <Switch>
                        {this.props.repoSettingsAreaRoutes.map(
                            ({ render, path, exact, condition = () => true }) =>
                                /* eslint-disable react/jsx-no-bind */
                                condition(context) && (
                                    <Route
                                        // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                                        key="hardcoded-key"
                                        path={this.props.match.url + path}
                                        exact={exact}
                                        render={routeComponentProps => render({ ...context, ...routeComponentProps })}
                                    />
                                )
                        )}

                        <Route key="hardcoded-key" component={NotFoundPage} />
                    </Switch>
                </div>
            </div>
        )
    }
}
