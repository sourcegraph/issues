import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React, { useEffect } from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import * as GQL from '../../../../shared/src/graphql/schema'
import { HeroPage } from '../../components/HeroPage'
import { RepoHeaderBreadcrumbNavItem } from '../RepoHeaderBreadcrumbNavItem'
import { RepositoryBranchesAllPage } from './RepositoryBranchesAllPage'
import { RepositoryBranchesNavbar } from './RepositoryBranchesNavbar'
import { RepositoryBranchesOverviewPage } from './RepositoryBranchesOverviewPage'
import { UpdateBreadcrumbsProps } from '../../components/Breadcrumbs'

const NotFoundPage: React.FunctionComponent = () => (
    <HeroPage
        icon={MapSearchIcon}
        title="404: Not Found"
        subtitle="Sorry, the requested repository branches page was not found."
    />
)

interface Props extends RouteComponentProps<{}>, UpdateBreadcrumbsProps {
    repo: GQL.IRepository
}

/**
 * Properties passed to all page components in the repository branches area.
 */
export interface RepositoryBranchesAreaPageProps {
    /**
     * The active repository.
     */
    repo: GQL.IRepository
}

/**
 * Renders pages related to repository branches.
 */
export const RepositoryBranchesArea: React.FunctionComponent<Props> = props => {
    const transferProps: { repo: GQL.IRepository } = {
        repo: props.repo,
    }

    useEffect(
        () => props.pushBreadcrumb(<RepoHeaderBreadcrumbNavItem key="branches">Branches</RepoHeaderBreadcrumbNavItem>),
        []
    )

    return (
        <div className="repository-branches-area container">
            <RepositoryBranchesNavbar className="my-3" repo={props.repo.name} />
            <Switch>
                {/* eslint-disable react/jsx-no-bind */}
                <Route
                    path={`${props.match.url}`}
                    key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                    exact={true}
                    render={routeComponentProps => (
                        <RepositoryBranchesOverviewPage {...routeComponentProps} {...transferProps} />
                    )}
                />
                <Route
                    path={`${props.match.url}/all`}
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
