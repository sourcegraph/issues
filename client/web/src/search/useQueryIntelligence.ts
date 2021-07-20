import * as Monaco from 'monaco-editor'
import { useEffect, useMemo } from 'react'
import { Observable } from 'rxjs'

import { SearchPatternType } from '@sourcegraph/shared/src/graphql/schema'
import { getDiagnostics } from '@sourcegraph/shared/src/search/query/diagnostics'
import { getProviders } from '@sourcegraph/shared/src/search/query/providers'
import { scanSearchQuery } from '@sourcegraph/shared/src/search/query/scanner'
import { SearchSuggestion } from '@sourcegraph/shared/src/search/suggestions'

export const SOURCEGRAPH_SEARCH = 'sourcegraphSearch' as const

/**
 * Adds code intelligence for the Sourcegraph search syntax to Monaco.
 */
export function useQueryIntelligence(
    fetchSuggestions: (query: string) => Observable<SearchSuggestion[]>,
    options: {
        patternType: SearchPatternType
        globbing: boolean
        interpretComments?: boolean
        isSourcegraphDotCom?: boolean
    }
): void {
    const memoizedOptions = useMemo(
        () => ({
            patternType: options.patternType,
            globbing: options.globbing,
            interpretComments: options.interpretComments,
            isSourcegraphDotCom: options.isSourcegraphDotCom,
        }),
        [options.patternType, options.globbing, options.interpretComments, options.isSourcegraphDotCom]
    )

    useEffect(() => {
        // Register language ID
        Monaco.languages.register({ id: SOURCEGRAPH_SEARCH })

        // Register providers
        const providers = getProviders(fetchSuggestions, memoizedOptions)
        const disposables = [
            Monaco.languages.setTokensProvider(SOURCEGRAPH_SEARCH, providers.tokens),
            Monaco.languages.registerHoverProvider(SOURCEGRAPH_SEARCH, providers.hover),
            Monaco.languages.registerCompletionItemProvider(SOURCEGRAPH_SEARCH, providers.completion),
        ]
        return () => {
            for (const disposable of disposables) {
                disposable.dispose()
            }
        }
    }, [fetchSuggestions, memoizedOptions])
}

/**
 * Adds diagnostic markers for the Sourcegraph search syntax to Monaco.
 * For example, it adds a marker if the query contains a filter with an invalid value: `patterntype:invalid`.
 */
export function useQueryDiagnostics(
    editor: Monaco.editor.IStandaloneCodeEditor | undefined,
    options: {
        patternType: SearchPatternType
        interpretComments?: boolean
    }
): void {
    const memoizedOptions = useMemo(
        () => ({
            patternType: options.patternType,
            interpretComments: options.interpretComments,
        }),
        [options.patternType, options.interpretComments]
    )

    useEffect(() => {
        if (!editor) {
            return
        }
        const disposable = editor.onDidChangeModelContent(() => {
            const model = editor.getModel()
            if (!model) {
                return
            }
            const scanned = scanSearchQuery(
                model.getValue(),
                memoizedOptions.interpretComments ?? false,
                memoizedOptions.patternType
            )
            const markers = scanned.type === 'success' ? getDiagnostics(scanned.term, memoizedOptions.patternType) : []
            Monaco.editor.setModelMarkers(model, 'diagnostics', markers)
        })
        return () => disposable.dispose()
    }, [editor, memoizedOptions])
}
