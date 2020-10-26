import { from, Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { dataOrThrowErrors, gql } from '../graphql/graphql'
import * as GQL from '../graphql/schema'
import { PlatformContext } from '../platform/context'
import { memoizeObservable } from '../util/memoizeObservable'
import { RepoSpec } from '../util/url'
import { CloneInProgressError, RepoNotFoundError } from './errors'

/**
 * @returns Observable that emits the `rawRepoName`. Errors with a `CloneInProgressError` if the repo is still being cloned.
 */
export const resolveRawRepoName = memoizeObservable(
    ({
        requestGraphQL,
        repoName,
    }: Pick<RepoSpec, 'repoName'> & Pick<PlatformContext, 'requestGraphQL'>): Observable<string> =>
        from(
            requestGraphQL<GQL.IQuery>({
                request: gql`
                    query ResolveRawRepoName($repoName: String!) {
                        repository(name: $repoName) {
                            uri
                            mirrorInfo {
                                cloned
                            }
                        }
                    }
                `,
                variables: { repoName },
                mightContainPrivateInfo: true,
            })
        ).pipe(
            map(dataOrThrowErrors),
            map(({ repository }) => {
                if (!repository) {
                    throw new RepoNotFoundError(repoName)
                }
                if (!repository.mirrorInfo.cloned) {
                    throw new CloneInProgressError(repoName)
                }
                return repository.uri
            })
        ),
    ({ repoName }) => repoName
)
