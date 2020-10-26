import * as Monaco from 'monaco-editor'
import { Observable, fromEventPattern, of } from 'rxjs'
import { parseSearchQuery } from './parser'
import { map, first, takeUntil, publishReplay, refCount, switchMap, debounceTime, share } from 'rxjs/operators'
import { getMonacoTokens } from './tokens'
import { getDiagnostics } from './diagnostics'
import { getCompletionItems } from './completion'
import { getHoverResult } from './hover'
import { SearchSuggestion } from '../suggestions'
import { SearchPatternType } from '../../graphql-operations'

interface SearchFieldProviders {
    tokens: Monaco.languages.TokensProvider
    hover: Monaco.languages.HoverProvider
    completion: Monaco.languages.CompletionItemProvider
    diagnostics: Observable<Monaco.editor.IMarkerData[]>
}

/**
 * A dummy parsing state, required for the token provider.
 */
const PARSER_STATE: Monaco.languages.IState = {
    clone: () => ({ ...PARSER_STATE }),
    equals: () => false,
}

const printable = ' !"#$%&\'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~'
const latin1Alpha = 'ÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÐÑÒÓÔÕÖ×ØÙÚÛÜÝÞßàáâãäåæçèéêëìíîïðñòóôõö÷øùúûüýþÿ'

/**
 * Returns the providers used by the Monaco query input to provide syntax highlighting,
 * hovers, completions and diagnostics for the Sourcegraph search syntax.
 */
export function getProviders(
    searchQueries: Observable<string>,
    fetchSuggestions: (input: string) => Observable<SearchSuggestion[]>,
    options: {
        patternType: SearchPatternType
        globbing: boolean
        interpretComments?: boolean
    }
): SearchFieldProviders {
    const parsedQueries = searchQueries.pipe(
        map(rawQuery => {
            const parsed = parseSearchQuery(rawQuery, options.interpretComments ?? false)
            return { rawQuery, parsed }
        }),
        publishReplay(1),
        refCount()
    )

    const debouncedDynamicSuggestions = searchQueries.pipe(debounceTime(300), switchMap(fetchSuggestions), share())

    return {
        tokens: {
            getInitialState: () => PARSER_STATE,
            tokenize: line => {
                const result = parseSearchQuery(line, options.interpretComments ?? false)
                if (result.type === 'success') {
                    return {
                        tokens: getMonacoTokens(result.token),
                        endState: PARSER_STATE,
                    }
                }
                return { endState: PARSER_STATE, tokens: [] }
            },
        },
        hover: {
            provideHover: (textModel, position, token) =>
                parsedQueries
                    .pipe(
                        first(),
                        map(({ parsed }) => (parsed.type === 'error' ? null : getHoverResult(parsed.token, position))),
                        takeUntil(fromEventPattern(handler => token.onCancellationRequested(handler)))
                    )
                    .toPromise(),
        },
        completion: {
            // An explicit list of trigger characters is needed for the Monaco editor to show completions.
            triggerCharacters: [...printable, ...latin1Alpha],
            provideCompletionItems: (textModel, position, context, token) =>
                parsedQueries
                    .pipe(
                        first(),
                        switchMap(parsedQuery =>
                            parsedQuery.parsed.type === 'error'
                                ? of(null)
                                : getCompletionItems(
                                      parsedQuery.parsed.token,
                                      position,
                                      debouncedDynamicSuggestions,
                                      options.globbing
                                  )
                        ),
                        takeUntil(fromEventPattern(handler => token.onCancellationRequested(handler)))
                    )
                    .toPromise(),
        },
        diagnostics: parsedQueries.pipe(
            map(({ parsed }) => (parsed.type === 'success' ? getDiagnostics(parsed.token, options.patternType) : []))
        ),
    }
}
