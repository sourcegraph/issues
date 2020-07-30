import { gql } from '../../../../shared/src/graphql/graphql'
import { DEFAULT_SOURCEGRAPH_URL, getAssetsURL } from '../util/context'
import { createPlatformContext } from './context'

describe('Platform Context', () => {
    describe('requestGraphQL()', () => {
        it('throws if the request risks leaking private information to the public sourcegraph.com', () => {
            window.SOURCEGRAPH_URL = DEFAULT_SOURCEGRAPH_URL
            const { requestGraphQL } = createPlatformContext(
                {
                    urlToFile: () => '',
                    getContext: () => ({ rawRepoName: 'foo', privateRepository: true }),
                },
                {
                    sourcegraphURL: DEFAULT_SOURCEGRAPH_URL,
                    assetsURL: getAssetsURL(DEFAULT_SOURCEGRAPH_URL),
                },
                false
            )
            return expect(
                requestGraphQL({
                    request: gql`
                        query ResolveRepoTest($repoName: String!) {
                            repository(name: $repoName) {
                                url
                            }
                        }
                    `,
                    variables: { repoName: 'foo' },
                    mightContainPrivateInfo: true,
                }).toPromise()
            ).rejects.toMatchObject({
                message:
                    'A ResolveRepo GraphQL request to the public Sourcegraph.com was blocked because the current repository is private.',
            })
        })
    })
})
