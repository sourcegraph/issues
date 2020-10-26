import React, { useCallback, useEffect, useMemo, useState } from 'react'
import * as H from 'history'
import { ThemeProps } from '../../../shared/src/theme'
import { PageTitle } from '../components/PageTitle'
import * as Monaco from 'monaco-editor'
import { MonacoEditor } from '../components/MonacoEditor'
import * as GQL from '../../../shared/src/graphql/schema'
import { SearchResultsList, SearchResultsListProps } from './results/SearchResultsList'
import { ErrorLike } from '../../../shared/src/util/errors'
import { addSourcegraphSearchCodeIntelligence } from './input/MonacoQueryInput'
import { BehaviorSubject, concat, NEVER, of } from 'rxjs'
import { useObservable } from '../../../shared/src/util/useObservable'
import { search } from './backend'
import { ExtensionsControllerProps } from '../../../shared/src/extensions/controller'
import { Omit } from 'utility-types'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { parseSearchURLQuery, parseSearchURLPatternType } from '.'
import { SearchPatternType } from '../graphql-operations'

interface SearchConsolePageProps
    extends ThemeProps,
        Omit<
            SearchResultsListProps,
            | 'extensionsController'
            | 'onSavedQueryModalClose'
            | 'onShowMoreResultsClick'
            | 'onExpandAllResultsToggle'
            | 'onSavedQueryModalClose'
            | 'onDidCreateSavedQuery'
            | 'onSaveQueryClick'
        >,
        ExtensionsControllerProps<'executeCommand' | 'services' | 'extHostAPI'> {
    globbing: boolean
    history: H.History
    location: H.Location
}

const options: Monaco.editor.IEditorOptions = {
    readOnly: false,
    minimap: {
        enabled: false,
    },
    lineNumbers: 'off',
    fontSize: 14,
    glyphMargin: false,
    overviewRulerBorder: false,
    rulers: [],
    overviewRulerLanes: 0,
    wordBasedSuggestions: false,
    quickSuggestions: false,
    fixedOverflowWidgets: true,
    renderLineHighlight: 'none',
    contextmenu: false,
    links: true,
    // Display the cursor as a 1px line.
    cursorStyle: 'line',
    cursorWidth: 1,
}

export const SearchConsolePage: React.FunctionComponent<SearchConsolePageProps> = props => {
    const searchQuery = useMemo(() => new BehaviorSubject<string>(parseSearchURLQuery(props.location.search) ?? ''), [
        props.location.search,
    ])
    const patternType = useMemo(
        () => parseSearchURLPatternType(props.location.search) || SearchPatternType.structural,
        [props.location.search]
    )
    const triggerSearch = useCallback(() => {
        props.history.push('/search/console?q=' + encodeURIComponent(searchQuery.value))
    }, [props.history, searchQuery])
    // Fetch search results when the `q` URL query parameter changes
    const resultsOrError = useObservable<'loading' | GQL.ISearchResults | ErrorLike>(
        useMemo(() => {
            const query = parseSearchURLQuery(props.location.search)
            return query
                ? concat(
                      of('loading' as const),
                      search(
                          query.replace(/\/\/.*/g, ''),
                          'V2',
                          patternType,
                          undefined,
                          props.extensionsController.extHostAPI
                      )
                  )
                : NEVER
        }, [patternType, props.extensionsController, props.location.search])
    )
    const [allExpanded, setAllExpanded] = useState(false)
    const [monacoInstance, setMonacoInstance] = useState<typeof Monaco>()
    const { globbing } = props
    useEffect(() => {
        if (!monacoInstance) {
            return
        }
        const subscription = addSourcegraphSearchCodeIntelligence(monacoInstance, searchQuery, {
            patternType,
            globbing,
            interpretComments: true,
        })
        return () => subscription.unsubscribe()
    }, [monacoInstance, searchQuery, patternType, globbing])
    const [editorInstance, setEditorInstance] = useState<Monaco.editor.IStandaloneCodeEditor>()
    useEffect(() => {
        if (!editorInstance) {
            return
        }
        const disposable = editorInstance.onDidChangeModelContent(() => {
            const query = editorInstance.getValue()
            searchQuery.next(query)
        })
        return () => disposable.dispose()
    }, [editorInstance, searchQuery, props.history])
    useEffect(() => {
        if (!editorInstance) {
            return
        }
        const disposable = editorInstance.addAction({
            id: 'submit-on-cmd-enter',
            label: 'Submit search',
            keybindings: [Monaco.KeyMod.CtrlCmd | Monaco.KeyCode.Enter],
            run: () => triggerSearch(),
        })
        return () => disposable.dispose()
    }, [editorInstance, triggerSearch])

    const onExpandAllResultsToggle = useCallback((): void => {
        setAllExpanded(allExpanded => {
            props.telemetryService.log(allExpanded ? 'allResultsExpanded' : 'allResultsCollapsed')
            return allExpanded
        })
    }, [setAllExpanded, props.telemetryService])

    const voidCallback = useCallback(() => undefined, [])

    return (
        <div className="w-100 p-2">
            <PageTitle title="Search console" />
            <div className="d-flex">
                <div className="flex-1 p-1">
                    <div className="mb-1 d-flex align-items-center justify-content-between">
                        <div />
                        <button className="btn btn-lg btn-primary" type="button" onClick={triggerSearch}>
                            Search
                        </button>
                    </div>
                    <MonacoEditor
                        {...props}
                        language="sourcegraphSearch"
                        options={options}
                        height={600}
                        editorWillMount={setMonacoInstance}
                        onEditorCreated={setEditorInstance}
                        value={searchQuery.value}
                    />
                </div>
                <div className="flex-1 p-1 search-console-page__results">
                    {resultsOrError &&
                        (resultsOrError === 'loading' ? (
                            <LoadingSpinner />
                        ) : (
                            <SearchResultsList
                                {...props}
                                allExpanded={allExpanded}
                                resultsOrError={resultsOrError}
                                onExpandAllResultsToggle={onExpandAllResultsToggle}
                                showSavedQueryButton={false}
                                onDidCreateSavedQuery={voidCallback}
                                onSavedQueryModalClose={voidCallback}
                                onSaveQueryClick={voidCallback}
                            />
                        ))}
                </div>
            </div>
        </div>
    )
}
