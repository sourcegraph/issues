import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React, { useMemo } from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import * as GQL from '../../../../shared/src/graphql/schema'
import { HeroPage } from '../../components/HeroPage'
import { RepoContainerContext } from '../RepoContainer'
import { RepoHeaderBreadcrumbNavItem } from '../RepoHeaderBreadcrumbNavItem'
import { RepositoryReleasesTagsPage } from './RepositoryReleasesTagsPage'
import * as H from 'history'
import { BreadcrumbSetters } from '../../components/Breadcrumbs'

const NotFoundPage: React.FunctionComponent = () => (
    <HeroPage
        icon={MapSearchIcon}
        title="404: Not Found"
        subtitle="Sorry, the requested repository tags page was not found."
    />
)

interface Props
    extends RouteComponentProps<{}>,
        Pick<RepoContainerContext, 'repo' | 'routePrefix' | 'repoHeaderContributionsLifecycleProps'>,
        BreadcrumbSetters {
    repo: GQL.IRepository
    history: H.History
}

/**
 * Properties passed to all page components in the repository branches area.
 */
export interface RepositoryReleasesAreaPageProps {
    /**
     * The active repository.
     */
    repo: GQL.IRepository
}

/**
 * Renders pages related to repository branches.
 */
export const RepositoryReleasesArea: React.FunctionComponent<Props> = ({ useBreadcrumb, repo, routePrefix }) => {
    useBreadcrumb(
        useMemo(
            () => ({
                key: 'tags',
                element: <RepoHeaderBreadcrumbNavItem key="tags">Tags</RepoHeaderBreadcrumbNavItem>,
            }),
            []
        )
    )

    const transferProps: { repo: GQL.IRepository } = {
        repo,
    }

    return (
        <div className="repository-graph-area">
            <div className="container">
                <div className="container-inner">
                    <Switch>
                        {/* eslint-disable react/jsx-no-bind */}
                        <Route
                            path={`${routePrefix}/-/tags`}
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
