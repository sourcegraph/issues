import * as H from 'history'
import * as React from 'react'
import { Link } from 'react-router-dom'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { CircleChevronLeftIcon } from '../../../shared/src/components/icons'
import { TabsWithLocalStorageViewStatePersistence } from '../../../shared/src/components/Tabs'
import { gql, dataOrThrowErrors } from '../../../shared/src/graphql/graphql'
import * as GQL from '../../../shared/src/graphql/schema'
import { memoizeObservable } from '../../../shared/src/util/memoizeObservable'
import { queryGraphQL } from '../backend/graphql'
import { FilteredConnection, FilteredConnectionQueryArgs } from '../components/FilteredConnection'
import { eventLogger } from '../tracking/eventLogger'
import { replaceRevisionInURL } from '../util/url'
import { GitReferenceNode, queryGitReferences } from './GitReference'
import { RevisionSpec } from '../../../shared/src/util/url'
import { RepositoryGitCommitResult } from '../graphql-operations'

const fetchRepositoryCommits = memoizeObservable(
    (
        args: RevisionSpec & { repo: GQL.Scalars['ID']; first?: number; query?: string }
    ): Observable<GQL.GitCommitConnection> =>
        queryGraphQL<RepositoryGitCommitResult>(
            gql`
                query RepositoryGitCommit($repo: ID!, $first: Int, $revision: String!, $query: String) {
                    node(id: $repo) {
                        __typename
                        ... on Repository {
                            commit(rev: $revision) {
                                ancestors(first: $first, query: $query) {
                                    nodes {
                                        id
                                        oid
                                        abbreviatedOID
                                        author {
                                            person {
                                                name
                                                avatarURL
                                            }
                                            date
                                        }
                                        subject
                                    }
                                    pageInfo {
                                        hasNextPage
                                    }
                                }
                            }
                        }
                    }
                }
            `,
            args
        ).pipe(
            map(dataOrThrowErrors),
            map(({ node }) => {
                if (!node) {
                    throw new Error(`Repository ${args.repo} not found`)
                }
                if (node.__typename !== 'Repository') {
                    throw new Error(`Node is a ${node.__typename}, not a Repository`)
                }
                if (!node.commit?.ancestors) {
                    throw new Error(`Cannot load ancestors for repository ${args.repo}`)
                }
                return node.commit.ancestors
            })
        ),
    args => JSON.stringify(args)
)

interface GitRefPopoverNodeProps {
    node: GQL.GitRef

    defaultBranch: string
    currentRevision: string | undefined

    location: H.Location
}

const GitReferencePopoverNode: React.FunctionComponent<GitRefPopoverNodeProps> = ({
    node,
    defaultBranch,
    currentRevision,
    location,
}) => {
    let isCurrent: boolean
    if (currentRevision) {
        isCurrent = node.name === currentRevision || node.abbrevName === currentRevision
    } else {
        isCurrent = node.name === `refs/heads/${defaultBranch}`
    }
    return (
        <GitReferenceNode
            node={node}
            url={replaceRevisionInURL(location.pathname + location.search + location.hash, node.abbrevName)}
            ancestorIsLink={false}
        >
            {isCurrent && (
                <CircleChevronLeftIcon
                    className="icon-inline connection-popover__node-link-icon"
                    data-tooltip="Current"
                />
            )}
        </GitReferenceNode>
    )
}

interface GitCommitNodeProps {
    node: GQL.GitCommit

    currentCommitID: string | undefined

    location: H.Location
}

const GitCommitNode: React.FunctionComponent<GitCommitNodeProps> = ({ node, currentCommitID, location }) => {
    const isCurrent = currentCommitID === node.oid
    return (
        <li key={node.oid} className="connection-popover__node revisions-popover-git-commit-node">
            <Link
                to={replaceRevisionInURL(location.pathname + location.search + location.hash, node.oid)}
                className={`connection-popover__node-link ${
                    isCurrent ? 'connection-popover__node-link--active' : ''
                } revisions-popover-git-commit-node__link`}
            >
                <code className="revisions-popover-git-commit-node__oid" title={node.oid}>
                    {node.abbreviatedOID}
                </code>
                <span className="revisions-popover-git-commit-node__message">{(node.subject || '').slice(0, 200)}</span>
                {isCurrent && (
                    <CircleChevronLeftIcon
                        className="icon-inline connection-popover__node-link-icon"
                        data-tooltip="Current commit"
                    />
                )}
            </Link>
        </li>
    )
}

interface Props {
    repo: GQL.Scalars['ID']
    repoName: string
    defaultBranch: string

    /** The current revision, or undefined for the default branch. */
    currentRev: string | undefined

    currentCommitID?: string

    history: H.History
    location: H.Location
}

type RevisionsPopoverTabID = 'branches' | 'tags' | 'commits'

interface RevisionsPopoverTab {
    id: RevisionsPopoverTabID
    label: string
    noun: string
    pluralNoun: string
    type?: GQL.GitRefType
}

/**
 * A popover that displays a searchable list of revisions (grouped by type) for
 * the current repository.
 */
export class RevisionsPopover extends React.PureComponent<Props> {
    private static LAST_TAB_STORAGE_KEY = 'RevisionsPopover.lastTab'

    private static TABS: RevisionsPopoverTab[] = [
        { id: 'branches', label: 'Branches', noun: 'branch', pluralNoun: 'branches', type: GQL.GitRefType.GIT_BRANCH },
        { id: 'tags', label: 'Tags', noun: 'tag', pluralNoun: 'tags', type: GQL.GitRefType.GIT_TAG },
        { id: 'commits', label: 'Commits', noun: 'commit', pluralNoun: 'commits' },
    ]

    public componentDidMount(): void {
        eventLogger.logViewEvent('RevisionsPopover')
    }

    public render(): JSX.Element | null {
        return (
            <div className="revisions-popover connection-popover">
                <TabsWithLocalStorageViewStatePersistence
                    tabs={RevisionsPopover.TABS}
                    storageKey={RevisionsPopover.LAST_TAB_STORAGE_KEY}
                    className="revisions-popover__tabs"
                >
                    {RevisionsPopover.TABS.map(tab =>
                        tab.type ? (
                            <FilteredConnection<GQL.GitRef, Omit<GitRefPopoverNodeProps, 'node'>>
                                key={tab.id}
                                className="connection-popover__content"
                                showMoreClassName="connection-popover__show-more"
                                compact={true}
                                noun={tab.noun}
                                pluralNoun={tab.pluralNoun}
                                queryConnection={
                                    tab.type === GQL.GitRefType.GIT_BRANCH ? this.queryGitBranches : this.queryGitTags
                                }
                                nodeComponent={GitReferencePopoverNode}
                                nodeComponentProps={{
                                    defaultBranch: this.props.defaultBranch,
                                    currentRevision: this.props.currentRev,
                                    location: this.props.location,
                                }}
                                defaultFirst={50}
                                autoFocus={true}
                                noSummaryIfAllNodesVisible={true}
                                useURLQuery={false}
                                history={this.props.history}
                                location={this.props.location}
                            />
                        ) : (
                            <FilteredConnection<GQL.GitCommit, Omit<GitCommitNodeProps, 'node'>>
                                key={tab.id}
                                className="connection-popover__content"
                                compact={true}
                                noun={tab.noun}
                                pluralNoun={tab.pluralNoun}
                                queryConnection={this.queryRepositoryCommits}
                                nodeComponent={GitCommitNode}
                                nodeComponentProps={{
                                    currentCommitID: this.props.currentCommitID,
                                    location: this.props.location,
                                }}
                                defaultFirst={15}
                                autoFocus={true}
                                history={this.props.history}
                                location={this.props.location}
                                noSummaryIfAllNodesVisible={true}
                                useURLQuery={false}
                            />
                        )
                    )}
                </TabsWithLocalStorageViewStatePersistence>
            </div>
        )
    }

    private queryGitBranches = (args: FilteredConnectionQueryArgs): Observable<GQL.GitRefConnection> =>
        queryGitReferences({ ...args, repo: this.props.repo, type: GQL.GitRefType.GIT_BRANCH, withBehindAhead: false })

    private queryGitTags = (args: FilteredConnectionQueryArgs): Observable<GQL.GitRefConnection> =>
        queryGitReferences({ ...args, repo: this.props.repo, type: GQL.GitRefType.GIT_TAG, withBehindAhead: false })

    private queryRepositoryCommits = (args: FilteredConnectionQueryArgs): Observable<GQL.GitCommitConnection> =>
        fetchRepositoryCommits({
            ...args,
            repo: this.props.repo,
            revision: this.props.currentRev || this.props.defaultBranch,
        })
}
