import { subYears, formatISO } from 'date-fns'
import * as H from 'history'
import BookOpenVariantIcon from 'mdi-react/BookOpenVariantIcon'
import FolderIcon from 'mdi-react/FolderIcon'
import HistoryIcon from 'mdi-react/HistoryIcon'
import SettingsIcon from 'mdi-react/SettingsIcon'
import SourceBranchIcon from 'mdi-react/SourceBranchIcon'
import SourceCommitIcon from 'mdi-react/SourceCommitIcon'
import SourceRepositoryIcon from 'mdi-react/SourceRepositoryIcon'
import TagIcon from 'mdi-react/TagIcon'
import UserIcon from 'mdi-react/UserIcon'
import React, { useState, useMemo, useCallback, useEffect, useContext } from 'react'
import { Link, Redirect } from 'react-router-dom'
import { Observable, EMPTY, from } from 'rxjs'
import { catchError, map, switchMap } from 'rxjs/operators'

import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { ActionItem } from '@sourcegraph/shared/src/actions/ActionItem'
import { ActionsContainer } from '@sourcegraph/shared/src/actions/ActionsContainer'
import { wrapRemoteObservable } from '@sourcegraph/shared/src/api/client/api/common'
import { FileDecorationsByPath } from '@sourcegraph/shared/src/api/extension/extensionHostApi'
import { ContributableMenu } from '@sourcegraph/shared/src/api/protocol'
import { ActivationProps } from '@sourcegraph/shared/src/components/activation/Activation'
import { displayRepoName } from '@sourcegraph/shared/src/components/RepoFileLink'
import { ExtensionsControllerProps } from '@sourcegraph/shared/src/extensions/controller'
import { gql, dataOrThrowErrors } from '@sourcegraph/shared/src/graphql/graphql'
import * as GQL from '@sourcegraph/shared/src/graphql/schema'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { VersionContextProps } from '@sourcegraph/shared/src/search/util'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { asError, ErrorLike, isErrorLike } from '@sourcegraph/shared/src/util/errors'
import { memoizeObservable } from '@sourcegraph/shared/src/util/memoizeObservable'
import { pluralize } from '@sourcegraph/shared/src/util/strings'
import { encodeURIPathComponent, toPrettyBlobURL, toURIWithPath } from '@sourcegraph/shared/src/util/url'
import { useObservable } from '@sourcegraph/shared/src/util/useObservable'
import { Container, PageHeader } from '@sourcegraph/wildcard'

import { getFileDecorations } from '../../backend/features'
import { queryGraphQL } from '../../backend/graphql'
import { BatchChangesProps } from '../../batches/batches'
import { RepoBatchChangesButton } from '../../batches/RepoBatchChangesButton'
import { ErrorAlert } from '../../components/alerts'
import { BreadcrumbSetters } from '../../components/Breadcrumbs'
import { FilteredConnection } from '../../components/FilteredConnection'
import { PageTitle } from '../../components/PageTitle'
import { GitCommitFields, Scalars, TreePageRepositoryFields } from '../../graphql-operations'
import { InsightsApiContext, InsightsViewGrid } from '../../insights'
import { Settings } from '../../schema/settings.schema'
import { PatternTypeProps, CaseSensitivityProps, SearchContextProps } from '../../search'
import { basename } from '../../util/path'
import { fetchTreeEntries } from '../backend'
import { GitCommitNode, GitCommitNodeProps } from '../commits/GitCommitNode'
import { gitCommitFragment } from '../commits/RepositoryCommitsPage'
import { FilePathBreadcrumbs } from '../FilePathBreadcrumbs'

import { TreeEntriesSection } from './TreeEntriesSection'

const fetchTreeCommits = memoizeObservable(
    (args: {
        repo: Scalars['ID']
        revspec: string
        first?: number
        filePath?: string
        after?: string
    }): Observable<GQL.IGitCommitConnection> =>
        queryGraphQL(
            gql`
                query TreeCommits($repo: ID!, $revspec: String!, $first: Int, $filePath: String, $after: String) {
                    node(id: $repo) {
                        __typename
                        ... on Repository {
                            commit(rev: $revspec) {
                                ancestors(first: $first, path: $filePath, after: $after) {
                                    nodes {
                                        ...GitCommitFields
                                    }
                                    pageInfo {
                                        hasNextPage
                                    }
                                }
                            }
                        }
                    }
                }
                ${gitCommitFragment}
            `,
            args
        ).pipe(
            map(dataOrThrowErrors),
            map(data => {
                if (!data.node) {
                    throw new Error('Repository not found')
                }
                if (data.node.__typename !== 'Repository') {
                    throw new Error('Node is not a Repository')
                }
                if (!data.node.commit) {
                    throw new Error('Commit not found')
                }
                return data.node.commit.ancestors
            })
        ),
    args => `${args.repo}:${args.revspec}:${String(args.first)}:${String(args.filePath)}:${String(args.after)}`
)

interface Props
    extends SettingsCascadeProps<Settings>,
        ExtensionsControllerProps,
        PlatformContextProps,
        ThemeProps,
        TelemetryProps,
        ActivationProps,
        PatternTypeProps,
        CaseSensitivityProps,
        VersionContextProps,
        BatchChangesProps,
        Pick<SearchContextProps, 'selectedSearchContextSpec'>,
        BreadcrumbSetters {
    repo: TreePageRepositoryFields
    /** The tree's path in TreePage. We call it filePath for consistency elsewhere. */
    filePath: string
    commitID: string
    revision: string
    location: H.Location
    history: H.History
    globbing: boolean
}

export const treePageRepositoryFragment = gql`
    fragment TreePageRepositoryFields on Repository {
        id
        name
        description
        viewerCanAdminister
    }
`

export const TreePage: React.FunctionComponent<Props> = ({
    repo,
    commitID,
    revision,
    filePath,
    patternType,
    caseSensitive,
    settingsCascade,
    useBreadcrumb,
    batchChangesEnabled,
    ...props
}) => {
    useEffect(() => {
        if (filePath === '') {
            props.telemetryService.logViewEvent('Repository')
        } else {
            props.telemetryService.logViewEvent('Tree')
        }
    }, [filePath, props.telemetryService])

    useBreadcrumb(
        useMemo(() => {
            if (!filePath) {
                return
            }
            return {
                key: 'treePath',
                className: 'flex-shrink-past-contents',
                element: (
                    <FilePathBreadcrumbs
                        key="path"
                        repoName={repo.name}
                        revision={revision}
                        filePath={filePath}
                        isDir={true}
                    />
                ),
            }
        }, [repo.name, revision, filePath])
    )

    const [showOlderCommits, setShowOlderCommits] = useState(false)

    const onShowOlderCommitsClicked = useCallback(
        (event: React.MouseEvent): void => {
            event.preventDefault()
            setShowOlderCommits(true)
        },
        [setShowOlderCommits]
    )

    const treeOrError = useObservable(
        useMemo(
            () =>
                fetchTreeEntries({
                    repoName: repo.name,
                    commitID,
                    revision,
                    filePath,
                    first: 2500,
                }).pipe(catchError((error): [ErrorLike] => [asError(error)])),
            [repo.name, commitID, revision, filePath]
        )
    )

    const fileDecorationsByPath =
        useObservable<FileDecorationsByPath>(
            useMemo(
                () =>
                    treeOrError && !isErrorLike(treeOrError)
                        ? getFileDecorations({
                              files: treeOrError.entries,
                              extensionsController: props.extensionsController,
                              repoName: repo.name,
                              commitID,
                              parentNodeUri: treeOrError.url,
                          })
                        : EMPTY,
                [treeOrError, repo.name, commitID, props.extensionsController]
            )
        ) ?? {}

    const showCodeInsights =
        !isErrorLike(settingsCascade.final) &&
        !!settingsCascade.final?.experimentalFeatures?.codeInsights &&
        settingsCascade.final['insights.displayLocation.directory'] !== false

    // Add DirectoryViewer
    const uri = toURIWithPath({ repoName: repo.name, commitID, filePath })
    useEffect(() => {
        if (!showCodeInsights) {
            return
        }

        const viewerIdPromise = props.extensionsController.extHostAPI
            .then(extensionHostAPI =>
                extensionHostAPI.addViewerIfNotExists({
                    type: 'DirectoryViewer',
                    isActive: true,
                    resource: uri,
                })
            )
            .catch(error => {
                console.error('Error adding viewer to extension host:', error)
                return null
            })

        return () => {
            Promise.all([props.extensionsController.extHostAPI, viewerIdPromise])
                .then(([extensionHostAPI, viewerId]) => {
                    if (viewerId) {
                        return extensionHostAPI.removeViewer(viewerId)
                    }
                    return
                })
                .catch(error => console.error('Error removing viewer from extension host:', error))
        }
    }, [uri, showCodeInsights, props.extensionsController])

    // Observe directory views
    const workspaceUri = useObservable(
        useMemo(
            () =>
                from(props.extensionsController.extHostAPI).pipe(
                    switchMap(extensionHostAPI => wrapRemoteObservable(extensionHostAPI.getWorkspaceRoots())),
                    map(workspaceRoots => workspaceRoots[0]?.uri)
                ),
            [props.extensionsController]
        )
    )

    // eslint-disable-next-line unicorn/prevent-abbreviations
    const enableAPIDocs =
        !isErrorLike(settingsCascade.final) && settingsCascade.final?.experimentalFeatures?.apiDocs !== false

    const { getCombinedViews } = useContext(InsightsApiContext)
    const views = useObservable(
        useMemo(
            () =>
                showCodeInsights && workspaceUri
                    ? getCombinedViews(() =>
                          from(props.extensionsController.extHostAPI).pipe(
                              switchMap(extensionHostAPI =>
                                  wrapRemoteObservable(
                                      extensionHostAPI.getDirectoryViews({
                                          viewer: {
                                              type: 'DirectoryViewer',
                                              directory: {
                                                  uri,
                                              },
                                          },
                                          workspace: {
                                              uri: workspaceUri,
                                          },
                                      })
                                  )
                              )
                          )
                      )
                    : EMPTY,
            [getCombinedViews, showCodeInsights, workspaceUri, uri, props.extensionsController]
        )
    )

    const getPageTitle = (): string => {
        const repoString = displayRepoName(repo.name)
        if (filePath) {
            return `${basename(filePath)} - ${repoString}`
        }
        return `${repoString}`
    }

    const queryCommits = useCallback(
        (args: { first?: number }): Observable<GQL.IGitCommitConnection> => {
            const after: string | undefined = showOlderCommits ? undefined : formatISO(subYears(Date.now(), 1))
            return fetchTreeCommits({
                ...args,
                repo: repo.id,
                revspec: revision || '',
                filePath,
                after,
            })
        },
        [filePath, repo.id, revision, showOlderCommits]
    )

    const emptyElement = showOlderCommits ? (
        <>No commits in this tree.</>
    ) : (
        <div className="test-tree-page-no-recent-commits">
            <p className="mb-2">No commits in this tree in the past year.</p>
            <button
                type="button"
                className="btn btn-secondary btn-sm test-tree-page-show-all-commits"
                onClick={onShowOlderCommitsClicked}
            >
                Show all commits
            </button>
        </div>
    )

    const TotalCountSummary: React.FunctionComponent<{ totalCount: number }> = ({ totalCount }) => (
        <div className="mt-2">
            {showOlderCommits ? (
                <>
                    {totalCount} total {pluralize('commit', totalCount)} in this tree.
                </>
            ) : (
                <>
                    <p className="mb-2">
                        {totalCount} {pluralize('commit', totalCount)} in this tree in the past year.
                    </p>
                    <button type="button" className="btn btn-secondary btn-sm" onClick={onShowOlderCommitsClicked}>
                        Show all commits
                    </button>
                </>
            )}
        </div>
    )

    return (
        <div className="tree-page">
            <Container className="tree-page__container">
                <PageTitle title={getPageTitle()} />
                {treeOrError === undefined ? (
                    <div>
                        <LoadingSpinner className="icon-inline tree-page__entries-loader" /> Loading files and
                        directories
                    </div>
                ) : isErrorLike(treeOrError) ? (
                    // If the tree is actually a blob, be helpful and redirect to the blob page.
                    // We don't have error names on GraphQL errors.
                    /not a directory/i.test(treeOrError.message) ? (
                        <Redirect to={toPrettyBlobURL({ repoName: repo.name, revision, commitID, filePath })} />
                    ) : (
                        <ErrorAlert error={treeOrError} />
                    )
                ) : (
                    <>
                        <header className="mb-3">
                            {treeOrError.isRoot ? (
                                <>
                                    <PageHeader
                                        path={[{ icon: SourceRepositoryIcon, text: displayRepoName(repo.name) }]}
                                        className="mb-3 test-tree-page-title"
                                    />
                                    {repo.description && <p>{repo.description}</p>}
                                    <div className="btn-group">
                                        {enableAPIDocs && (
                                            <Link
                                                className="btn btn-outline-secondary"
                                                to={`${treeOrError.url}/-/docs`}
                                            >
                                                <BookOpenVariantIcon className="icon-inline" /> API docs
                                            </Link>
                                        )}
                                        <Link className="btn btn-outline-secondary" to={`${treeOrError.url}/-/commits`}>
                                            <SourceCommitIcon className="icon-inline" /> Commits
                                        </Link>
                                        <Link
                                            className="btn btn-outline-secondary"
                                            to={`/${encodeURIPathComponent(repo.name)}/-/branches`}
                                        >
                                            <SourceBranchIcon className="icon-inline" /> Branches
                                        </Link>
                                        <Link
                                            className="btn btn-outline-secondary"
                                            to={`/${encodeURIPathComponent(repo.name)}/-/tags`}
                                        >
                                            <TagIcon className="icon-inline" /> Tags
                                        </Link>
                                        <Link
                                            className="btn btn-outline-secondary"
                                            to={
                                                revision
                                                    ? `/${encodeURIPathComponent(
                                                          repo.name
                                                      )}/-/compare/...${encodeURIComponent(revision)}`
                                                    : `/${encodeURIPathComponent(repo.name)}/-/compare`
                                            }
                                        >
                                            <HistoryIcon className="icon-inline" /> Compare
                                        </Link>
                                        <Link
                                            className="btn btn-outline-secondary"
                                            to={`/${encodeURIPathComponent(repo.name)}/-/stats/contributors`}
                                        >
                                            <UserIcon className="icon-inline" /> Contributors
                                        </Link>
                                        {batchChangesEnabled && (
                                            <RepoBatchChangesButton
                                                className="btn btn-outline-secondary"
                                                repoName={repo.name}
                                            />
                                        )}
                                        {repo.viewerCanAdminister && (
                                            <Link
                                                className="btn btn-outline-secondary"
                                                to={`/${encodeURIPathComponent(repo.name)}/-/settings`}
                                            >
                                                <SettingsIcon className="icon-inline" /> Settings
                                            </Link>
                                        )}
                                    </div>
                                </>
                            ) : (
                                <PageHeader
                                    path={[{ icon: FolderIcon, text: filePath }]}
                                    className="mb-3 test-tree-page-title"
                                />
                            )}
                        </header>
                        {views && (
                            <InsightsViewGrid
                                telemetryService={props.telemetryService}
                                className="tree-page__section mb-3"
                                views={views}
                            />
                        )}
                        <section className="tree-page__section test-tree-entries mb-3">
                            <h2>Files and directories</h2>
                            <TreeEntriesSection
                                parentPath={filePath}
                                entries={treeOrError.entries}
                                fileDecorationsByPath={fileDecorationsByPath}
                                isLightTheme={props.isLightTheme}
                            />
                        </section>
                        <ActionsContainer {...props} menu={ContributableMenu.DirectoryPage} empty={null}>
                            {items => (
                                <section className="tree-page__section">
                                    <h2>Actions</h2>
                                    {items.map(item => (
                                        <ActionItem
                                            {...props}
                                            key={item.action.id}
                                            {...item}
                                            className="btn btn-secondary mr-1 mb-1"
                                        />
                                    ))}
                                </section>
                            )}
                        </ActionsContainer>

                        <div className="tree-page__section">
                            <h2>Changes</h2>
                            <FilteredConnection<GitCommitFields, Pick<GitCommitNodeProps, 'className' | 'compact'>>
                                location={props.location}
                                className="mt-2 tree-page__section--commits"
                                listClassName="list-group list-group-flush"
                                noun="commit in this tree"
                                pluralNoun="commits in this tree"
                                queryConnection={queryCommits}
                                nodeComponent={GitCommitNode}
                                nodeComponentProps={{
                                    className: 'list-group-item',
                                    compact: true,
                                }}
                                updateOnChange={`${repo.name}:${revision}:${filePath}:${String(showOlderCommits)}`}
                                defaultFirst={7}
                                useURLQuery={false}
                                hideSearch={true}
                                emptyElement={emptyElement}
                                totalCountSummaryComponent={TotalCountSummary}
                            />
                        </div>
                    </>
                )}
            </Container>
        </div>
    )
}
