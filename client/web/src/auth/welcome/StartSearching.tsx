import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { useSteps } from '@sourcegraph/wildcard/src/components/Steps'
import {
    Terminal,
    TerminalTitle,
    TerminalLine,
    TerminalDetails,
    TerminalProgress,
} from '@sourcegraph/wildcard/src/components/Terminal'

import { UserAreaUserFields } from '../../graphql-operations'
import { LogoAscii } from '../LogoAscii'
import { RepoSelectionMode } from '../PostSignUpPage'
import { useExternalServices } from '../useExternalServices'
import { useRepoCloningStatus } from '../useRepoCloningStatus'
import { selectedReposVar, useSaveSelectedRepos } from '../useSelectedRepos'

interface StartSearching {
    user: UserAreaUserFields
    repoSelectionMode: RepoSelectionMode
}

export const useShowAlert = (isDoneCloning: boolean): { showAlert: boolean } => {
    const [showAlert, setShowAlert] = useState(false)

    useEffect(() => {
        const timer = setTimeout(() => setShowAlert(true), 10000)

        if (isDoneCloning) {
            clearTimeout(timer)
            setShowAlert(false)
        }

        return () => {
            clearTimeout(timer)
            setShowAlert(false)
        }
    }, [isDoneCloning])

    return { showAlert }
}

export const StartSearching: React.FunctionComponent<StartSearching> = ({ user, repoSelectionMode }) => {
    const { externalServices } = useExternalServices(user.id)
    const saveSelectedRepos = useSaveSelectedRepos()

    const { repos: cloningStatusLines, loading: cloningStatusLoading, isDoneCloning } = useRepoCloningStatus({
        userId: user.id,
        pollInterval: 2000,
        selectedReposVar,
    })

    useEffect(() => {
        const selectedRepos = selectedReposVar()

        if (externalServices && selectedRepos) {
            const codeHostRepoPromises = []

            for (const host of externalServices) {
                const repos: string[] = []
                for (const repo of selectedRepos) {
                    if (repo.externalRepository.id !== host.id) {
                        continue
                    }

                    const nameWithoutService = repo.name.slice(repo.name.indexOf('/') + 1)
                    repos.push(nameWithoutService)
                }

                codeHostRepoPromises.push(
                    saveSelectedRepos({
                        variables: {
                            id: host.id,
                            allRepos: repoSelectionMode === 'all',
                            repos: (repoSelectionMode === 'selected' && repos) || null,
                        },
                    })
                )
            }
        }
    }, [externalServices, saveSelectedRepos, repoSelectionMode])

    const { showAlert } = useShowAlert(isDoneCloning)
    const { currentIndex, setComplete } = useSteps()

    useEffect(() => {
        if (showAlert || isDoneCloning) {
            setComplete(currentIndex, true)
        }
    }, [currentIndex, setComplete, showAlert, isDoneCloning])

    return (
        <>
            <h3>Start searching...</h3>
            <p className="text-muted">
                We’re cloning your repos to Sourcegraph. In just a few moments, you can make your first search!
            </p>
            <div className="border overflow-hidden rounded">
                <header>
                    <h3 className="m-0 pl-4 py-3">Activity log</h3>
                </header>
                <Terminal>
                    {!isDoneCloning && (
                        <TerminalLine>
                            <code>Cloning Repositories...</code>
                        </TerminalLine>
                    )}
                    {cloningStatusLoading && (
                        <TerminalLine>
                            <TerminalTitle>Loading...</TerminalTitle>
                        </TerminalLine>
                    )}
                    {!cloningStatusLoading &&
                        !isDoneCloning &&
                        cloningStatusLines?.map(({ id, title, details, progress }) => (
                            <React.Fragment key={id}>
                                <TerminalLine>
                                    <TerminalTitle>{title}</TerminalTitle>
                                </TerminalLine>
                                <TerminalLine>
                                    <TerminalDetails>{details}</TerminalDetails>
                                </TerminalLine>
                                <TerminalLine>
                                    <TerminalProgress character="#" progress={progress} />
                                </TerminalLine>
                            </React.Fragment>
                        ))}
                    {isDoneCloning && (
                        <>
                            <TerminalLine>
                                <TerminalTitle>Done!</TerminalTitle>
                            </TerminalLine>
                            <TerminalLine>
                                <LogoAscii />
                            </TerminalLine>
                        </>
                    )}
                </Terminal>
            </div>
            {showAlert && (
                <div className="alert alert-warning mt-4">
                    Cloning your repositories is taking a long time. You can wait for cloning to finish, or{' '}
                    <Link to="/search">continue to Sourcegraph now</Link> while cloning continues in the background.
                    Note that you can only search repos that have finished cloning. Check status at any time in Settings
                    → Your repositories.
                </div>
            )}
        </>
    )
}
