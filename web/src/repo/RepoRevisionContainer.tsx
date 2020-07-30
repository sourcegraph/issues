import { isEqual } from 'lodash'
import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import MenuDownIcon from 'mdi-react/MenuDownIcon'
import * as React from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import { UncontrolledPopover } from 'reactstrap'
import { defer, Subject, Subscription } from 'rxjs'
import { catchError, delay, distinctUntilChanged, map, retryWhen, switchMap, tap } from 'rxjs/operators'
import {
    CloneInProgressError,
    isCloneInProgressErrorLike,
    isRevisionNotFoundErrorLike,
    isRepoNotFoundErrorLike,
} from '../../../shared/src/backend/errors'
import { ActivationProps } from '../../../shared/src/components/activation/Activation'
import { ExtensionsControllerProps } from '../../../shared/src/extensions/controller'
import * as GQL from '../../../shared/src/graphql/schema'
import { PlatformContextProps } from '../../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../../shared/src/settings/settings'
import { ErrorLike, isErrorLike } from '../../../shared/src/util/errors'
import { HeroPage } from '../components/HeroPage'
import { ChromeExtensionToast } from '../marketing/BrowserExtensionToast'
import { IS_CHROME } from '../marketing/util'
import { ThemeProps } from '../../../shared/src/theme'
import { EventLoggerProps } from '../tracking/eventLogger'
import { RouteDescriptor } from '../util/contributions'
import { CopyLinkAction } from './actions/CopyLinkAction'
import { GoToPermalinkAction } from './actions/GoToPermalinkAction'
import { ResolvedRevision, resolveRevision } from './backend'
import { RepoContainerContext } from './RepoContainer'
import { RepoHeaderContributionsLifecycleProps } from './RepoHeader'
import { RepoHeaderContributionPortal } from './RepoHeaderContributionPortal'
import { EmptyRepositoryPage, RepositoryCloningInProgressPage } from './RepositoryGitDataContainer'
import { RevisionsPopover } from './RevisionsPopover'
import { PatternTypeProps, CaseSensitivityProps, CopyQueryButtonProps } from '../search'
import { RepoSettingsAreaRoute } from './settings/RepoSettingsArea'
import { ErrorMessage } from '../components/alerts'
import * as H from 'history'
import { VersionContextProps } from '../../../shared/src/search/util'
import { RevisionSpec } from '../../../shared/src/util/url'
import { RepoSettingsSideBarGroup } from './settings/RepoSettingsSidebar'
import { OptionalAuthProps } from '../auth'

/** Props passed to sub-routes of {@link RepoRevisionContainer}. */
export interface RepoRevisionContainerContext
    extends RepoHeaderContributionsLifecycleProps,
        SettingsCascadeProps,
        ExtensionsControllerProps,
        PlatformContextProps,
        ThemeProps,
        EventLoggerProps,
        ActivationProps,
        Pick<
            RepoContainerContext,
            Exclude<keyof RepoContainerContext, 'onDidUpdateRepository' | 'onDidUpdateExternalLinks'>
        >,
        PatternTypeProps,
        CaseSensitivityProps,
        CopyQueryButtonProps,
        VersionContextProps,
        RevisionSpec {
    repo: GQL.Repository
    resolvedRev: ResolvedRevision

    /** The URL route match for {@link RepoRevisionContainer}. */
    routePrefix: string

    globbing: boolean
}

/** A sub-route of {@link RepoRevisionContainer}. */
export interface RepoRevisionContainerRoute extends RouteDescriptor<RepoRevisionContainerContext> {}

interface RepoRevisionContainerProps
    extends RouteComponentProps<{}>,
        RepoHeaderContributionsLifecycleProps,
        SettingsCascadeProps,
        PlatformContextProps,
        EventLoggerProps,
        ExtensionsControllerProps,
        ThemeProps,
        ActivationProps,
        PatternTypeProps,
        CaseSensitivityProps,
        CopyQueryButtonProps,
        VersionContextProps,
        OptionalAuthProps,
        RevisionSpec {
    routes: readonly RepoRevisionContainerRoute[]
    repoSettingsAreaRoutes: readonly RepoSettingsAreaRoute[]
    repoSettingsSidebarGroups: readonly RepoSettingsSideBarGroup[]
    repo: GQL.Repository
    routePrefix: string

    /**
     * The resolved revision or an error if it could not be resolved. This value lives in RepoContainer (this
     * component's parent) but originates from this component.
     */
    resolvedRevisionOrError?: ResolvedRevision | ErrorLike

    /** Called when the resolvedRevOrError state in this component's parent should be updated. */
    onResolvedRevisionOrError: (v: ResolvedRevision | ErrorLike | undefined) => void
    history: H.History

    globbing: boolean
}

interface RepoRevisionContainerState {}

/**
 * A container for a repository page that incorporates revisioned Git data. (For example,
 * blob and tree pages are revisioned, but the repository settings page is not.)
 */
export class RepoRevisionContainer extends React.PureComponent<RepoRevisionContainerProps, RepoRevisionContainerState> {
    public state: RepoRevisionContainerState = {}

    private propsUpdates = new Subject<RepoRevisionContainerProps>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        const repoRevisionChanges = this.propsUpdates.pipe(
            // Pick repoName and revision out of the props
            map(props => ({ repoName: props.repo.name, revision: props.revision })),
            distinctUntilChanged((a, b) => isEqual(a, b))
        )

        // Fetch repository revision.
        this.subscriptions.add(
            repoRevisionChanges
                .pipe(
                    // Reset resolved revision / error state
                    tap(() => this.props.onResolvedRevisionOrError(undefined)),
                    switchMap(({ repoName, revision }) =>
                        defer(() => resolveRevision({ repoName, revision })).pipe(
                            // On a CloneInProgress error, retry after 1s
                            retryWhen(errors =>
                                errors.pipe(
                                    tap(error => {
                                        if (isCloneInProgressErrorLike(error)) {
                                            // Display cloning screen to the user and retry
                                            this.props.onResolvedRevisionOrError(error)
                                            return
                                        }
                                        // Display error to the user and do not retry
                                        throw error
                                    }),
                                    delay(1000)
                                )
                            ),
                            // Save any error in the sate to display to the user
                            catchError(error => {
                                this.props.onResolvedRevisionOrError(error)
                                return []
                            })
                        )
                    )
                )
                .subscribe(
                    resolvedRevision => {
                        this.props.onResolvedRevisionOrError(resolvedRevision)
                    },
                    error => {
                        // Should never be reached because errors are caught above
                        console.error(error)
                    }
                )
        )

        this.propsUpdates.next(this.props)
    }

    public componentDidUpdate(): void {
        this.propsUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        if (!this.props.resolvedRevisionOrError) {
            // Render nothing while loading
            return null
        }

        if (isErrorLike(this.props.resolvedRevisionOrError)) {
            // Show error page
            if (isCloneInProgressErrorLike(this.props.resolvedRevisionOrError)) {
                return (
                    <RepositoryCloningInProgressPage
                        repoName={this.props.repo.name}
                        progress={(this.props.resolvedRevisionOrError as CloneInProgressError).progress}
                    />
                )
            }
            if (isRepoNotFoundErrorLike(this.props.resolvedRevisionOrError)) {
                return (
                    <HeroPage
                        icon={MapSearchIcon}
                        title="404: Not Found"
                        subtitle="The requested repository was not found."
                    />
                )
            }
            if (isRevisionNotFoundErrorLike(this.props.resolvedRevisionOrError)) {
                if (!this.props.revision) {
                    return <EmptyRepositoryPage />
                }
                return (
                    <HeroPage
                        icon={MapSearchIcon}
                        title="404: Not Found"
                        subtitle="The requested revision was not found."
                    />
                )
            }
            return (
                <HeroPage
                    icon={AlertCircleIcon}
                    title="Error"
                    subtitle={<ErrorMessage error={this.props.resolvedRevisionOrError} history={this.props.history} />}
                />
            )
        }

        const context: RepoRevisionContainerContext = {
            platformContext: this.props.platformContext,
            extensionsController: this.props.extensionsController,
            isLightTheme: this.props.isLightTheme,
            telemetryService: this.props.telemetryService,
            activation: this.props.activation,
            repo: this.props.repo,
            repoHeaderContributionsLifecycleProps: this.props.repoHeaderContributionsLifecycleProps,
            resolvedRev: this.props.resolvedRevisionOrError,
            revision: this.props.revision,
            routePrefix: this.props.routePrefix,
            authenticatedUser: this.props.authenticatedUser,
            settingsCascade: this.props.settingsCascade,
            patternType: this.props.patternType,
            setPatternType: this.props.setPatternType,
            caseSensitive: this.props.caseSensitive,
            setCaseSensitivity: this.props.setCaseSensitivity,
            repoSettingsAreaRoutes: this.props.repoSettingsAreaRoutes,
            repoSettingsSidebarGroups: this.props.repoSettingsSidebarGroups,
            copyQueryButton: this.props.copyQueryButton,
            versionContext: this.props.versionContext,
            globbing: this.props.globbing,
        }

        return (
            <div className="repo-revision-container">
                {IS_CHROME && <ChromeExtensionToast />}
                <RepoHeaderContributionPortal
                    position="nav"
                    priority={100}
                    element={
                        <div className="d-flex align-items-center" key="repo-revision">
                            <span className="test-revision">
                                {(this.props.revision &&
                                this.props.revision === this.props.resolvedRevisionOrError.commitID
                                    ? this.props.resolvedRevisionOrError.commitID.slice(0, 7)
                                    : this.props.revision) ||
                                    this.props.resolvedRevisionOrError.defaultBranch ||
                                    'HEAD'}
                            </span>
                            <button type="button" id="repo-revision-popover" className="btn btn-link px-0">
                                <MenuDownIcon className="icon-inline" />
                            </button>
                            <UncontrolledPopover
                                placement="bottom-start"
                                target="repo-revision-popover"
                                trigger="legacy"
                            >
                                <RevisionsPopover
                                    repo={this.props.repo.id}
                                    repoName={this.props.repo.name}
                                    defaultBranch={this.props.resolvedRevisionOrError.defaultBranch}
                                    currentRev={this.props.revision}
                                    currentCommitID={this.props.resolvedRevisionOrError.commitID}
                                    history={this.props.history}
                                    location={this.props.location}
                                />
                            </UncontrolledPopover>
                        </div>
                    }
                    repoHeaderContributionsLifecycleProps={this.props.repoHeaderContributionsLifecycleProps}
                />
                <Switch>
                    {/* eslint-disable react/jsx-no-bind */}
                    {this.props.routes.map(
                        ({ path, render, exact, condition = () => true }) =>
                            condition(context) && (
                                <Route
                                    path={this.props.routePrefix + path}
                                    key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                                    exact={exact}
                                    render={routeComponentProps => render({ ...context, ...routeComponentProps })}
                                />
                            )
                    )}
                    {/* eslint-enable react/jsx-no-bind */}
                </Switch>
                <RepoHeaderContributionPortal
                    position="left"
                    element={<CopyLinkAction key="copy-link" location={this.props.location} />}
                    repoHeaderContributionsLifecycleProps={this.props.repoHeaderContributionsLifecycleProps}
                />
                <RepoHeaderContributionPortal
                    position="right"
                    priority={3}
                    element={
                        <GoToPermalinkAction
                            key="go-to-permalink"
                            revision={this.props.revision}
                            commitID={this.props.resolvedRevisionOrError.commitID}
                            location={this.props.location}
                            history={this.props.history}
                        />
                    }
                    repoHeaderContributionsLifecycleProps={this.props.repoHeaderContributionsLifecycleProps}
                />
            </div>
        )
    }
}
