import React, { useCallback } from 'react'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import CloudOutlineIcon from 'mdi-react/CloudOutlineIcon'
import TickIcon from 'mdi-react/TickIcon'
import GithubIcon from 'mdi-react/GithubIcon'
import GitlabIcon from 'mdi-react/GitlabIcon'
import BitbucketIcon from 'mdi-react/BitbucketIcon'
import SourceRepositoryIcon from 'mdi-react/SourceRepositoryIcon'
import { RepoLink } from '../../../shared/src/components/RepoLink'
import ChevronRightIcon from 'mdi-react/ChevronRightIcon'
import {ExternalServiceKind} from "../graphql-operations";

interface RepositoryNodeProps {
    name: string
    mirrorInfo?: {
        cloneInProgress: boolean
        cloned: boolean
    }
    onClick?: () => void
    url: string
    serviceType: string
    isPrivate: boolean
    prefixComponent?: JSX.Element
}

interface StatusIconProps {
    mirrorInfo?: {
        cloneInProgress: boolean
        cloned: boolean
    }
}

const StatusIcon: React.FunctionComponent<StatusIconProps> = ({ mirrorInfo }) => {
    if (mirrorInfo === undefined) {
        return null
    }
    if (mirrorInfo.cloneInProgress) {
        return (
            <small data-tooltip="Clone in progress." className="mr-2 text-success">
                <LoadingSpinner className="icon-inline" />
            </small>
        )
    }
    if (!mirrorInfo.cloned) {
        return (
            <small
                className="mr-2 text-muted"
                data-tooltip="Visit the repository to clone it. See its mirroring settings for diagnostics."
            >
                <CloudOutlineIcon className="icon-inline" />
            </small>
        )
    }
    return (
        <small className="mr-2">
            <TickIcon className="icon-inline user-settings-repos_check-icon" />
        </small>
    )
}

interface CodeHostIconProps {
    hostType: string
}

const CodeHostIcon: React.FunctionComponent<CodeHostIconProps> = ({ hostType }) => {
    switch (hostType) {
        case ExternalServiceKind.GITHUB:
            return (
                <small className="mr-2">
                    <GithubIcon className="icon-inline user-settings-repos_github-icon" />
                </small>
            )
        case ExternalServiceKind.GITLAB:
            return (
                <small className="mr-2">
                    <GitlabIcon className="icon-inline user-settings-repos_gitlab-icon" />
                </small>
            )
        case ExternalServiceKind.BITBUCKETCLOUD:
            return (
                <small className="mr-2">
                    <BitbucketIcon className="icon-inline user-settings-repos_bitbucket-icon" />
                </small>
            )
        default:
            return (
                <small className="mr-2">
                    <SourceRepositoryIcon className="icon-inline" />
                </small>
            )
    }
}

export const RepositoryNode: React.FunctionComponent<RepositoryNodeProps> = ({
    name,
    mirrorInfo,
    url,
    onClick,
    serviceType,
    isPrivate,
    prefixComponent,
}) => {
    const handleOnClick = useCallback(
        (event: React.MouseEvent<HTMLAnchorElement, MouseEvent>): void => {
            if (onClick !== undefined) {
                event.preventDefault()
                onClick()
            }
        },
        [onClick]
    )
    return (
        <tr className="repository-node">
            <td>
                <a className="w-100 d-flex justify-content-between align-items-center" href={url} onClick={handleOnClick}>
                    <div className="d-flex align-items-center">
                        {prefixComponent && prefixComponent}
                        <StatusIcon mirrorInfo={mirrorInfo} />
                        <CodeHostIcon hostType={serviceType} />
                        <RepoLink
                            className="text-muted"
                            repoClassName="text-primary"
                            repoName={name}
                            to={url}
                            onClick={handleOnClick}
                        />
                    </div>
                    <div>
                        {isPrivate && <div className="badge badge-secondary text-muted">Private</div>}
                        <ChevronRightIcon className="icon-inline ml-2 caret-icon" />
                    </div>
                </a>
            </td>
        </tr>
    )
}

interface CheckboxRepositoryNodeProps {
    name: string
    mirrorInfo?: {
        cloneInProgress: boolean
        cloned: boolean
    }
    onClick: () => void
    checked: boolean
    serviceType: string
    isPrivate: boolean
}

export const CheckboxRepositoryNode: React.FunctionComponent<CheckboxRepositoryNodeProps> = ({
    name,
    mirrorInfo,
    onClick,
    checked,
    serviceType,
    isPrivate,
}) => {
    const handleOnClick = useCallback(
        (event: React.MouseEvent<HTMLAnchorElement, MouseEvent>): void => {
            if (onClick !== undefined) {
                event.preventDefault()
                onClick()
            }
        },
        [onClick]
    )
    return (
        <tr className="repository-node" key={name}>
            <td className="w-100 d-flex justify-content-between" onClick={onClick}>
                <div className="d-flex align-items-center">
                    <input className="mr-2" type="checkbox" onChange={onClick} checked={checked} />
                    <StatusIcon mirrorInfo={mirrorInfo} />
                    <CodeHostIcon hostType={serviceType} />
                    <RepoLink
                        className="text-muted"
                        repoClassName="font-weight-bolder"
                        repoName={name}
                        to=""
                        onClick={handleOnClick}
                    />
                </div>
                <div>{isPrivate && <div className="badge badge-secondary text-muted">Private</div>}</div>
            </td>
        </tr>
    )
}
