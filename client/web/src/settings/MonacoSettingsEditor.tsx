import * as jsonc from '@sqs/jsonc-parser'
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api'
import * as React from 'react'
import { Subject, Subscription } from 'rxjs'
import { distinctUntilChanged, distinctUntilKeyChanged, map, startWith } from 'rxjs/operators'
import jsonSchemaMetaSchema from '../../../../schema/json-schema-draft-07.schema.json'
import settingsSchema from '../../../../schema/settings.schema.json'
import { MonacoEditor } from '../components/MonacoEditor'
import { ThemeProps } from '../../../shared/src/theme'
import { TelemetryService } from '../../../shared/src/telemetry/telemetryService'

/**
 * Minimal shape of a JSON Schema. These values are treated as opaque, so more specific types are
 * not needed.
 */
interface JSONSchema {
    $id: string
}

export interface Props extends ThemeProps {
    id?: string
    className?: string
    value: string | undefined
    onChange?: (newValue: string) => void
    readOnly?: boolean | undefined
    height?: number

    language?: string

    /**
     * JSON Schema of the document.
     */
    jsonSchema?: JSONSchema

    monacoRef?: (monacoValue: typeof monaco | null) => void
    /**
     * Called when the user presses the key binding for "save" (Ctrl+S/Cmd+S).
     */
    onDidSave?: () => void
}

interface State {}

/**
 * A JSON settings editor using the Monaco editor.
 */
export class MonacoSettingsEditor extends React.PureComponent<Props, State> {
    public state: State = {}

    private monaco: typeof monaco | null = null
    private editor: monaco.editor.ICodeEditor | null = null

    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()
    private disposables: monaco.IDisposable[] = []

    public componentDidMount(): void {
        const componentUpdates = this.componentUpdates.pipe(startWith(this.props))

        this.subscriptions.add(
            componentUpdates
                .pipe(
                    map(props => props.readOnly),
                    distinctUntilChanged()
                )
                .subscribe(readOnly => {
                    if (this.editor) {
                        this.editor.updateOptions({ readOnly })
                    }
                })
        )

        this.subscriptions.add(
            componentUpdates.pipe(distinctUntilKeyChanged('jsonSchema')).subscribe(props => {
                if (this.monaco) {
                    setDiagnosticsOptions(this.monaco, props.jsonSchema)
                }
            })
        )
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
        for (const disposable of this.disposables) {
            disposable.dispose()
        }
        this.monaco = null
        this.editor = null
    }

    public render(): JSX.Element | null {
        return (
            <MonacoEditor
                id={this.props.id}
                className={this.props.className}
                language={this.props.language || 'json'}
                height={this.props.height || 400}
                isLightTheme={this.props.isLightTheme}
                value={this.props.value}
                editorWillMount={this.editorWillMount}
                options={{
                    lineNumbers: 'off',
                    automaticLayout: true,
                    minimap: { enabled: false },
                    formatOnType: true,
                    formatOnPaste: true,
                    autoIndent: true,
                    renderIndentGuides: false,
                    glyphMargin: false,
                    folding: false,
                    renderLineHighlight: 'none',
                    scrollBeyondLastLine: false,
                    quickSuggestions: true,
                    quickSuggestionsDelay: 200,
                    wordBasedSuggestions: false,
                    readOnly: this.props.readOnly,
                    wordWrap: 'on',
                }}
            />
        )
    }

    private editorWillMount = (editor: typeof monaco): void => {
        this.monaco = editor
        if (editor) {
            this.onDidEditorMount()
        }
    }

    private onDidEditorMount(): void {
        const monaco = this.monaco!

        if (this.props.monacoRef) {
            this.props.monacoRef(monaco)
            this.subscriptions.add(() => {
                if (this.props.monacoRef) {
                    this.props.monacoRef(null)
                }
            })
        }

        setDiagnosticsOptions(monaco, this.props)

        // Only listen to 1 event each to avoid receiving events from other Monaco editors on the
        // same page (if there are multiple).
        const editorDisposable = monaco.editor.onDidCreateEditor(editor => {
            this.onDidCreateEditor(editor)
            editorDisposable.dispose()
        })
        this.disposables.push(editorDisposable)
        const modelDisposable = monaco.editor.onDidCreateModel(model => {
            this.onDidCreateModel(model)
            modelDisposable.dispose()
        })
        this.disposables.push(modelDisposable)
    }

    private onDidCreateEditor(editor: monaco.editor.ICodeEditor): void {
        this.editor = editor

        // Necessary to wrap in setTimeout or else _standaloneKeyBindingService won't be ready and the editor will
        // refuse to add the command because it's missing the keybinding service.
        setTimeout(() => {
            if (MonacoSettingsEditor.isStandaloneCodeEditor(editor)) {
                editor.addCommand(
                    monaco.KeyMod.CtrlCmd | monaco.KeyCode.KEY_S,
                    () => {
                        if (this.props.onDidSave) {
                            this.props.onDidSave()
                        }
                    },
                    ''
                )
            }
        })
    }

    private onDidCreateModel(model: monaco.editor.IModel): void {
        this.disposables.push(
            model.onDidChangeContent(() => {
                if (this.props.onChange) {
                    this.props.onChange(model.getValue())
                }
            })
        )
    }

    public static isStandaloneCodeEditor(
        editor: monaco.editor.ICodeEditor
    ): editor is monaco.editor.IStandaloneCodeEditor {
        const maybeStandaloneEditor: Partial<monaco.editor.IStandaloneCodeEditor> = editor
        return (
            editor.getEditorType() === monaco.editor.EditorType.ICodeEditor &&
            typeof maybeStandaloneEditor.addAction === 'function'
        )
    }

    public static addEditorAction(
        inputEditor: monaco.editor.IStandaloneCodeEditor,
        model: monaco.editor.IModel,
        label: string,
        id: string,
        run: ConfigInsertionFunction,
        telemetryService: TelemetryService
    ): void {
        inputEditor.addAction({
            label,
            id,
            run: editor => {
                telemetryService.log('SiteConfigurationActionExecuted')
                editor.focus()
                editor.pushUndoStop()
                const { edits, selectText, cursorOffset } = run(editor.getValue())
                const monacoEdits = toMonacoEdits(model, edits)
                let selection: monaco.Selection | undefined
                if (typeof selectText === 'string') {
                    const afterText = jsonc.applyEdits(editor.getValue(), edits)
                    let offset = afterText.slice(edits[0].offset).indexOf(selectText)
                    if (offset !== -1) {
                        offset += edits[0].offset
                        if (typeof cursorOffset === 'number') {
                            selection = monaco.Selection.fromPositions(
                                getPositionAt(afterText, offset + cursorOffset),
                                getPositionAt(afterText, offset + cursorOffset)
                            )
                        } else {
                            selection = monaco.Selection.fromPositions(
                                getPositionAt(afterText, offset),
                                getPositionAt(afterText, offset + selectText.length)
                            )
                        }
                    }
                }
                if (!selection) {
                    // TODO: This is buggy. See
                    // https://github.com/sourcegraph/sourcegraph/issues/2756.
                    selection = monaco.Selection.fromPositions(
                        monacoEdits[0].range.getStartPosition(),
                        monacoEdits[monacoEdits.length - 1].range.getEndPosition()
                    )
                }
                editor.executeEdits(id, monacoEdits, [selection])
                editor.revealPositionInCenter(selection.getStartPosition())
            },
        })
    }
}

function setDiagnosticsOptions(editor: typeof monaco, jsonSchema: any): void {
    editor.languages.json.jsonDefaults.setDiagnosticsOptions({
        validate: true,
        allowComments: true,
        schemas: [
            {
                uri: 'root#', // doesn't matter as long as it doesn't collide
                schema: jsonSchema,
                fileMatch: ['*'],
            },

            // Include these schemas because they are referenced by other schemas.
            {
                uri: 'http://json-schema.org/draft-07/schema',
                schema: jsonSchemaMetaSchema as JSONSchema,
            },
            {
                uri: 'settings.schema.json#',
                schema: settingsSchema,
            },
            {
                uri: 'settings.schema.json',
                schema: settingsSchema,
            },
        ],
    })
}

function toMonacoEdits(
    model: monaco.editor.IModel,
    edits: jsonc.Edit[]
): monaco.editor.IIdentifiedSingleEditOperation[] {
    return edits.map((edit, index) => ({
        identifier: { major: model.getVersionId(), minor: index },
        range: monaco.Range.fromPositions(
            model.getPositionAt(edit.offset),
            model.getPositionAt(edit.offset + edit.length)
        ),
        forceMoveMarkers: true,
        text: edit.content,
    }))
}

/**
 * A helper function that modifies site configuration to configure specific
 * common things, such as syncing GitHub repositories.
 */
export type ConfigInsertionFunction = (
    configJSON: string
) => {
    /** The edits to make to the input configuration to insert the new configuration. */
    edits: jsonc.Edit[]

    /** Select text in inserted JSON. */
    selectText?: string

    /**
     * If set, the selection is an empty selection that begins at the left-hand match of selectText plus this
     * offset. For example, if selectText is "foo" and cursorOffset is 2, then the final selection will be a cursor
     * "|" positioned as "fo|o".
     */
    cursorOffset?: number
}

function getPositionAt(text: string, offset: number): monaco.IPosition {
    const lines = text.split('\n')
    let position = 0
    for (const [index, line] of lines.entries()) {
        if (offset < position + line.length + 1) {
            return new monaco.Position(index + 1, offset - position + 1)
        }
        position += line.length + 1
    }
    throw new Error(`offset ${offset} out of bounds in text of length ${text.length}`)
}
