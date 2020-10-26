import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import CloudDownloadIcon from 'mdi-react/CloudDownloadIcon'
import CloudOutlineIcon from 'mdi-react/CloudOutlineIcon'
import SettingsIcon from 'mdi-react/SettingsIcon'
import React, { useEffect, useCallback } from 'react'
import { RouteComponentProps } from 'react-router'
import { Link } from 'react-router-dom'
import { Observable } from 'rxjs'
import { RepoLink } from '../../../shared/src/components/RepoLink'
import {
    FilteredConnection,
    FilteredConnectionFilter,
    FilteredConnectionQueryArguments,
} from '../components/FilteredConnection'
import { PageTitle } from '../components/PageTitle'
import { refreshSiteFlags } from '../site/backend'
import { fetchAllRepositoriesAndPollIfEmptyOrAnyCloning } from './backend'
import { TelemetryProps } from '../../../shared/src/telemetry/telemetryService'
import { RepositoriesResult, SiteAdminRepositoryFields } from '../graphql-operations'

interface RepositoryNodeProps {
    node: SiteAdminRepositoryFields
}

const RepositoryNode: React.FunctionComponent<RepositoryNodeProps> = ({ node }) => (
    <li
        className="repository-node list-group-item py-2"
        data-test-repository={node.name}
        data-test-cloned={node.mirrorInfo.cloned}
    >
        <div className="d-flex align-items-center justify-content-between">
            <div>
                <RepoLink repoName={node.name} to={node.url} />
                {node.mirrorInfo.cloneInProgress && (
                    <small className="ml-2 text-success">
                        <LoadingSpinner className="icon-inline" /> Cloning
                    </small>
                )}
                {!node.mirrorInfo.cloneInProgress && !node.mirrorInfo.cloned && (
                    <small
                        className="ml-2 text-muted"
                        data-tooltip="Visit the repository to clone it. See its mirroring settings for diagnostics."
                    >
                        <CloudOutlineIcon className="icon-inline" /> Not yet cloned
                    </small>
                )}
            </div>
            <div className="repository-node__actions">
                {!node.mirrorInfo.cloneInProgress && !node.mirrorInfo.cloned && (
                    <Link className="btn btn-sm btn-secondary" to={node.url}>
                        <CloudDownloadIcon className="icon-inline" /> Clone now
                    </Link>
                )}{' '}
                {
                    <Link
                        className="btn btn-secondary btn-sm"
                        to={`/${node.name}/-/settings`}
                        data-tooltip="Repository settings"
                    >
                        <SettingsIcon className="icon-inline" /> Settings
                    </Link>
                }{' '}
            </div>
        </div>
    </li>
)

interface Props extends RouteComponentProps<{}>, TelemetryProps {}

const FILTERS: FilteredConnectionFilter[] = [
    {
        label: 'All',
        id: 'all',
        tooltip: 'Show all repositories',
        args: {},
    },
    {
        label: 'Cloned',
        id: 'cloned',
        tooltip: 'Show cloned repositories only',
        args: { cloned: true, notCloned: false },
    },
    {
        label: 'Not cloned',
        id: 'not-cloned',
        tooltip: 'Show only repositories that have not been cloned yet',
        args: { cloned: false, notCloned: true },
    },
    {
        label: 'Needs index',
        id: 'needs-index',
        tooltip: 'Show only repositories that need to be indexed',
        args: { indexed: false },
    },
]

/**
 * A page displaying the repositories on this site.
 */
export const SiteAdminRepositoriesPage: React.FunctionComponent<Props> = ({ history, location, telemetryService }) => {
    useEffect(() => {
        telemetryService.logViewEvent('SiteAdminRepos')
    }, [telemetryService])

    // Refresh global alert about enabling repositories when the user visits & navigates away from this page.
    useEffect(() => {
        refreshSiteFlags()
            .toPromise()
            .then(null, error => console.error(error))
        return () => {
            refreshSiteFlags()
                .toPromise()
                .then(null, error => console.error(error))
        }
    }, [])
    const queryRepositories = useCallback(
        (args: FilteredConnectionQueryArguments): Observable<RepositoriesResult['repositories']> =>
            fetchAllRepositoriesAndPollIfEmptyOrAnyCloning(args),
        []
    )
    const showRepositoriesAddedBanner = new URLSearchParams(location.search).has('repositoriesUpdated')

    return (
        <div className="site-admin-repositories-page">
            <PageTitle title="Repositories - Admin" />
            {showRepositoriesAddedBanner && (
                <p className="alert alert-success">
                    Updating repositories. It may take a few moments to clone and index each repository. Repository
                    statuses are displayed below.
                </p>
            )}
            <h2>Repositories</h2>
            <p>
                Repositories are synced from connected{' '}
                <Link to="/site-admin/external-services">code host connections</Link>.
            </p>
            <FilteredConnection<SiteAdminRepositoryFields, Omit<RepositoryNodeProps, 'node'>>
                className="list-group list-group-flush mt-3"
                noun="repository"
                pluralNoun="repositories"
                queryConnection={queryRepositories}
                nodeComponent={RepositoryNode}
                filters={FILTERS}
                history={history}
                location={location}
            />
        </div>
    )
}
