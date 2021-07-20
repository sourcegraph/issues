import { createMemoryHistory, createLocation } from 'history'
import { noop } from 'lodash'
import React from 'react'
import renderer from 'react-test-renderer'
import { NEVER } from 'rxjs'

import { NOOP_TELEMETRY_SERVICE } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { extensionsController } from '@sourcegraph/shared/src/util/searchTestHelpers'

import { SearchPatternType } from '../../graphql-operations'

import { SearchResultsInfoBar, SearchResultsInfoBarProps } from './SearchResultsInfoBar'

const COMMON_PROPS: Omit<SearchResultsInfoBarProps, 'enableCodeMonitoring'> = {
    extensionsController,
    platformContext: { forceUpdateTooltip: noop, settings: NEVER },
    history: createMemoryHistory(),
    location: createLocation('/search'),
    authenticatedUser: { id: 'userID' },
    resultsFound: true,
    allExpanded: true,
    onExpandAllResultsToggle: noop,
    onSaveQueryClick: noop,
    stats: <div />,
    telemetryService: NOOP_TELEMETRY_SERVICE,
    patternType: SearchPatternType.literal,
    caseSensitive: false,
}

describe('SearchResultsInfoBar', () => {
    test('code monitoring feature flag disabled', () => {
        expect(
            renderer
                .create(<SearchResultsInfoBar {...COMMON_PROPS} enableCodeMonitoring={false} query="foo type:diff" />)
                .toJSON()
        ).toMatchSnapshot()
    })

    test('code monitoring feature flag enabled, cannot create monitor from query', () => {
        expect(
            renderer.create(<SearchResultsInfoBar {...COMMON_PROPS} enableCodeMonitoring={true} query="foo" />).toJSON()
        ).toMatchSnapshot()
    })

    test('code monitoring feature flag enabled, can create monitor from query', () => {
        expect(
            renderer
                .create(<SearchResultsInfoBar {...COMMON_PROPS} enableCodeMonitoring={true} query="foo type:diff" />)
                .toJSON()
        ).toMatchSnapshot()
    })

    test('code monitoring feature flag enabled, can create monitor from query, user not logged in', () => {
        expect(
            renderer
                .create(
                    <SearchResultsInfoBar
                        {...COMMON_PROPS}
                        enableCodeMonitoring={true}
                        query="foo type:diff"
                        authenticatedUser={null}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })

    test('unauthenticated user', () => {
        expect(
            renderer
                .create(
                    <SearchResultsInfoBar
                        {...COMMON_PROPS}
                        enableCodeMonitoring={true}
                        query="foo type:diff"
                        authenticatedUser={null}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })
})
