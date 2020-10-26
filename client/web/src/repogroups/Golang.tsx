import { RepogroupMetadata } from './types'
import * as React from 'react'
import { SearchPatternType } from '../graphql-operations'

export const golang: RepogroupMetadata = {
    title: 'Go',
    name: 'golang',
    url: '/golang',
    description: 'Use these search examples to explore Go repositories on GitHub.',
    examples: [
        {
            title: 'Search for usages of the Retry-After header in non-vendor Go files',
            exampleQuery: (
                <>
                    <span className="search-keyword">lang:</span>go <span className="search-keyword">-file:</span>
                    vendor/ Retry-After
                </>
            ),
            patternType: SearchPatternType.literal,
            rawQuery: 'lang:go -file:vendor/ Retry-After',
        },
        {
            title: 'Find examples of sending JSON in a HTTP POST request',
            exampleQuery: (
                <>
                    <span className="search-keyword">lang:</span>go http.Post json
                </>
            ),
            rawQuery: 'lang:go http.Post json',
            patternType: SearchPatternType.regexp,
        },
        {
            title: 'Find error handling examples in Go',
            exampleQuery: (
                <>
                    {'if err != nil {:[_]}'} <span className="search-keyword">lang:</span>go
                </>
            ),
            rawQuery: 'if err != nil {:[_]} lang:go',
            patternType: SearchPatternType.structural,
        },
        {
            title: 'Find usage examples of cmp.Diff with options',
            exampleQuery: (
                <>
                    <span className="search-keyword">lang:go</span> cmp.Diff(:[_], :[_], :[opts])
                </>
            ),
            rawQuery: 'lang:go cmp.Diff(:[_], :[_], :[opts])',
            patternType: SearchPatternType.structural,
        },
        {
            title: 'Find examples for setting timeouts on http.Transport',
            exampleQuery: (
                <>
                    {'http.Transport{:[_], MaxIdleConns: :[idleconns], :[_]}'}{' '}
                    <span className="search-keyword">-file:</span>vendor <span className="search-keyword">lang:</span>go
                </>
            ),
            rawQuery: 'http.Transport{:[_], MaxIdleConns: :[idleconns], :[_]} -file:vendor lang:go',
            patternType: SearchPatternType.structural,
        },
        {
            title: 'Find examples of Switch statements in Go',
            exampleQuery: (
                <>
                    {'switch :[_] := :[_].(type) { :[string] }'} <span className="search-keyword">lang:</span>go{' '}
                </>
            ),
            rawQuery: 'switch :[_] := :[_].(type) { :[string] } lang:go',
            patternType: SearchPatternType.structural,
        },
    ],
    homepageDescription: 'Find code examples in top Go repositories.',
    homepageIcon: 'https://code.benco.io/icon-collection/logos/go-lang.svg',
}
