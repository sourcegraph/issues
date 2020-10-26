import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React, { useMemo } from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import * as GQL from '../../../../shared/src/graphql/schema'
import { HeroPage } from '../../components/HeroPage'
import { RepositoryStatsContributorsPage } from './RepositoryStatsContributorsPage'
import { RepositoryStatsNavbar } from './RepositoryStatsNavbar'
import { PatternTypeProps } from '../../search'
import { BreadcrumbSetters } from '../../components/Breadcrumbs'

const NotFoundPage: React.FunctionComponent = () => (
    <HeroPage
        icon={MapSearchIcon}
        title="404: Not Found"
        subtitle="Sorry, the requested repository stats page was not found."
    />
)

interface Props extends RouteComponentProps<{}>, BreadcrumbSetters, Omit<PatternTypeProps, 'setPatternType'> {
    repo: GQL.IRepository
    globbing: boolean
}

/**
 * Properties passed to all page components in the repository stats area.
 */
export interface RepositoryStatsAreaPageProps {
    /**
     * The active repository.
     */
    repo: GQL.IRepository
}

const showNavbar = false

/**
 * Renders pages related to repository stats.
 */
export const RepositoryStatsArea: React.FunctionComponent<Props> = ({
    useBreadcrumb,

    ...props
}) => {
    useBreadcrumb(useMemo(() => ({ key: 'contributors', element: 'Contributors' }), []))

    return (
        <div className="repository-stats-area container mt-3">
            {showNavbar && <RepositoryStatsNavbar className="mb-3" repo={props.repo.name} />}
            <Switch>
                {/* eslint-disable react/jsx-no-bind */}
                <Route
                    path={`${props.match.url}/contributors`}
                    key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                    exact={true}
                    render={routeComponentProps => (
                        <RepositoryStatsContributorsPage {...routeComponentProps} {...props} />
                    )}
                />
                <Route key="hardcoded-key" component={NotFoundPage} />
                {/* eslint-enable react/jsx-no-bind */}
            </Switch>
        </div>
    )
}
