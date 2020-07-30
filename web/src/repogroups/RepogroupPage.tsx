import React, { useEffect, useMemo } from 'react'
import * as H from 'history'
import { PageTitle } from '../components/PageTitle'
import { KeyboardShortcutsProps } from '../keyboardShortcuts/keyboardShortcuts'
import { Link } from '../../../shared/src/components/Link'
import { SettingsCascadeProps, Settings, isSettingsValid } from '../../../shared/src/settings/settings'
import { ThemeProps } from '../../../shared/src/theme'
import { ThemePreferenceProps } from '../theme'
import { ActivationProps } from '../../../shared/src/components/activation/Activation'
import {
    PatternTypeProps,
    CaseSensitivityProps,
    InteractiveSearchProps,
    SmartSearchFieldProps,
    CopyQueryButtonProps,
} from '../search'
import { EventLoggerProps, eventLogger } from '../tracking/eventLogger'
import { ExtensionsControllerProps } from '../../../shared/src/extensions/controller'
import { PlatformContextProps } from '../../../shared/src/platform/context'
import { VersionContextProps } from '../../../shared/src/search/util'
import { VersionContext } from '../schema/site.schema'
import { submitSearch } from '../search/helpers'
import * as GQL from '../../../shared/src/graphql/schema'
import SourceRepositoryMultipleIcon from 'mdi-react/SourceRepositoryMultipleIcon'
import GithubIcon from 'mdi-react/GithubIcon'
import GitlabIcon from 'mdi-react/GitlabIcon'
import BitbucketIcon from 'mdi-react/BitbucketIcon'
import { RepogroupMetadata } from './types'
import { SearchPageInput } from '../search/input/SearchPageInput'
import { displayRepoName } from '../../../shared/src/components/RepoFileLink'
import { PrivateCodeCta } from '../search/input/PrivateCodeCta'
import { OptionalAuthProps } from '../auth'

export interface RepogroupPageProps
    extends SettingsCascadeProps<Settings>,
        ThemeProps,
        ThemePreferenceProps,
        ActivationProps,
        PatternTypeProps,
        CaseSensitivityProps,
        KeyboardShortcutsProps,
        EventLoggerProps,
        ExtensionsControllerProps<'executeCommand' | 'services'>,
        PlatformContextProps<'forceUpdateTooltip' | 'settings'>,
        InteractiveSearchProps,
        SmartSearchFieldProps,
        CopyQueryButtonProps,
        VersionContextProps,
        OptionalAuthProps {
    location: H.Location
    history: H.History
    isSourcegraphDotCom: boolean
    setVersionContext: (versionContext: string | undefined) => void
    availableVersionContexts: VersionContext[] | undefined

    // For NavLinks
    authRequired?: boolean
    showCampaigns: boolean

    /** Controls focusing the query input on the page. Query inputs are autofocused by default. */
    autoFocus?: boolean

    // Repogroup page metadata
    repogroupMetadata: RepogroupMetadata

    /** Whether globbing is enabled for filters. */
    globbing: boolean
}

export const RepogroupPage: React.FunctionComponent<RepogroupPageProps> = (props: RepogroupPageProps) => {
    useEffect(() => eventLogger.logViewEvent(`Repogroup:${props.repogroupMetadata.name}`))

    const repogroupQuery = `repogroup:${props.repogroupMetadata.name}`

    // Get repogroups from settings.
    const repogroups: { [name: string]: string[] } | undefined = useMemo(
        () =>
            isSettingsValid<Settings>(props.settingsCascade) && props.settingsCascade.final['search.repositoryGroups'],
        [props.settingsCascade]
    )

    // Find the repositories for this specific repogroup.
    const repogroupRepoList = repogroups?.[props.repogroupMetadata.name]

    const onSubmitExample = (query: string, patternType: GQL.SearchPatternType) => (
        event?: React.MouseEvent<HTMLButtonElement>
    ): void => {
        eventLogger.log('RepositoryGroupSuggestionClicked')
        // eslint-disable-next-line no-unused-expressions
        event?.preventDefault()
        submitSearch({ ...props, query, patternType, source: 'repogroupPage' })
    }

    return (
        <div className="repogroup-page">
            <PageTitle title={props.repogroupMetadata.title} />
            <RepogroupPageLogo
                className="repogroup-page__logo"
                icon={props.repogroupMetadata.homepageIcon}
                text={props.repogroupMetadata.title}
            />
            <div className="repogroup-page__subheading">
                <span className="text-monospace">
                    <span className="repogroup-page__keyword-text">repogroup:</span>
                    {props.repogroupMetadata.name}
                </span>
            </div>
            <div className="repogroup-page__container">
                <SearchPageInput
                    {...props}
                    queryPrefix={repogroupQuery}
                    source="repogroupPage"
                    interactiveModeHomepageMode={true}
                />
            </div>
            <div className="row">
                <div className="repogroup-page__column col-xs-12 col-lg-7">
                    <p className="repogroup-page__content-description h5 font-weight-normal mb-4">
                        {props.repogroupMetadata.description}
                    </p>

                    <h2>Search examples</h2>
                    {props.repogroupMetadata.examples.map(example => (
                        <div className="mt-3" key={example.title}>
                            <h3 className="mb-3">{example.title}</h3>
                            <p>{example.description}</p>
                            <div className="d-flex mb-4">
                                <div className="repogroup-page__example-bar form-control text-monospace ">
                                    <span className="repogroup-page__keyword-text">repogroup:</span>
                                    {props.repogroupMetadata.name} {example.exampleQuery}
                                </div>
                                <div className="d-flex">
                                    <button
                                        className="repogroup-page__example-search-button btn btn-primary search-button__btn test-search-button btn-secondary"
                                        type="button"
                                        aria-label="Search"
                                        onClick={onSubmitExample(
                                            `${repogroupQuery} ${example.rawQuery}`,
                                            example.patternType
                                        )}
                                    >
                                        Search
                                    </button>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
                <div className="repogroup-page__column col-xs-12 col-lg-5">
                    <div className="order-1-lg order-2-xs">
                        <PrivateCodeCta />
                    </div>
                    <div className="order-2-lg order-1-xs">
                        <div className="repogroup-page__repo-card card">
                            <h2 className="font-weight-normal">
                                <SourceRepositoryMultipleIcon className="icon-inline mr-2" />
                                Repositories
                            </h2>
                            <p>
                                Using the syntax{' '}
                                <span className="text-monospace">
                                    <span className="repogroup-page__keyword-text">repogroup:</span>
                                    {props.repogroupMetadata.name}
                                </span>{' '}
                                in a query will search these repositories:
                            </p>
                            <div className="repogroup-page__repo-list row">
                                <div className="col-lg-6">
                                    {repogroupRepoList?.slice(0, Math.ceil(repogroupRepoList.length / 2)).map(repo => (
                                        <RepoLink key={repo} repo={repo} />
                                    ))}
                                </div>
                                <div className="col-lg-6">
                                    {repogroupRepoList
                                        ?.slice(Math.ceil(repogroupRepoList.length / 2), repogroupRepoList.length)
                                        .map(repo => (
                                            <RepoLink key={repo} repo={repo} />
                                        ))}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

const RepoLinkClicked = (repoName: string) => (): void =>
    eventLogger.log('RepogroupPageRepoLinkClicked', { repo_name: repoName })

const RepoLink: React.FunctionComponent<{ repo: string }> = ({ repo }) => (
    <li className="repogroup-page__repo-item list-unstyled mb-3" key={repo}>
        {repo.startsWith('github.com') && (
            <>
                <a href={`https://${repo}`} target="_blank" rel="noopener noreferrer" onClick={RepoLinkClicked(repo)}>
                    <GithubIcon className="icon-inline repogroup-page__repo-list-icon" />
                </a>
                <Link to={`/${repo}`} className="text-monospace repogroup-page__web-link">
                    {displayRepoName(repo)}
                </Link>
            </>
        )}
        {repo.startsWith('gitlab.com') && (
            <>
                <a href={`https://${repo}`} target="_blank" rel="noopener noreferrer" onClick={RepoLinkClicked(repo)}>
                    <GitlabIcon className="icon-inline repogroup-page__repo-list-icon" />
                </a>
                <Link to={`/${repo}`} className="text-monospace repogroup-page__web-link">
                    {displayRepoName(repo)}
                </Link>
            </>
        )}
        {repo.startsWith('bitbucket.com') && (
            <>
                <a href={`https://${repo}`} target="_blank" rel="noopener noreferrer" onClick={RepoLinkClicked(repo)}>
                    <BitbucketIcon className="icon-inline repogroup-page__repo-list-icon" />
                </a>
                <Link to={`/${repo}`} className="text-monospace repogroup-page__web-link">
                    {displayRepoName(repo)}
                </Link>
            </>
        )}
    </li>
)

interface RepogroupPageLogoProps extends Exclude<React.ImgHTMLAttributes<HTMLImageElement>, 'src'> {
    icon: string
    text: string
}

/**
 * The repogroup logo image.
 */
const RepogroupPageLogo: React.FunctionComponent<RepogroupPageLogoProps> = props => (
    <div className="repogroup-page__logo-container d-flex align-items-center">
        <img {...props} src={props.icon} />
        <span className="h3 font-weight-normal mb-0 ml-1">{props.text}</span>
    </div>
)
