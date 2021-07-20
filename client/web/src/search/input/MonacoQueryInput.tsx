import classNames from 'classnames'
import { isPlainObject, noop } from 'lodash'
import * as Monaco from 'monaco-editor'
import React, { useCallback, useEffect, useLayoutEffect, useState } from 'react'

import { KeyboardShortcut } from '@sourcegraph/shared/src/keyboardShortcuts'
import { appendContextFilter } from '@sourcegraph/shared/src/search/query/transformer'
import { VersionContextProps } from '@sourcegraph/shared/src/search/util'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { isErrorLike } from '@sourcegraph/shared/src/util/errors'
import { hasProperty } from '@sourcegraph/shared/src/util/types'

import { CaseSensitivityProps, PatternTypeProps, SearchContextProps } from '..'
import { MonacoEditor } from '../../components/MonacoEditor'
import { KEYBOARD_SHORTCUT_FOCUS_SEARCHBAR } from '../../keyboardShortcuts/keyboardShortcuts'
import { observeResize } from '../../util/dom'
import { fetchSuggestions } from '../backend'
import { QueryChangeSource, QueryState } from '../helpers'
import { SOURCEGRAPH_SEARCH, useQueryIntelligence, useQueryDiagnostics } from '../useQueryIntelligence'

export interface MonacoQueryInputProps
    extends ThemeProps,
        Pick<CaseSensitivityProps, 'caseSensitive'>,
        Pick<PatternTypeProps, 'patternType'>,
        Pick<SearchContextProps, 'selectedSearchContextSpec'>,
        SettingsCascadeProps,
        VersionContextProps {
    isSourcegraphDotCom: boolean // significant for query suggestions
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

    // Whether comments are parsed and highlighted
    interpretComments?: boolean

    className?: string
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
        addDynamicKeybinding(
            commandId: string,
            _keybinding: number | undefined,
            handler: Monaco.editor.ICommandHandler
        ): void
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
    selectedSearchContextSpec,
    versionContext,
    patternType,
    globbing,
    interpretComments,
    isSourcegraphDotCom,
    isLightTheme,
    className,
    settingsCascade,
}) => {
    const acceptSearchSuggestionOnEnter: boolean | undefined =
        !isErrorLike(settingsCascade.final) &&
        settingsCascade.final?.experimentalFeatures?.acceptSearchSuggestionOnEnter
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

    const fetchSuggestionsWithContext = useCallback(
        (query: string) => fetchSuggestions(appendContextFilter(query, selectedSearchContextSpec, versionContext)),
        [selectedSearchContextSpec, versionContext]
    )

    useQueryIntelligence(fetchSuggestionsWithContext, {
        patternType,
        globbing,
        interpretComments,
        isSourcegraphDotCom,
    })
    useQueryDiagnostics(editor, { patternType, interpretComments })

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

        Monaco.editor.registerCommand('completionItemSelected', onCompletionItemSelected)
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
            editor._standaloneKeybindingService.addDynamicKeybinding(`-${action}`, undefined, () => {})
        }
        // Free CMD+L keybinding, which is part of Monaco's CoreNavigationCommands, and
        // not exposed on editor._actions.
        editor._standaloneKeybindingService.addDynamicKeybinding('-expandLineSelection', undefined, () => {})
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

    // Always focus the editor on selectedSearchContextSpec change
    useEffect(() => {
        if (selectedSearchContextSpec) {
            editor?.focus()
        }
    }, [editor, selectedSearchContextSpec])

    useEffect(() => {
        if (!editor) {
            return
        }

        switch (queryState.changeSource) {
            case QueryChangeSource.userInput:
                // Don't react to user input
                break
            case QueryChangeSource.searchReference: {
                editor.setSelection(queryState.selection)
                if (queryState.showSuggestions) {
                    editor.trigger('triggerSuggestions', 'editor.action.triggerSuggest', {})
                }
                editor.revealRange(queryState.revealRange)
                break
            }
            default: {
                // Place the cursor at the end of the query.
                const position = {
                    // +2 as Monaco is 1-indexed.
                    column: editor.getValue().length + 2,
                    lineNumber: 1,
                }
                editor.setPosition(position)
                editor.revealPosition(position)
            }
        }
        editor.focus()
    }, [editor, queryState])

    // Prevent newline insertion in model, and surface query changes with stripped newlines.
    useEffect(() => {
        if (!editor) {
            return
        }
        const replacePattern = /[\n\r↵]/g
        const disposable = editor.onDidChangeModelContent(() => {
            const value = editor.getValue()
            onChange({ query: value.replace(replacePattern, ''), changeSource: QueryChangeSource.userInput })
        })
        return () => disposable.dispose()
    }, [editor, onChange])

    // Submit on enter, hiding the suggestions widget if it's visible.
    useEffect(() => {
        if (!editor) {
            return
        }

        if (!acceptSearchSuggestionOnEnter) {
            // Unconditionally trigger the search when pressing `Enter`,
            // including when there are visible completion suggestions.
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
        }

        const run = (): void => {
            onSubmit()
            editor.trigger('submitOnEnter', 'hideSuggestWidget', [])
        }
        const disposables = [
            // Trigger the search with "Enter" on the condition that there are
            // no visible completion suggestions.
            editor.addAction({
                id: 'submitOnEnter',
                label: 'submitOnEnter',
                keybindings: [Monaco.KeyCode.Enter],
                precondition: '!suggestWidgetVisible',
                run,
            }),
            // Unconditionally trigger the search with "Command/Ctrl + Enter",
            // ignoring the visibility of completion suggestions.
            editor.addAction({
                id: 'submitOnCommandEnter',
                label: 'submitOnCommandEnter',
                keybindings: [Monaco.KeyCode.Enter | Monaco.KeyMod.CtrlCmd],
                run,
            }),
        ]
        return () => {
            for (const disposable of disposables) {
                disposable.dispose()
            }
        }
    }, [editor, onSubmit, acceptSearchSuggestionOnEnter])

    const options: Monaco.editor.IStandaloneEditorConstructionOptions = {
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
        hover: { delay: 150 },
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
        // Match our monospace/code style from code.scss
        fontFamily: 'sfmono-regular, consolas, menlo, dejavu sans mono, monospace',
        // Display the cursor as a 1px line.
        cursorStyle: 'line',
        cursorWidth: 1,
    }
    return (
        <div
            ref={setContainer}
            className={classNames('flex-grow-1 flex-shrink-past-contents', className)}
            onFocus={onFocus}
        >
            <MonacoEditor
                id="monaco-query-input"
                language={SOURCEGRAPH_SEARCH}
                value={queryState.query}
                height={17}
                isLightTheme={isLightTheme}
                editorWillMount={noop}
                onEditorCreated={setEditor}
                options={options}
                border={false}
                keyboardShortcutForFocus={KEYBOARD_SHORTCUT_FOCUS_SEARCHBAR}
                className="test-query-input monaco-query-input"
            />
        </div>
    )
}
