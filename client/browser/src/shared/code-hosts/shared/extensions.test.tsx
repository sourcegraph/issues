import { DEFAULT_SOURCEGRAPH_URL, getAssetsURL } from '../../util/context'
import { initializeExtensions } from './extensions'

describe('Extensions controller', () => {
    it('Blocks GraphQL requests from extensions if they risk leaking private information to the public sourcegraph.com instance', () => {
        window.SOURCEGRAPH_URL = DEFAULT_SOURCEGRAPH_URL
        const { extensionsController } = initializeExtensions(
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
            extensionsController.executeCommand({
                command: 'queryGraphQL',
                arguments: [
                    `
                        query ResolveRepo($repoName: String!) {
                            repository(name: $repoName) {
                                url
                            }
                        }
                    `,
                    { repoName: 'foo' },
                ],
            })
        ).rejects.toMatchObject({
            message:
                'A ResolveRepo GraphQL request to the public Sourcegraph.com was blocked because the current repository is private.',
        })
    })
})
