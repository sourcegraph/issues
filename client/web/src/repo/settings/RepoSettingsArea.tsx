import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import DoNotDisturbIcon from 'mdi-react/DoNotDisturbIcon'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React, { useMemo } from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import { of } from 'rxjs'
import { catchError } from 'rxjs/operators'
import * as GQL from '../../../../shared/src/graphql/schema'
import { HeroPage } from '../../components/HeroPage'
import { fetchRepository } from './backend'
import { RepoSettingsSidebar, RepoSettingsSideBarGroups } from './RepoSettingsSidebar'
import { RouteDescriptor } from '../../util/contributions'
import { ErrorMessage } from '../../components/alerts'
import { asError, ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import * as H from 'history'
import { useObservable } from '../../../../shared/src/util/useObservable'
import { BreadcrumbSetters } from '../../components/Breadcrumbs'
import { AuthenticatedUser } from '../../auth'
import { TelemetryProps } from '../../../../shared/src/telemetry/telemetryService'

const NotFoundPage: React.FunctionComponent = () => (
    <HeroPage
        icon={MapSearchIcon}
        title="404: Not Found"
        subtitle="Sorry, the requested repository page was not found."
    />
)

export interface RepoSettingsAreaRouteContext extends TelemetryProps {
    repo: GQL.IRepository
    onDidUpdateRepository: (update: Partial<GQL.IRepository>) => void
}

export interface RepoSettingsAreaRoute extends RouteDescriptor<RepoSettingsAreaRouteContext> {}

interface Props extends RouteComponentProps<{}>, BreadcrumbSetters, TelemetryProps {
    repoSettingsAreaRoutes: readonly RepoSettingsAreaRoute[]
    repoSettingsSidebarGroups: RepoSettingsSideBarGroups
    repo: GQL.IRepository
    authenticatedUser: AuthenticatedUser | null
    onDidUpdateRepository: (update: Partial<GQL.IRepository>) => void
    history: H.History
}

/**
 * Renders a layout of a sidebar and a content area to display pages related to
 * a repository's settings.
 */
export const RepoSettingsArea: React.FunctionComponent<Props> = ({
    useBreadcrumb,

    ...props
}) => {
    const repoName = props.repo.name
    const repoOrError = useObservable(
        useMemo(() => fetchRepository(repoName).pipe(catchError(error => of<ErrorLike>(asError(error)))), [repoName])
    )

    useBreadcrumb(useMemo(() => ({ key: 'settings', element: 'Settings' }), []))

    if (repoOrError === undefined) {
        return null
    }
    if (isErrorLike(repoOrError)) {
        return (
            <HeroPage
                icon={AlertCircleIcon}
                title="Error"
                subtitle={<ErrorMessage error={repoOrError.message} history={props.history} />}
            />
        )
    }
    if (repoOrError === null) {
        return <NotFoundPage />
    }
    if (!repoOrError.viewerCanAdminister) {
        return (
            <HeroPage
                icon={DoNotDisturbIcon}
                title="Forbidden"
                subtitle="You are not authorized to view or change this repository's settings."
            />
        )
    }

    if (!props.authenticatedUser) {
        return null
    }
    const context: RepoSettingsAreaRouteContext = {
        repo: repoOrError,
        onDidUpdateRepository: props.onDidUpdateRepository,
        telemetryService: props.telemetryService,
    }

    return (
        <div className="repo-settings-area container d-flex mt-3">
            <RepoSettingsSidebar className="flex-0 mr-3" {...props} {...context} />
            <div className="flex-1">
                <Switch>
                    {props.repoSettingsAreaRoutes.map(
                        ({ render, path, exact, condition = () => true }) =>
                            /* eslint-disable react/jsx-no-bind */
                            condition(context) && (
                                <Route
                                    // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                                    key="hardcoded-key"
                                    path={props.match.url + path}
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
