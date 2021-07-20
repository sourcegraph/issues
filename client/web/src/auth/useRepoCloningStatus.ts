import { ApolloError, ReactiveVar } from '@apollo/client'
import { upperFirst, xor } from 'lodash'

import { useQuery, gql } from '@sourcegraph/shared/src/graphql/graphql'

import { Maybe, UserRepositoriesVariables } from '../graphql-operations'

import { MinSelectedRepo } from './useSelectedRepos'

interface UseRepoCloningStatusArguments {
    userId: string
    pollInterval: number
    selectedReposVar: ReactiveVar<MinSelectedRepo[] | undefined>
}

interface RepoLine {
    id: string
    title: string
    details: string
    progress: number
}

export interface RepoCloningStatus {
    repos: RepoLine[] | undefined
    loading: boolean
    isDoneCloning: boolean
    error: ApolloError | undefined
}

interface RepoFields {
    id: string
    name: string
    mirrorInfo: { cloned: boolean; cloneProgress: string; cloneInProgress: boolean; updatedAt: Maybe<string> }
}

interface CloneProgressResult {
    node: Maybe<{
        repositories: {
            nodes: RepoFields[]
        }
    }>
}

// temp object to store previous cloning progress percentage
let previousPercentage: { [key: string]: number } = {}

const USER_AFFILIATED_REPOS_MIRROR_INFO = gql`
    query UserRepositoriesMirrorInfo(
        $id: ID!
        $first: Int
        $query: String
        $cloned: Boolean
        $notCloned: Boolean
        $indexed: Boolean
        $notIndexed: Boolean
        $externalServiceID: ID
    ) {
        node(id: $id) {
            ... on User {
                repositories(
                    first: $first
                    query: $query
                    cloned: $cloned
                    notCloned: $notCloned
                    indexed: $indexed
                    notIndexed: $notIndexed
                    externalServiceID: $externalServiceID
                ) {
                    nodes {
                        id
                        name
                        mirrorInfo {
                            cloned
                            cloneInProgress
                            cloneProgress
                            updatedAt
                        }
                    }
                    totalCount
                }
            }
        }
    }
`

const getRepoNames = (repos: { name: string }[] | undefined): string[] | [] =>
    repos ? repos.map(({ name }) => name) : []

export const useRepoCloningStatus = ({
    userId,
    pollInterval = 5000,
    selectedReposVar,
}: UseRepoCloningStatusArguments): RepoCloningStatus => {
    let areAllRepoCloned = true

    const { called, data, loading, error, stopPolling } = useQuery<CloneProgressResult, UserRepositoriesVariables>(
        USER_AFFILIATED_REPOS_MIRROR_INFO,
        {
            variables: {
                id: userId,
                cloned: true,
                notCloned: true,
                indexed: true,
                notIndexed: true,
                first: 2000,
                query: null,
                externalServiceID: null,
            },
            pollInterval,
            fetchPolicy: 'no-cache',
        }
    )

    const selectedRepos = selectedReposVar()

    const repos = data?.node?.repositories.nodes

    if (!Array.isArray(repos)) {
        return {
            repos: undefined,
            isDoneCloning: false,
            loading,
            error,
        }
    }

    // check if we received cloning status for all selected repos
    const didReceiveAllRepoStatuses = xor(getRepoNames(selectedRepos), getRepoNames(repos)).length === 0

    const repoLines: RepoLine[] = repos.reduce((lines, { id, name, mirrorInfo }) => {
        const { details, progress, cloned } = parseMirrorInfo(id, mirrorInfo)

        if (!cloned) {
            areAllRepoCloned = false
        }

        lines.push({
            id,
            details,
            progress,
            title: ignoreExternalService(name),
        })

        return lines
    }, [] as RepoLine[])

    repoLines.sort((lineA, lineB) => lineB.progress - lineA.progress)

    const isDoneCloning = didReceiveAllRepoStatuses && areAllRepoCloned

    // stop polling and cleanup memory when all repos are done cloning
    if (called && isDoneCloning) {
        stopPolling()
        previousPercentage = {}
    }

    return { repos: repoLines, isDoneCloning, loading, error }
}

const parseMirrorInfo = (
    id: string,
    mirrorInfo: RepoFields['mirrorInfo']
): { progress: number; details: string; cloned: boolean } => {
    const { cloneProgress, cloned, cloneInProgress } = mirrorInfo

    // cloned
    if (cloned) {
        return {
            cloned,
            details: 'Successfully cloned',
            progress: 100,
        }
    }

    // not cloned and cloning is not in progress
    if (!cloneInProgress) {
        return {
            cloned,
            details: 'Not cloned yet',
            progress: 0,
        }
    }

    // remove extra spaces and upper fist character
    const normalizedDetails = normalizeDetails(cloneProgress)

    /**
     * for the details like:
     * 1. * [new branch] master -> master
     * 2. * [new ref] refs/pull/9/merge -> refs/pull/9/merge
     * use previously parsed progress value or 0
     */
    let percentage = 0
    if (normalizedDetails.startsWith('*')) {
        percentage = previousPercentage[id] || 0
    } else {
        percentage = findProgressPercentage(cloneProgress)
        previousPercentage[id] = percentage
    }

    return {
        cloned,
        details: normalizedDetails,
        progress: percentage,
    }
}

const findProgressPercentage = (progress: string = ''): number => {
    // fist 1-3 digits before the % sign
    const PERCENTAGE = /\d{1,3}(?=%)/
    const match = progress.match(PERCENTAGE)

    if (match) {
        // convert first string into numbers
        return +match[0]
    }

    return 0
}

const normalizeDetails = (string: string): string => upperFirst(string.replace(/\s\s+/g, ' '))
const ignoreExternalService = (fullRepoName: string): string => fullRepoName.slice(fullRepoName.indexOf('/') + 1)
