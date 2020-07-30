import React from 'react'
import { createRenderer } from 'react-test-renderer/shallow'
import * as H from 'history'
import { RepositoryReleasesTagsPage } from './RepositoryReleasesTagsPage'
import { Repository, GitRef } from '../../../../shared/src/graphql/schema'
import { of } from 'rxjs'

describe('RepositoryReleasesTagsPage', () => {
    const history = H.createMemoryHistory()
    test('renders', () =>
        expect(
            createRenderer().render(
                <RepositoryReleasesTagsPage
                    history={history}
                    location={history.location}
                    repo={{ id: '123' } as Repository}
                    queryGitReferences={() =>
                        of({
                            totalCount: 0,
                            nodes: [] as GitRef[],
                            __typename: 'GitRefConnection',
                            pageInfo: { __typename: 'PageInfo', endCursor: '', hasNextPage: false },
                        })
                    }
                />
            )
        ).toMatchSnapshot())
})
