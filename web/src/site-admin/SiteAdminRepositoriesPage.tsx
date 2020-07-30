import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import CloudDownloadIcon from 'mdi-react/CloudDownloadIcon'
import CloudOutlineIcon from 'mdi-react/CloudOutlineIcon'
import SettingsIcon from 'mdi-react/SettingsIcon'
import React, { useEffect, useCallback } from 'react'
import { RouteComponentProps } from 'react-router'
import { Link } from 'react-router-dom'
import { Subject, Observable } from 'rxjs'
import { ActivationProps } from '../../../shared/src/components/activation/Activation'
import { RepoLink } from '../../../shared/src/components/RepoLink'
import * as GQL from '../../../shared/src/graphql/schema'
import {
    FilteredConnection,
    FilteredConnectionFilter,
    FilteredConnectionQueryArgs,
} from '../components/FilteredConnection'
import { PageTitle } from '../components/PageTitle'
import { refreshSiteFlags } from '../site/backend'
import { eventLogger } from '../tracking/eventLogger'
import { fetchAllRepositoriesAndPollIfEmptyOrAnyCloning } from './backend'
import * as H from 'history'

interface RepositoryNodeProps extends ActivationProps {
    node: GQL.Repository
    onDidUpdate?: () => void
    history: H.History
}

const RepositoryNode: React.FunctionComponent<RepositoryNodeProps> = props => (
    <li
        className="repository-node list-group-item py-2"
        data-test-repository={props.node.name}
        data-test-cloned={props.node.mirrorInfo.cloned}
    >
        <div className="d-flex align-items-center justify-content-between">
            <div>
                <RepoLink repoName={props.node.name} to={props.node.url} />
                {props.node.mirrorInfo.cloneInProgress && (
                    <small className="ml-2 text-success">
                        <LoadingSpinner className="icon-inline" /> Cloning
                    </small>
                )}
                {!props.node.mirrorInfo.cloneInProgress && !props.node.mirrorInfo.cloned && (
                    <small
                        className="ml-2 text-muted"
                        data-tooltip="Visit the repository to clone it. See its mirroring settings for diagnostics."
                    >
                        <CloudOutlineIcon className="icon-inline" /> Not yet cloned
                    </small>
                )}
            </div>
            <div className="repository-node__actions">
                {!props.node.mirrorInfo.cloneInProgress && !props.node.mirrorInfo.cloned && (
                    <Link className="btn btn-sm btn-secondary" to={props.node.url}>
                        <CloudDownloadIcon className="icon-inline" /> Clone now
                    </Link>
                )}{' '}
                {
                    <Link
                        className="btn btn-secondary btn-sm"
                        to={`/${props.node.name}/-/settings`}
                        data-tooltip="Repository settings"
                    >
                        <SettingsIcon className="icon-inline" /> Settings
                    </Link>
                }{' '}
            </div>
        </div>
    </li>
)

interface Props extends RouteComponentProps<{}>, ActivationProps {}

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
        args: { cloned: true, cloneInProgress: false, notCloned: false },
    },
    {
        label: 'Not cloned',
        id: 'not-cloned',
        tooltip: 'Show only repositories that have not been cloned yet',
        args: { cloned: false, cloneInProgress: false, notCloned: true },
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
export const SiteAdminRepositoriesPage: React.FunctionComponent<Props> = props => {
    useEffect(() => {
        eventLogger.logViewEvent('SiteAdminRepos')
    })

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
    })
    const repositoryUpdates = new Subject<void>()
    const nodeProps: Omit<RepositoryNodeProps, 'node'> = {
        onDidUpdate: repositoryUpdates.next.bind(repositoryUpdates),
        activation: props.activation,
        history: props.history,
    }
    const queryRepositories = useCallback(
        (args: FilteredConnectionQueryArgs): Observable<GQL.RepositoryConnection> =>
            fetchAllRepositoriesAndPollIfEmptyOrAnyCloning({ ...args }),
        []
    )
    const showRepositoriesAddedBanner = new URLSearchParams(props.location.search).has('repositoriesUpdated')

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
            <FilteredConnection<GQL.Repository, Omit<RepositoryNodeProps, 'node'>>
                className="list-group list-group-flush mt-3"
                noun="repository"
                pluralNoun="repositories"
                queryConnection={queryRepositories}
                nodeComponent={RepositoryNode}
                nodeComponentProps={nodeProps}
                updates={repositoryUpdates}
                filters={FILTERS}
                history={props.history}
                location={props.location}
            />
        </div>
    )
}
