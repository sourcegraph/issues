import { storiesOf } from '@storybook/react'
import { createBrowserHistory } from 'history'
import React from 'react'
import { NEVER, of } from 'rxjs'
import sinon from 'sinon'

import { SearchPatternType } from '@sourcegraph/shared/src/graphql-operations'
import { AggregateStreamingSearchResults } from '@sourcegraph/shared/src/search/stream'
import { NOOP_TELEMETRY_SERVICE } from '@sourcegraph/shared/src/telemetry/telemetryService'
import {
    extensionsController,
    HIGHLIGHTED_FILE_LINES_LONG,
    MULTIPLE_SEARCH_RESULT,
    REPO_MATCH_RESULT,
} from '@sourcegraph/shared/src/util/searchTestHelpers'

import { AuthenticatedUser } from '../../auth'
import { WebStory } from '../../components/WebStory'
import { EMPTY_FEATURE_FLAGS } from '../../featureFlags/featureFlags'

import { StreamingSearchResults, StreamingSearchResultsProps } from './StreamingSearchResults'

const history = createBrowserHistory()
history.replace({ search: 'q=r:golang/oauth2+test+f:travis' })

const streamingSearchResult: AggregateStreamingSearchResults = {
    state: 'complete',
    results: [...MULTIPLE_SEARCH_RESULT.results, REPO_MATCH_RESULT],
    filters: MULTIPLE_SEARCH_RESULT.filters,
    progress: {
        durationMs: 500,
        matchCount: MULTIPLE_SEARCH_RESULT.progress.matchCount,
        skipped: [],
    },
}

const defaultProps: StreamingSearchResultsProps = {
    parsedSearchQuery: 'r:golang/oauth2 test f:travis',
    caseSensitive: false,
    patternType: SearchPatternType.literal,
    versionContext: undefined,
    availableVersionContexts: [],
    previousVersionContext: null,

    extensionsController,
    telemetryService: NOOP_TELEMETRY_SERVICE,

    history,
    location: history.location,
    authenticatedUser: null,
    isLightTheme: true,

    navbarSearchQueryState: { query: '' },
    onNavbarQueryChange: () => {},
    isSourcegraphDotCom: false,

    settingsCascade: {
        subjects: null,
        final: null,
    },
    platformContext: { forceUpdateTooltip: sinon.spy(), settings: NEVER },

    streamSearch: () => of(streamingSearchResult),

    fetchHighlightedFileLineRanges: () => of(HIGHLIGHTED_FILE_LINES_LONG),
    enableCodeMonitoring: false,
    featureFlags: EMPTY_FEATURE_FLAGS,
}

const { add } = storiesOf('web/search/results/StreamingSearchResults', module).addParameters({
    chromatic: { viewports: [577, 769, 993] },
})

add('standard render', () => <WebStory>{() => <StreamingSearchResults {...defaultProps} />}</WebStory>)

add('no results', () => {
    const result: AggregateStreamingSearchResults = {
        state: 'complete',
        results: [],
        filters: [],
        progress: {
            durationMs: 500,
            matchCount: 0,
            skipped: [],
        },
    }

    return <WebStory>{() => <StreamingSearchResults {...defaultProps} streamSearch={() => of(result)} />}</WebStory>
})

add('diffs tab selected, code monitoring enabled, user logged in', () => (
    <WebStory>
        {() => (
            <StreamingSearchResults
                {...defaultProps}
                parsedSearchQuery="r:golang/oauth2 test f:travis type:diff"
                enableCodeMonitoring={true}
                authenticatedUser={
                    {
                        url: '/users/alice',
                        displayName: 'Alice',
                        username: 'alice',
                        email: 'alice@email.test',
                    } as AuthenticatedUser
                }
            />
        )}
    </WebStory>
))

add('code tab selected, code monitoring enabled, user logged in', () => (
    <WebStory>
        {() => (
            <StreamingSearchResults
                {...defaultProps}
                parsedSearchQuery="r:golang/oauth2 test f:travis"
                enableCodeMonitoring={true}
                authenticatedUser={
                    {
                        url: '/users/alice',
                        displayName: 'Alice',
                        username: 'alice',
                        email: 'alice@email.test',
                    } as AuthenticatedUser
                }
            />
        )}
    </WebStory>
))

add('search with quotes', () => (
    <WebStory>
        {() => <StreamingSearchResults {...defaultProps} parsedSearchQuery='r:golang/oauth2 test f:travis "test"' />}
    </WebStory>
))

add('progress with warnings', () => {
    const result: AggregateStreamingSearchResults = {
        state: 'complete',
        results: MULTIPLE_SEARCH_RESULT.results,
        filters: MULTIPLE_SEARCH_RESULT.filters,
        progress: {
            durationMs: 500,
            matchCount: MULTIPLE_SEARCH_RESULT.progress.matchCount,
            skipped: [
                {
                    reason: 'excluded-fork',
                    message: '10k forked repositories excluded',
                    severity: 'info',
                    title: '10k forked repositories excluded',
                    suggested: {
                        title: 'forked:yes',
                        queryExpression: 'forked:yes',
                    },
                },
                {
                    reason: 'excluded-archive',
                    message: '60k archived repositories excluded',
                    severity: 'info',
                    title: '60k archived repositories excluded',
                    suggested: {
                        title: 'archived:yes',
                        queryExpression: 'archived:yes',
                    },
                },
                {
                    reason: 'shard-timedout',
                    message: 'Search timed out',
                    severity: 'warn',
                    title: 'Search timed out',
                    suggested: {
                        title: 'timeout:2m',
                        queryExpression: 'timeout:2m',
                    },
                },
            ],
        },
    }

    return <WebStory>{() => <StreamingSearchResults {...defaultProps} streamSearch={() => of(result)} />}</WebStory>
})

add('show version context warning', () => {
    const history = createBrowserHistory()
    history.replace({ search: 'q=r:golang/oauth2+test+f:travis&c=test' })

    return (
        <WebStory>
            {() => (
                <StreamingSearchResults
                    {...defaultProps}
                    history={history}
                    location={history.location}
                    previousVersionContext={null}
                    availableVersionContexts={[
                        { name: 'test', revisions: [] },
                        { name: 'other', revisions: [] },
                    ]}
                />
            )}
        </WebStory>
    )
})

add('loading with no results', () => (
    <WebStory>{() => <StreamingSearchResults {...defaultProps} streamSearch={() => NEVER} />}</WebStory>
))

add('loading with some results', () => {
    const result: AggregateStreamingSearchResults = {
        state: 'loading',
        results: MULTIPLE_SEARCH_RESULT.results,
        filters: MULTIPLE_SEARCH_RESULT.filters,
        progress: {
            durationMs: 500,
            matchCount: MULTIPLE_SEARCH_RESULT.progress.matchCount,
            skipped: [],
        },
    }

    return <WebStory>{() => <StreamingSearchResults {...defaultProps} streamSearch={() => of(result)} />}</WebStory>
})

add('server-side alert', () => {
    const result: AggregateStreamingSearchResults = {
        state: 'complete',
        results: MULTIPLE_SEARCH_RESULT.results,
        filters: MULTIPLE_SEARCH_RESULT.filters,
        progress: {
            durationMs: 500,
            matchCount: MULTIPLE_SEARCH_RESULT.progress.matchCount,
            skipped: [],
        },
        alert: {
            proposedQueries: [{ query: 'test', description: 'new query' }],
            title: 'Test alert',
            description: 'This is an alert',
        },
    }

    return <WebStory>{() => <StreamingSearchResults {...defaultProps} streamSearch={() => of(result)} />}</WebStory>
})

add('server-side alert with no results', () => {
    const result: AggregateStreamingSearchResults = {
        state: 'complete',
        results: [],
        filters: [],
        progress: {
            durationMs: 500,
            matchCount: MULTIPLE_SEARCH_RESULT.progress.matchCount,
            skipped: [],
        },
        alert: {
            proposedQueries: [{ query: 'test', description: 'Test query' }],
            title: 'Test Alert',
            description: 'This is a test alert',
        },
    }

    return <WebStory>{() => <StreamingSearchResults {...defaultProps} streamSearch={() => of(result)} />}</WebStory>
})

add('error with no results', () => {
    const result: AggregateStreamingSearchResults = {
        state: 'error',
        results: [],
        filters: [],
        progress: {
            durationMs: 500,
            matchCount: MULTIPLE_SEARCH_RESULT.progress.matchCount,
            skipped: [],
        },
        error: new Error('test error'),
    }

    return <WebStory>{() => <StreamingSearchResults {...defaultProps} streamSearch={() => of(result)} />}</WebStory>
})

add('error with some results', () => {
    const result: AggregateStreamingSearchResults = {
        state: 'error',
        results: MULTIPLE_SEARCH_RESULT.results,
        filters: MULTIPLE_SEARCH_RESULT.filters,
        progress: {
            durationMs: 500,
            matchCount: MULTIPLE_SEARCH_RESULT.progress.matchCount,
            skipped: [],
        },
        error: new Error('test error'),
    }

    return <WebStory>{() => <StreamingSearchResults {...defaultProps} streamSearch={() => of(result)} />}</WebStory>
})

add('limit hit with some results', () => {
    const result: AggregateStreamingSearchResults = {
        state: 'complete',
        results: MULTIPLE_SEARCH_RESULT.results,
        filters: MULTIPLE_SEARCH_RESULT.filters,
        progress: {
            durationMs: 500,
            matchCount: MULTIPLE_SEARCH_RESULT.progress.matchCount,
            skipped: [
                {
                    reason: 'document-match-limit',
                    message: 'result limit hit',
                    severity: 'info',
                    title: 'result limit hit',
                },
            ],
        },
    }

    return <WebStory>{() => <StreamingSearchResults {...defaultProps} streamSearch={() => of(result)} />}</WebStory>
})

add('results with signup CTA', () => {
    const result: AggregateStreamingSearchResults = {
        state: 'complete',
        results: MULTIPLE_SEARCH_RESULT.results,
        filters: MULTIPLE_SEARCH_RESULT.filters,
        progress: {
            durationMs: 500,
            matchCount: MULTIPLE_SEARCH_RESULT.progress.matchCount,
            skipped: [],
        },
    }

    return <WebStory>{() => <StreamingSearchResults {...defaultProps} streamSearch={() => of(result)} />}</WebStory>
})
