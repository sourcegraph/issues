import * as Monaco from 'monaco-editor'
import { Observable, fromEventPattern, of, asyncScheduler, Subject } from 'rxjs'
import { map, takeUntil, switchMap, debounceTime, share, observeOn } from 'rxjs/operators'

import { SearchPatternType } from '../../graphql-operations'
import { SearchSuggestion } from '../suggestions'

import { getCompletionItems } from './completion'
import { getMonacoTokens } from './decoratedToken'
import { getHoverResult } from './hover'
import { scanSearchQuery } from './scanner'

interface SearchFieldProviders {
    tokens: Monaco.languages.TokensProvider
    hover: Monaco.languages.HoverProvider
    completion: Monaco.languages.CompletionItemProvider
}

/**
 * A dummy scanner state, required for the token provider.
 */
const SCANNER_STATE: Monaco.languages.IState = {
    clone: () => ({ ...SCANNER_STATE }),
    equals: () => false,
}

const printable = ' !"#$%&\'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~'
const latin1Alpha = 'ÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÐÑÒÓÔÕÖ×ØÙÚÛÜÝÞßàáâãäåæçèéêëìíîïðñòóôõö÷øùúûüýþÿ'

/**
 * Returns the providers used by the Monaco query input to provide syntax highlighting,
 * hovers and completions for the Sourcegraph search syntax.
 */
export function getProviders(
    fetchSuggestions: (input: string) => Observable<SearchSuggestion[]>,
    options: {
        patternType: SearchPatternType
        globbing: boolean
        interpretComments?: boolean
        isSourcegraphDotCom?: boolean
    }
): SearchFieldProviders {
    // To debounce the dynamic suggestions we have to pipe them through a Subject and supply the queries in `provideCompletionItems`.
    // Debouncing the `fetchSuggestions` request directly in `provideCompletionItems` would have no effect, since the observables
    // are not connected between `provideCompletionItems` calls.
    const completionRequests = new Subject<string>()
    const debouncedDynamicSuggestions = completionRequests.pipe(debounceTime(300), switchMap(fetchSuggestions), share())

    return {
        tokens: {
            getInitialState: () => SCANNER_STATE,
            tokenize: line => {
                const result = scanSearchQuery(line, options.interpretComments ?? false, options.patternType)
                if (result.type === 'success') {
                    return {
                        tokens: getMonacoTokens(result.term),
                        endState: SCANNER_STATE,
                    }
                }
                return { endState: SCANNER_STATE, tokens: [] }
            },
        },
        hover: {
            provideHover: (textModel, position, token) =>
                of(textModel.getValue())
                    .pipe(
                        map(value => scanSearchQuery(value, options.interpretComments ?? false, options.patternType)),
                        map(scanned => (scanned.type === 'error' ? null : getHoverResult(scanned.term, position))),
                        takeUntil(fromEventPattern(handler => token.onCancellationRequested(handler)))
                    )
                    .toPromise(),
        },
        completion: {
            // An explicit list of trigger characters is needed for the Monaco editor to show completions.
            triggerCharacters: [...printable, ...latin1Alpha],
            provideCompletionItems: (textModel, position, context, token) => {
                const value = textModel.getValue()
                completionRequests.next(value)
                return of(value)
                    .pipe(
                        map(value => scanSearchQuery(value, options.interpretComments ?? false, options.patternType)),
                        switchMap(scanned =>
                            scanned.type === 'error'
                                ? of(null)
                                : getCompletionItems(
                                      scanned.term,
                                      position,
                                      debouncedDynamicSuggestions,
                                      options.globbing,
                                      options.isSourcegraphDotCom
                                  )
                        ),
                        observeOn(asyncScheduler),
                        map(completions => (token.isCancellationRequested ? undefined : completions))
                    )
                    .toPromise()
            },
        },
    }
}
