import { FetchResult } from '@apollo/client'
import classNames from 'classnames'
import { isEqual } from 'lodash'
import React, { useCallback, useEffect, useState, FunctionComponent, Dispatch, SetStateAction } from 'react'

import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { useRedesignToggle } from '@sourcegraph/shared/src/util/useRedesignToggle'
import { Container, PageSelector } from '@sourcegraph/wildcard'
import { useSteps } from '@sourcegraph/wildcard/src/components/Steps'

import { RepoSelectionMode } from '../../../auth/PostSignUpPage'
import { useAffiliatedRepos } from '../../../auth/useAffiliatedRepos'
import { useExternalServices } from '../../../auth/useExternalServices'
import { useSelectedRepos, selectedReposVar, MinSelectedRepo } from '../../../auth/useSelectedRepos'
import { AwayPrompt } from '../../../components/AwayPrompt'
import {
    ExternalServiceKind,
    Maybe,
    SiteAdminRepositoryFields,
    SetExternalServiceReposResult,
} from '../../../graphql-operations'
import { externalServiceUserModeFromTags } from '../cloud-ga'

import { CheckboxRepositoryNode } from './RepositoryNode'
export interface AffiliatedReposReference {
    submit: () => Promise<FetchResult<SetExternalServiceReposResult>[] | void>
}

interface authenticatedUser {
    id: string
    siteAdmin: boolean
    tags: string[]
}

interface Props extends TelemetryProps {
    authenticatedUser: authenticatedUser
    onRepoSelectionModeChange: Dispatch<SetStateAction<RepoSelectionMode>>
}

export interface Repo {
    name: string
    codeHost: Maybe<{ kind: ExternalServiceKind; id: string; displayName: string }>
    private: boolean
    mirrorInfo?: SiteAdminRepositoryFields['mirrorInfo']
}

interface GitHubConfig {
    repos?: string[]
    repositoryQuery?: string[]
    token: 'REDACTED'
    url: string
}

interface GitLabConfig {
    projectQuery?: string[]
    projects?: { name: string }[]
    token: 'REDACTED'
    url: string
}

const PER_PAGE = 20

// project queries that are used when user syncs all repos from a code host
const GITLAB_SYNC_ALL_PROJECT_QUERY = 'projects?membership=true&archived=no'
const GITHUB_SYNC_ALL_PROJECT_QUERY = 'affiliated'

// initial state constants
const emptyRepos: Repo[] = []
const initialRepoState = {
    repos: emptyRepos,
    loading: false,
    loaded: false,
}

const initialSelectionState = {
    repos: new Map<string, Repo>(),
    loaded: false,
    radio: '',
}

/**
 * A page to manage the repositories a user syncs from their connected code hosts.
 */
export const SelectAffiliatedRepos: FunctionComponent<Props> = ({
    authenticatedUser,
    onRepoSelectionModeChange,
    telemetryService,
}) => {
    useEffect(() => {
        telemetryService.logViewEvent('UserSettingsRepositories')
    }, [telemetryService])

    const { setComplete, currentIndex } = useSteps()
    const { externalServices } = useExternalServices(authenticatedUser.id)
    const { affiliatedRepos } = useAffiliatedRepos(authenticatedUser.id)
    const { selectedRepos } = useSelectedRepos(authenticatedUser.id)

    // if we should tweak UI messaging and copy
    const ALLOW_PRIVATE_CODE = externalServiceUserModeFromTags(authenticatedUser.tags) === 'all'

    // if 'sync all' radio button is enabled and users can sync all repos from code hosts
    const ALLOW_SYNC_ALL = authenticatedUser.tags.includes('AllowUserExternalServiceSyncAll')

    // set up state hooks
    const [isRedesignEnabled] = useRedesignToggle()

    const [currentPage, setPage] = useState(1)
    const [repoState, setRepoState] = useState(initialRepoState)
    const [onloadSelectedRepos, setOnloadSelectedRepos] = useState<string[]>([])
    const [selectionState, setSelectionState] = useState(initialSelectionState)
    const [didSelectionChange, setDidSelectionChange] = useState(false)
    const [query, setQuery] = useState('')
    const [codeHostFilter, setCodeHostFilter] = useState('')
    const [filteredRepos, setFilteredRepos] = useState<Repo[]>([])

    const getRepoServiceAndName = (repo: Repo): string => `${repo.codeHost?.kind || 'unknown'}/${repo.name}`

    useEffect(() => {
        if (externalServices && affiliatedRepos) {
            const codeHostsHaveSyncAllQuery = []

            for (const host of externalServices) {
                if (host.lastSyncError || host.warning) {
                    continue
                }

                const cfg = JSON.parse(host.config) as GitHubConfig | GitLabConfig
                switch (host.kind) {
                    case ExternalServiceKind.GITLAB: {
                        const gitLabCfg = cfg as GitLabConfig

                        if (Array.isArray(gitLabCfg.projectQuery)) {
                            codeHostsHaveSyncAllQuery.push(
                                gitLabCfg.projectQuery.includes(GITLAB_SYNC_ALL_PROJECT_QUERY)
                            )
                        }

                        break
                    }

                    case ExternalServiceKind.GITHUB: {
                        const gitHubCfg = cfg as GitHubConfig

                        if (Array.isArray(gitHubCfg.repositoryQuery)) {
                            codeHostsHaveSyncAllQuery.push(
                                gitHubCfg.repositoryQuery.includes(GITHUB_SYNC_ALL_PROJECT_QUERY)
                            )
                        }

                        break
                    }
                }
            }

            const selectedAffiliatedRepos = new Map<string, Repo>()

            const cachedSelectedRepos = selectedReposVar()
            const userSelectedRepos = cachedSelectedRepos || selectedRepos || []

            const affiliatedReposWithMirrorInfo = affiliatedRepos.map(affiliatedRepo => {
                let foundInSelected: SiteAdminRepositoryFields | MinSelectedRepo | null = null
                for (const selectedRepo of userSelectedRepos) {
                    const {
                        name,
                        externalRepository: { serviceType: selectedRepoServiceType },
                    } = selectedRepo
                    const selectedRepoName = name.slice(name.indexOf('/') + 1)

                    if (
                        selectedRepoName === affiliatedRepo.name &&
                        selectedRepoServiceType === affiliatedRepo.codeHost?.kind.toLocaleLowerCase()
                    ) {
                        foundInSelected = selectedRepo
                        break
                    }
                }

                if (foundInSelected) {
                    // save off only selected repos
                    selectedAffiliatedRepos.set(getRepoServiceAndName(affiliatedRepo), affiliatedRepo)
                }

                return affiliatedRepo
            })

            // sort affiliated repos with already selected repos at the top
            affiliatedReposWithMirrorInfo.sort((repoA, repoB): number => {
                const isRepoASelected = selectedAffiliatedRepos.has(getRepoServiceAndName(repoA))
                const isRepoBSelected = selectedAffiliatedRepos.has(getRepoServiceAndName(repoB))

                if (!isRepoASelected && isRepoBSelected) {
                    return 1
                }

                if (isRepoASelected && !isRepoBSelected) {
                    return -1
                }

                return 0
            })

            // safe off initial selection state
            setOnloadSelectedRepos(previousValue => [...previousValue, ...selectedAffiliatedRepos.keys()])

            /**
             * 1. if every code host has a project query to sync all repos or the
             * number of affiliated repos equals to the number of selected repos -
             * set radio to 'all'
             * 2. if only some repos were selected - set radio to 'selected'
             * 3. if no repos selected - empty state
             */

            const radioSelectOption =
                ALLOW_SYNC_ALL &&
                ((externalServices.length === codeHostsHaveSyncAllQuery.length &&
                    codeHostsHaveSyncAllQuery.every(Boolean)) ||
                    affiliatedReposWithMirrorInfo.length === selectedAffiliatedRepos.size)
                    ? 'all'
                    : selectedAffiliatedRepos.size > 0
                    ? 'selected'
                    : ''

            onRepoSelectionModeChange(radioSelectOption as RepoSelectionMode)

            // set sorted repos and mark as loaded
            setRepoState(previousRepoState => ({
                ...previousRepoState,
                repos: affiliatedReposWithMirrorInfo,
                loaded: true,
            }))

            setSelectionState({
                repos: selectedAffiliatedRepos,
                radio: radioSelectOption,
                loaded: true,
            })
        }
    }, [externalServices, affiliatedRepos, selectedRepos, ALLOW_SYNC_ALL, onRepoSelectionModeChange])

    // select repos by code host and query
    useEffect(() => {
        // filter our set of repos based on query & code host selection
        const filtered: Repo[] = []

        for (const repo of repoState.repos) {
            // filtering by code hosts
            if (codeHostFilter !== '' && repo.codeHost?.id !== codeHostFilter) {
                continue
            }

            const queryLoweCase = query.toLowerCase()
            const nameLowerCase = repo.name.toLowerCase()
            if (!nameLowerCase.includes(queryLoweCase)) {
                continue
            }

            filtered.push(repo)
        }

        // set new filtered pages and reset the pagination
        setFilteredRepos(filtered)
        setPage(1)
    }, [repoState.repos, codeHostFilter, query])

    // track selection changes
    useEffect(() => {
        const affiliatedRepos = selectionState.repos.keys()

        const currentlySelectedRepos = [...affiliatedRepos]

        const didChange = !isEqual(currentlySelectedRepos.sort(), onloadSelectedRepos.sort())

        // set current step as complete if user had already selected repos or made changes
        if (didChange || onloadSelectedRepos.length !== 0) {
            setComplete(currentIndex, true)
        } else {
            setComplete(currentIndex, false)
        }

        // save off last selected repos
        const selection = [...selectionState.repos.values()].reduce((accumulator, repo) => {
            const serviceType = repo.codeHost?.kind.toLowerCase()
            const serviceName = serviceType ? `${serviceType}.com` : 'unknown'

            accumulator.push({
                name: `${serviceName}/${repo.name}`,
                externalRepository: { serviceType: serviceType || 'unknown', id: repo.codeHost?.id },
            })
            return accumulator
        }, [] as MinSelectedRepo[])

        // safe off repo selection to apollo
        selectedReposVar(selection)

        setDidSelectionChange(didChange)
    }, [currentIndex, onloadSelectedRepos, selectionState.repos, setComplete])

    const handleRadioSelect = (changeEvent: React.ChangeEvent<HTMLInputElement>): void => {
        setSelectionState({
            repos: selectionState.repos,
            radio: changeEvent.currentTarget.value,
            loaded: selectionState.loaded,
        })

        onRepoSelectionModeChange(changeEvent.currentTarget.value as RepoSelectionMode)
    }

    const hasCodeHosts = Array.isArray(externalServices) && externalServices.length > 0

    const modeSelect: JSX.Element = (
        <>
            <label className="d-flex flex-row align-items-baseline">
                <input
                    type="radio"
                    value="all"
                    disabled={!ALLOW_SYNC_ALL}
                    checked={selectionState.radio === 'all'}
                    onChange={handleRadioSelect}
                />
                <div className="d-flex flex-column ml-2">
                    <p
                        className={classNames('mb-0', {
                            'user-settings-repos__text': ALLOW_SYNC_ALL,
                            'user-settings-repos__text-disabled': !ALLOW_SYNC_ALL,
                        })}
                    >
                        Sync all repositories {!ALLOW_SYNC_ALL && '(coming soon)'}
                    </p>
                    <p
                        className={classNames({
                            'user-settings-repos__text': ALLOW_SYNC_ALL,
                            'user-settings-repos__text-disabled': !ALLOW_SYNC_ALL,
                        })}
                    >
                        Will sync all current and future public and private repositories
                    </p>
                </div>
            </label>
            <label className="d-flex flex-row align-items-baseline mb-0">
                <input
                    type="radio"
                    value="selected"
                    checked={selectionState.radio === 'selected'}
                    onChange={handleRadioSelect}
                />
                <div className="d-flex flex-column ml-2">
                    <p
                        className={classNames({
                            'user-settings-repos__text-disabled': false,
                            'mb-0': true,
                        })}
                    >
                        Sync selected {!ALLOW_PRIVATE_CODE && 'public'} repositories
                    </p>
                </div>
            </label>
        </>
    )

    const filterControls: JSX.Element = (
        <div className="w-100 d-inline-flex justify-content-between flex-row mt-3">
            <div className="d-inline-flex flex-row mr-3 align-items-baseline">
                <p className="text-xl-center text-nowrap mr-2">Code Host:</p>
                <select
                    className="form-control"
                    name="code-host"
                    aria-label="select code host type"
                    onChange={event => setCodeHostFilter(event.target.value)}
                >
                    <option key="any" value="" label="Any" />
                    {externalServices?.map(value => (
                        <option key={value.id} value={value.id} label={value.displayName} />
                    ))}
                </select>
            </div>
            <input
                className="form-control user-settings-repos__filter-input"
                type="search"
                placeholder="Search..."
                name="query"
                autoComplete="off"
                autoCorrect="off"
                autoCapitalize="off"
                spellCheck={false}
                onChange={event => {
                    setQuery(event.target.value)
                }}
            />
        </div>
    )

    const onRepoClicked = useCallback(
        (repo: Repo) => (): void => {
            const clickedRepo = getRepoServiceAndName(repo)
            const newMap = new Map(selectionState.repos)
            if (newMap.has(clickedRepo)) {
                newMap.delete(clickedRepo)
            } else {
                newMap.set(clickedRepo, repo)
            }
            setSelectionState({
                repos: newMap,
                radio: selectionState.radio,
                loaded: selectionState.loaded,
            })
        },
        [selectionState, setSelectionState]
    )

    const selectAll = (): void => {
        const newMap = new Map<string, Repo>()
        // if not all repos are selected, we should select all, otherwise empty the selection

        if (selectionState.repos.size !== filteredRepos.length) {
            for (const repo of filteredRepos) {
                newMap.set(getRepoServiceAndName(repo), repo)
            }
        }
        setSelectionState({
            repos: newMap,
            loaded: selectionState.loaded,
            radio: selectionState.radio,
        })
    }

    const rows: JSX.Element = (
        <tbody>
            <tr className="align-items-baseline d-flex" key="header">
                <td className="user-settings-repos__repositorynode p-2 w-100 d-flex align-items-center border-top-0 border-bottom">
                    <input
                        id="select-all-repos"
                        className="mr-3"
                        type="checkbox"
                        checked={selectionState.repos.size !== 0 && selectionState.repos.size === filteredRepos.length}
                        onChange={selectAll}
                    />
                    <label
                        htmlFor="select-all-repos"
                        className={classNames({
                            'text-muted': selectionState.repos.size === 0,
                            'text-body': selectionState.repos.size !== 0,
                            'mb-0': true,
                        })}
                    >
                        {(selectionState.repos.size > 0 && (
                            <small>{`${selectionState.repos.size} ${
                                selectionState.repos.size === 1 ? 'repository' : 'repositories'
                            } selected`}</small>
                        )) || <small>Select all</small>}
                    </label>
                </td>
            </tr>
            {filteredRepos.map((repo, index) => {
                if (index < (currentPage - 1) * PER_PAGE || index >= currentPage * PER_PAGE) {
                    return
                }

                const serviceAndRepoName = getRepoServiceAndName(repo)

                return (
                    <CheckboxRepositoryNode
                        name={repo.name}
                        key={serviceAndRepoName}
                        onClick={onRepoClicked(repo)}
                        checked={selectionState.repos.has(serviceAndRepoName)}
                        serviceType={repo.codeHost?.kind || 'unknown'}
                        isPrivate={repo.private}
                    />
                )
            })}
        </tbody>
    )

    const modeSelectShimmer: JSX.Element = (
        <div className="container">
            <div className="mt-2 row">
                <div className="user-settings-repos__shimmer-circle mr-2" />
                <div className="user-settings-repos__shimmer mb-1 p-2 border-top-0 col-sm-2" />
            </div>
            <div className="mt-1 ml-2 row">
                <div className="user-settings-repos__shimmer mb-3 p-2 ml-1 border-top-0 col-sm-6" />
            </div>
            <div className="mt-2 row">
                <div className="user-settings-repos__shimmer-circle mr-2" />
                <div className="user-settings-repos__shimmer p-2 mb-1 border-top-0 col-sm-3" />
            </div>
        </div>
    )

    return (
        <div className="user-settings-repos mb-0">
            <Container>
                <ul className="list-group">
                    <li className="list-group-item user-settings-repos__container" key="from-code-hosts">
                        <div className={classNames(!isRedesignEnabled && 'p-4')}>
                            {!affiliatedRepos && modeSelectShimmer}

                            {/* display type of repo sync radio buttons */}
                            {hasCodeHosts && selectionState.loaded && modeSelect}

                            {
                                // if we're in 'selected' mode, show a list of all the repos on the code hosts to select from
                                hasCodeHosts && selectionState.radio === 'selected' && (
                                    <div className="ml-4">
                                        {filterControls}
                                        <table role="grid" className="table">
                                            {
                                                // if the repos are loaded display the rows of repos
                                                repoState.loaded && rows
                                            }
                                        </table>
                                        {filteredRepos.length > 0 && (
                                            <PageSelector
                                                currentPage={currentPage}
                                                onPageChange={setPage}
                                                totalPages={Math.ceil(filteredRepos.length / PER_PAGE)}
                                                className="pt-4"
                                            />
                                        )}
                                    </div>
                                )
                            }
                        </div>
                    </li>
                </ul>
            </Container>
            <AwayPrompt
                header="Discard unsaved changes?"
                message="Currently synced repositories will be unchanged"
                button_ok_text="Discard"
                when={didSelectionChange}
            />
        </div>
    )
}
