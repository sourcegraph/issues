import React, { useEffect, useLayoutEffect, useRef, useState } from 'react'
import * as H from 'history'
import * as Monaco from 'monaco-editor'
import { isPlainObject } from 'lodash'
import { MonacoEditor } from '../../components/MonacoEditor'
import { QueryState } from '../helpers'
import { getProviders } from '../../../../shared/src/search/query/providers'
import { fetchSuggestions } from '../backend'
import { Omit } from 'utility-types'
import { ThemeProps } from '../../../../shared/src/theme'
import { CaseSensitivityProps, PatternTypeProps, CopyQueryButtonProps, SearchContextProps } from '..'
import { Toggles, TogglesProps } from './toggles/Toggles'
import { hasProperty } from '../../../../shared/src/util/types'
import { KeyboardShortcut } from '../../../../shared/src/keyboardShortcuts'
import { KEYBOARD_SHORTCUT_FOCUS_SEARCHBAR } from '../../keyboardShortcuts/keyboardShortcuts'
import { observeResize } from '../../util/dom'
import { SearchPatternType } from '../../graphql-operations'
import { scanSearchQuery } from '../../../../shared/src/search/query/scanner'
import { getDiagnostics } from '../../../../shared/src/search/query/diagnostics'

export interface MonacoQueryInputProps
    extends Omit<TogglesProps, 'navbarSearchQuery'>,
        ThemeProps,
        CaseSensitivityProps,
        PatternTypeProps,
        SearchContextProps,
        CopyQueryButtonProps {
    location: H.Location
    history: H.History
    queryState: QueryState
    onChange: (newState: QueryState) => void
    onSubmit: () => void
    onFocus?: () => void
    onCompletionItemSelected?: () => void
    onSuggestionsInitialized?: (actions: { trigger: () => void }) => void
    autoFocus?: boolean
    keyboardShortcutForFocus?: KeyboardShortcut

    // Whether globbing is enabled for filters.
    globbing: boolean

    // Whether to additionally highlight or provide hovers for tokens, e.g., regexp character sets.
    enableSmartQuery: boolean

    // Whether comments are parsed and highlighted
    interpretComments?: boolean
}

const SOURCEGRAPH_SEARCH = 'sourcegraphSearch' as const

/**
 * Adds code intelligence for the Sourcegraph search syntax to Monaco.
 *
 * @returns Subscription
 */
export function useSourcegraphSearchCodeIntelligence(
    rawQuery: string,
    {
        patternType,
        globbing,
        interpretComments,
        enableSmartQuery,
    }: {
        patternType: SearchPatternType
        globbing: boolean
        interpretComments?: boolean
        enableSmartQuery: boolean
    }
): { setMonacoInstance: (monaco: typeof Monaco) => void } {
    const [monacoInstance, setMonacoInstance] = useState<typeof Monaco>()

    // The query is passed as a ref to providers, to avoid re-registering providers on all query changes.
    const queryReference = useRef<{ rawQuery: string; scanned: ReturnType<typeof scanSearchQuery> }>()

    // Scan the query, update the query ref & diagnostics when the query changes.
    // This is done with useLayoutEffect so that the update is synchronous,
    // otherwise providers run off an outdated query.
    useLayoutEffect(() => {
        const scanned = scanSearchQuery(rawQuery)
        queryReference.current = {
            rawQuery,
            scanned,
        }
        if (!monacoInstance) {
            return
        }
        // Set diagnostics
        const diagnostics = scanned.type === 'success' ? getDiagnostics(scanned.term, patternType) : []
        monacoInstance.editor.setModelMarkers(monacoInstance.editor.getModels()[0], 'diagnostics', diagnostics)
    }, [rawQuery, queryReference, monacoInstance, patternType])

    useEffect(() => {
        if (!monacoInstance) {
            return
        }
        console.log({ patternType })
        // Register language ID
        monacoInstance.languages.register({ id: SOURCEGRAPH_SEARCH })

        // Register providers
        const providers = getProviders(queryReference, fetchSuggestions, {
            patternType,
            globbing,
            interpretComments,
            enableSmartQuery,
        })
        const disposables: Monaco.IDisposable[] = [
            monacoInstance.languages.setTokensProvider(SOURCEGRAPH_SEARCH, providers.tokens),
            monacoInstance.languages.registerHoverProvider(SOURCEGRAPH_SEARCH, providers.hover),
            monacoInstance.languages.registerCompletionItemProvider(SOURCEGRAPH_SEARCH, providers.completion),
        ]
        return () => {
            for (const disposable of disposables) {
                disposable.dispose()
            }
        }
    }, [monacoInstance, queryReference, patternType, globbing, interpretComments, enableSmartQuery])

    return { setMonacoInstance }
}

/**
 * HACK: this interface and the below type guard are used to free default Monaco
 * keybindings (such as cmd + F, cmd + L) by unregistering them from the private
 * `_standaloneKeybindingService`.
 *
 * This is necessary as simply registering a noop command with editor.addCommand(keybinding, noop)
 * prevents the default Monaco behaviour, but doesn't free the keybinding, and thus still blocks the
 * default browser action (eg. select location with cmd + L).
 *
 * See upstream issues:
 * - https://github.com/microsoft/monaco-editor/issues/287
 * - https://github.com/microsoft/monaco-editor/issues/102 (main tracking issue)
 */
interface MonacoEditorWithKeybindingsService extends Monaco.editor.IStandaloneCodeEditor {
    _actions: {
        [id: string]: {
            id: string
            alias: string
            label: string
        }
    }
    _standaloneKeybindingService: {
        addDynamicKeybinding(keybinding: string): void
    }
}

const hasKeybindingService = (
    editor: Monaco.editor.IStandaloneCodeEditor
): editor is MonacoEditorWithKeybindingsService =>
    hasProperty('_actions')(editor) &&
    isPlainObject(editor._actions) &&
    hasProperty('_standaloneKeybindingService')(editor) &&
    typeof (editor._standaloneKeybindingService as MonacoEditorWithKeybindingsService['_standaloneKeybindingService'])
        .addDynamicKeybinding === 'function'

/**
 * HACK: this interface and the below type guard are used to add a custom command
 * to the editor. There is no public API to add a command with a specified ID and handler,
 * hence we need to use the private _commandService API.
 *
 * See upstream issue:
 * - https://github.com/Microsoft/monaco-editor/issues/900#issue-327455729
 * */
interface MonacoEditorWithCommandService extends Monaco.editor.IStandaloneCodeEditor {
    _commandService: {
        addCommand: (command: { id: string; handler: () => void }) => void
    }
}

const hasCommandService = (editor: Monaco.editor.IStandaloneCodeEditor): editor is MonacoEditorWithCommandService =>
    hasProperty('_commandService')(editor) &&
    typeof (editor._commandService as MonacoEditorWithCommandService['_commandService']).addCommand === 'function'

/**
 * A search query input backed by the Monaco editor, allowing it to provide
 * syntax highlighting, hovers, completions and diagnostics for search queries.
 *
 * This component should not be imported directly: use {@link LazyMonacoQueryInput} instead
 * to avoid bundling the Monaco editor on every page.
 */
export const MonacoQueryInput: React.FunctionComponent<MonacoQueryInputProps> = ({
    queryState,
    onFocus,
    onChange,
    onSubmit,
    onSuggestionsInitialized,
    onCompletionItemSelected,
    autoFocus,
    ...props
}) => {
    const [editor, setEditor] = useState<Monaco.editor.IStandaloneCodeEditor>()

    // Trigger a layout of the Monaco editor when its container gets resized.
    // The Monaco editor doesn't auto-resize with its container:
    // https://github.com/microsoft/monaco-editor/issues/28
    const [container, setContainer] = useState<HTMLElement | null>()
    useLayoutEffect(() => {
        if (!editor || !container) {
            return
        }
        const subscription = observeResize(container).subscribe(() => {
            editor.layout()
        })
        return () => subscription.unsubscribe()
    }, [editor, container])

    // Register themes and code intelligence providers.
    const { setMonacoInstance } = useSourcegraphSearchCodeIntelligence(queryState.query, props)

    // Register suggestions handle
    useEffect(() => {
        if (editor) {
            onSuggestionsInitialized?.({
                trigger: () => editor.trigger('triggerSuggestions', 'editor.action.triggerSuggest', {}),
            })
        }
    }, [editor, onSuggestionsInitialized])

    // Register onCompletionSelected handler
    useEffect(() => {
        if (!editor || !onCompletionItemSelected) {
            return
        }
        if (!hasCommandService(editor)) {
            throw new Error('Could not call onCompletionItemSelected: editor has no commandService')
        }

        editor._commandService.addCommand({
            id: 'completionItemSelected',
            handler: onCompletionItemSelected,
        })
    }, [editor, onCompletionItemSelected])

    // Disable default Monaco keybindings
    useEffect(() => {
        if (!editor) {
            return
        }
        if (!hasKeybindingService(editor)) {
            // Throw an error if hasKeybindingService() returns false,
            // to surface issues with this workaround when upgrading Monaco.
            throw new Error('Cannot unbind default Monaco keybindings')
        }
        for (const action of Object.keys(editor._actions)) {
            // Keep ctrl+space to show all available completions. Keep ctrl+k to delete text on right of cursor.
            if (action === 'editor.action.triggerSuggest' || action === 'deleteAllRight') {
                continue
            }
            // Prefixing action ids with `-` to unbind the default actions.
            editor._standaloneKeybindingService.addDynamicKeybinding(`-${action}`)
        }
        // Free CMD+L keybinding, which is part of Monaco's CoreNavigationCommands, and
        // not exposed on editor._actions.
        editor._standaloneKeybindingService.addDynamicKeybinding('-expandLineSelection')
    }, [editor])

    // Accessibility: allow tab usage to move focus to
    // next previous focusable element (and not to insert the tab character).
    // - Cannot be set through IEditorOptions
    // - Cannot be called synchronously (otherwise risks being overridden by Monaco defaults)
    useEffect(() => {
        if (!editor) {
            return
        }
        const disposable = editor.onDidFocusEditorText(() => {
            editor.createContextKey('editorTabMovesFocus', true)
        })
        return () => disposable.dispose()
    }, [editor])

    // Focus the editor if the autoFocus prop is truthy
    useEffect(() => {
        if (!editor || !autoFocus) {
            return
        }
        editor.focus()
    }, [editor, autoFocus])

    // If an edit wasn't triggered by the user,
    // place the cursor at the end of the query.
    useEffect(() => {
        if (!editor || queryState.fromUserInput) {
            return
        }
        const position = {
            // +2 as Monaco is 1-indexed.
            column: editor.getValue().length + 2,
            lineNumber: 1,
        }
        editor.setPosition(position)
        editor.revealPosition(position)
    }, [editor, queryState])

    // Prevent newline insertion in model, and surface query changes with stripped newlines.
    useEffect(() => {
        if (!editor) {
            return
        }
        const disposable = editor.onDidChangeModelContent(() => {
            onChange({ query: editor.getValue().replace(/[\n\r↵]/g, ''), fromUserInput: true })
        })
        return () => disposable.dispose()
    }, [editor, onChange])

    // Submit on enter, hiding the suggestions widget if it's visible.
    useEffect(() => {
        if (!editor) {
            return
        }
        const disposable = editor.addAction({
            id: 'submitOnEnter',
            label: 'submitOnEnter',
            keybindings: [Monaco.KeyCode.Enter],
            run: () => {
                onSubmit()
                editor.trigger('submitOnEnter', 'hideSuggestWidget', [])
            },
        })
        return () => disposable.dispose()
    }, [editor, onSubmit])

    const options: Monaco.editor.IEditorOptions = {
        readOnly: false,
        lineNumbers: 'off',
        lineHeight: 16,
        // Match the query input's height for suggestion items line height.
        suggestLineHeight: 34,
        minimap: {
            enabled: false,
        },
        scrollbar: {
            vertical: 'hidden',
            horizontal: 'hidden',
        },
        glyphMargin: false,
        lineDecorationsWidth: 0,
        lineNumbersMinChars: 0,
        overviewRulerBorder: false,
        folding: false,
        rulers: [],
        overviewRulerLanes: 0,
        wordBasedSuggestions: false,
        quickSuggestions: false,
        fixedOverflowWidgets: true,
        contextmenu: false,
        links: false,
        // Display the cursor as a 1px line.
        cursorStyle: 'line',
        cursorWidth: 1,
    }
    return (
        <>
            <div ref={setContainer} className="monaco-query-input-container">
                {props.showSearchContext && <span>context:global</span>}
                <div className="flex-grow-1 flex-shrink-past-contents" onFocus={onFocus}>
                    <MonacoEditor
                        id="monaco-query-input"
                        language={SOURCEGRAPH_SEARCH}
                        value={queryState.query}
                        height={17}
                        isLightTheme={props.isLightTheme}
                        editorWillMount={setMonacoInstance}
                        onEditorCreated={setEditor}
                        options={options}
                        border={false}
                        keyboardShortcutForFocus={KEYBOARD_SHORTCUT_FOCUS_SEARCHBAR}
                        className="test-query-input"
                    />
                </div>
                <Toggles
                    {...props}
                    navbarSearchQuery={queryState.query}
                    className="monaco-query-input-container__toggle-container"
                />
            </div>
        </>
    )
}
