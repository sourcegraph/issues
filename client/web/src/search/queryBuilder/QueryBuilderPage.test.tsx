import React from 'react'
import { createRenderer } from 'react-test-renderer/shallow'
import { QueryBuilderPage } from './QueryBuilderPage'
import { SearchPatternType } from '../../graphql-operations'

describe('QueryBuilderPage', () => {
    test('simple', () =>
        expect(
            createRenderer().render(
                <QueryBuilderPage patternType={SearchPatternType.literal} versionContext={undefined} />
            )
        ).toMatchSnapshot())
})
