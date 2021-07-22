import * as H from 'history'
import React from 'react'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'

import { IHighlightLineRange } from '../graphql/schema'
import { ContentMatch, SymbolMatch, PathMatch, getFileMatchUrl } from '../search/stream'
import { isSettingsValid, SettingsCascadeProps } from '../settings/settings'
import { SymbolIcon } from '../symbols/SymbolIcon'
import { ThemeProps } from '../theme'
import { isErrorLike } from '../util/errors'
import {
    appendLineRangeQueryParameter,
    toPositionOrRangeQueryParameter,
    appendSubtreeQueryParameter,
} from '../util/url'

import { CodeExcerpt, FetchFileParameters } from './CodeExcerpt'
import { CodeExcerptUnhighlighted } from './CodeExcerptUnhighlighted'
import { MatchItem } from './FileMatch'
import { MatchGroup, calculateMatchGroups } from './FileMatchContext'
import { LinkWithSelectableText } from './LinkWithSelectableText'

export interface EventLogger {
    log: (eventLabel: string, eventProperties?: any) => void
}

interface FileMatchProps extends SettingsCascadeProps, ThemeProps {
    location: H.Location
    eventLogger?: EventLogger
    items: MatchItem[]
    result: ContentMatch | SymbolMatch | PathMatch
    /* Called when the first result has fully loaded. */
    onFirstResultLoad?: () => void
    /**
     * Whether or not to show all matches for this file, or only a subset.
     */
    allMatches: boolean
    /**
     * The number of matches to show when the results are collapsed (allMatches===false, user has not clicked "Show N more matches")
     */
    subsetMatches: number
    fetchHighlightedFileLineRanges: (parameters: FetchFileParameters, force?: boolean) => Observable<string[][]>
    /**
     * Called when the file's search result is selected.
     */
    onSelect: () => void
}

// Dev flag for disabling syntax highlighting on search results pages.
const NO_SEARCH_HIGHLIGHTING = localStorage.getItem('noSearchHighlighting') !== null

export const FileMatchChildren: React.FunctionComponent<FileMatchProps> = props => {
    // The number of lines of context to show before and after each match.
    let context = 1

    if (props.location.pathname === '/search') {
        // Check if search.contextLines is configured in settings.
        const contextLinesSetting =
            isSettingsValid(props.settingsCascade) &&
            props.settingsCascade.final &&
            props.settingsCascade.final['search.contextLines']

        if (typeof contextLinesSetting === 'number' && contextLinesSetting >= 0) {
            context = contextLinesSetting
        }
    }

    const maxMatches = props.allMatches ? 0 : props.subsetMatches
    const [matches, grouped] = React.useMemo(() => calculateMatchGroups(props.items, maxMatches, context), [
        props.items,
        maxMatches,
        context,
    ])

    // If optimizeHighlighting is enabled, compile a list of the highlighted file ranges we want to
    // fetch (instead of the entire file.)
    const optimizeHighlighting =
        props.settingsCascade.final &&
        !isErrorLike(props.settingsCascade.final) &&
        props.settingsCascade.final.experimentalFeatures &&
        props.settingsCascade.final.experimentalFeatures.enableFastResultLoading

    const { result, isLightTheme, fetchHighlightedFileLineRanges, eventLogger, onFirstResultLoad } = props
    const fetchHighlightedFileRangeLines = React.useCallback(
        (isFirst, startLine, endLine, isLightTheme) => {
            const startTime = Date.now()
            return fetchHighlightedFileLineRanges(
                {
                    repoName: result.repository,
                    commitID: result.version || '',
                    filePath: result.name,
                    disableTimeout: false,
                    isLightTheme,
                    ranges: optimizeHighlighting
                        ? grouped.map(
                              (group): IHighlightLineRange => ({
                                  startLine: group.startLine,
                                  endLine: group.endLine,
                              })
                          )
                        : [{ startLine: 0, endLine: 2147483647 }], // entire file,
                },
                false
            ).pipe(
                map(lines => {
                    if (isFirst && onFirstResultLoad) {
                        onFirstResultLoad()
                    }
                    if (eventLogger) {
                        eventLogger.log('search.latencies.frontend.code-load', { durationMs: Date.now() - startTime })
                    }
                    return optimizeHighlighting
                        ? lines[grouped.findIndex(group => group.startLine === startLine && group.endLine === endLine)]
                        : lines[0].slice(startLine, endLine)
                })
            )
        },
        [result, fetchHighlightedFileLineRanges, grouped, optimizeHighlighting, eventLogger, onFirstResultLoad]
    )

    const createCodeExcerptLink = (group: MatchGroup): string => {
        const positionOrRangeQueryParameter = toPositionOrRangeQueryParameter({ position: group.position })
        return appendLineRangeQueryParameter(
            appendSubtreeQueryParameter(getFileMatchUrl(result)),
            positionOrRangeQueryParameter
        )
    }

    if (NO_SEARCH_HIGHLIGHTING) {
        return (
            <CodeExcerptUnhighlighted
                urlWithoutPosition={getFileMatchUrl(result)}
                items={matches}
                onSelect={props.onSelect}
            />
        )
    }

    return (
        <div className="file-match-children">
            {/* Path */}
            {result.type === 'path' && (
                <div className="file-match-children__item">
                    <small>Path match</small>
                </div>
            )}

            {/* Symbols */}
            {((result.type === 'symbol' && result.symbols) || []).map(symbol => (
                <LinkWithSelectableText
                    to={symbol.url}
                    className="file-match-children__item test-file-match-children-item"
                    key={`symbol:${symbol.name}${String(symbol.containerName)}${symbol.url}`}
                >
                    <SymbolIcon kind={symbol.kind} className="icon-inline mr-1" />
                    <code>
                        {symbol.name}{' '}
                        {symbol.containerName && <span className="text-muted">{symbol.containerName}</span>}
                    </code>
                </LinkWithSelectableText>
            ))}

            {/* Line matches */}
            {grouped.map((group, index) => (
                <div
                    key={`linematch:${getFileMatchUrl(result)}${group.position.line}:${group.position.character}`}
                    className="file-match-children__item-code-wrapper test-file-match-children-item-wrapper"
                >
                    <LinkWithSelectableText
                        to={createCodeExcerptLink(group)}
                        className="file-match-children__item test-file-match-children-item"
                        onClick={props.onSelect}
                    >
                        <CodeExcerpt
                            repoName={result.repository}
                            commitID={result.version || ''}
                            filePath={result.name}
                            startLine={group.startLine}
                            endLine={group.endLine}
                            highlightRanges={group.matches}
                            className="file-match-children__item-code-excerpt"
                            isLightTheme={isLightTheme}
                            fetchHighlightedFileRangeLines={fetchHighlightedFileRangeLines}
                            isFirst={index === 0}
                        />
                    </LinkWithSelectableText>
                </div>
            ))}
        </div>
    )
}
