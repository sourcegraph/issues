import { Hoverifier } from '@sourcegraph/codeintellify'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { ActionItemAction } from '../../../../shared/src/actions/ActionItem'
import { HoverMerged } from '../../../../shared/src/api/client/types/hover'
import { ExtensionsControllerProps } from '../../../../shared/src/extensions/controller'
import { gql } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { PlatformContextProps } from '../../../../shared/src/platform/context'
import { createAggregateError } from '../../../../shared/src/util/errors'
import { FileSpec, RepoSpec, ResolvedRevisionSpec, RevisionSpec } from '../../../../shared/src/util/url'
import { queryGraphQL } from '../../backend/graphql'
import { FileDiffConnection } from '../../components/diff/FileDiffConnection'
import { FileDiffNode } from '../../components/diff/FileDiffNode'
import { RepositoryCompareAreaPageProps } from './RepositoryCompareArea'
import { ThemeProps } from '../../../../shared/src/theme'
import { fileDiffFields, diffStatFields } from '../../backend/diff'

export function queryRepositoryComparisonFileDiffs(args: {
    repo: GQL.ID
    base: string | null
    head: string | null
    first?: number
    after?: string
    isLightTheme: boolean
}): Observable<GQL.IFileDiffConnection> {
    return queryGraphQL(
        gql`
            query RepositoryComparisonDiff(
                $repo: ID!
                $base: String
                $head: String
                $first: Int
                $after: String
                $isLightTheme: Boolean!
            ) {
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
        ExtensionsControllerProps,
        ThemeProps {
    /** The base of the comparison. */
    base: { repoName: string; repoID: GQL.ID; revision: string | null; commitID: string }

    /** The head of the comparison. */
    head: { repoName: string; repoID: GQL.ID; revision: string | null; commitID: string }
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
                    updateOnChange={String(this.props.isLightTheme)}
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
            isLightTheme: this.props.isLightTheme,
        })
}
