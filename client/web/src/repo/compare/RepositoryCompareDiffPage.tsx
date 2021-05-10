import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'

import { Hoverifier } from '@sourcegraph/codeintellify'
import { ActionItemAction } from '@sourcegraph/shared/src/actions/ActionItem'
import { HoverMerged } from '@sourcegraph/shared/src/api/client/types/hover'
import { ExtensionsControllerProps } from '@sourcegraph/shared/src/extensions/controller'
import { Scalars } from '@sourcegraph/shared/src/graphql-operations'
import { gql } from '@sourcegraph/shared/src/graphql/graphql'
import * as GQL from '@sourcegraph/shared/src/graphql/schema'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { createAggregateError } from '@sourcegraph/shared/src/util/errors'
import { FileSpec, RepoSpec, ResolvedRevisionSpec, RevisionSpec } from '@sourcegraph/shared/src/util/url'

import { fileDiffFields, diffStatFields } from '../../backend/diff'
import { queryGraphQL } from '../../backend/graphql'
import { FileDiffConnection } from '../../components/diff/FileDiffConnection'
import { FileDiffNode } from '../../components/diff/FileDiffNode'

import { RepositoryCompareAreaPageProps } from './RepositoryCompareArea'

export function queryRepositoryComparisonFileDiffs(args: {
    repo: Scalars['ID']
    base: string | null
    head: string | null
    first?: number
    after?: string
}): Observable<GQL.IFileDiffConnection> {
    return queryGraphQL(
        gql`
            query RepositoryComparisonDiff($repo: ID!, $base: String, $head: String, $first: Int, $after: String) {
                node(id: $repo) {
                    ... on Repository {
                        comparison(base: $base, head: $head) {
                            fileDiffs(first: $first, after: $after) {
                                nodes {
                                    ...FileDiffFields
                                }
                                totalCount
                                pageInfo {
                                    endCursor
                                    hasNextPage
                                }
                                diffStat {
                                    ...DiffStatFields
                                }
                            }
                        }
                    }
                }
            }

            ${fileDiffFields}

            ${diffStatFields}
        `,
        args
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.node) {
                throw createAggregateError(errors)
            }
            const repo = data.node as GQL.IRepository
            if (!repo.comparison || !repo.comparison.fileDiffs || errors) {
                throw createAggregateError(errors)
            }
            return repo.comparison.fileDiffs
        })
    )
}

interface RepositoryCompareDiffPageProps
    extends RepositoryCompareAreaPageProps,
        RouteComponentProps<{}>,
        PlatformContextProps,
        ThemeProps,
        ExtensionsControllerProps {
    /** The base of the comparison. */
    base: { repoName: string; repoID: Scalars['ID']; revision: string | null; commitID: string }

    /** The head of the comparison. */
    head: { repoName: string; repoID: Scalars['ID']; revision: string | null; commitID: string }
    hoverifier: Hoverifier<RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec, HoverMerged, ActionItemAction>
}

/** A page with the file diffs in the comparison. */
export class RepositoryCompareDiffPage extends React.PureComponent<RepositoryCompareDiffPageProps> {
    public render(): JSX.Element | null {
        return (
            <div className="repository-compare-page">
                <FileDiffConnection
                    listClassName="list-group list-group-flush"
                    noun="changed file"
                    pluralNoun="changed files"
                    queryConnection={this.queryDiffs}
                    nodeComponent={FileDiffNode}
                    nodeComponentProps={{
                        ...this.props,
                        extensionInfo: {
                            base: { ...this.props.base, revision: this.props.base.revision || 'HEAD' },
                            head: { ...this.props.head, revision: this.props.head.revision || 'HEAD' },
                            hoverifier: this.props.hoverifier,
                            extensionsController: this.props.extensionsController,
                        },
                        lineNumbers: true,
                    }}
                    defaultFirst={15}
                    hideSearch={true}
                    noSummaryIfAllNodesVisible={true}
                    history={this.props.history}
                    location={this.props.location}
                    cursorPaging={true}
                />
            </div>
        )
    }

    private queryDiffs = (args: { first?: number }): Observable<GQL.IFileDiffConnection> =>
        queryRepositoryComparisonFileDiffs({
            ...args,
            repo: this.props.repo.id,
            base: this.props.base.commitID,
            head: this.props.head.commitID,
        })
}
