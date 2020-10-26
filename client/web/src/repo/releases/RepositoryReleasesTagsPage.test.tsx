import React from 'react'
import { createRenderer } from 'react-test-renderer/shallow'
import * as H from 'history'
import { RepositoryReleasesTagsPage } from './RepositoryReleasesTagsPage'
import { IRepository, IGitRef } from '../../../../shared/src/graphql/schema'
import { of } from 'rxjs'

describe('RepositoryReleasesTagsPage', () => {
    const history = H.createMemoryHistory()
    test('renders', () =>
        expect(
            createRenderer().render(
                <RepositoryReleasesTagsPage
                    history={history}
                    location={history.location}
                    repo={{ id: '123' } as IRepository}
                    queryGitReferences={() =>
                        of({
                            totalCount: 0,
                            nodes: [] as IGitRef[],
                            __typename: 'GitRefConnection',
                            pageInfo: { __typename: 'PageInfo', endCursor: '', hasNextPage: false },
                        })
                    }
                />
            )
        ).toMatchSnapshot())
})
