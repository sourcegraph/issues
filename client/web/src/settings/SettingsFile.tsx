import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import * as H from 'history'
import * as _monaco from 'monaco-editor' // type only
import * as React from 'react'
import { Subject, Subscription } from 'rxjs'
import { distinctUntilChanged, filter, map, startWith } from 'rxjs/operators'
import * as GQL from '../../../shared/src/graphql/schema'
import { SaveToolbar } from '../components/SaveToolbar'
import { settingsActions } from '../site-admin/configHelpers'
import { ThemeProps } from '../../../shared/src/theme'
import { eventLogger } from '../tracking/eventLogger'
import { TelemetryProps } from '../../../shared/src/telemetry/telemetryService'

interface Props extends ThemeProps, TelemetryProps {
    history: H.History

    settings: GQL.ISettings | null

    /**
     * JSON Schema of the document.
     */
    jsonSchema?: { $id: string }

    /**
     * Called when the user saves changes to the settings file's contents.
     */
    onDidCommit: (lastID: number | null, contents: string) => void

    /**
     * Called when the user discards changes to the settings file's contents.
     */
    onDidDiscard: () => void

    /**
     * The error that occurred on the last call to the onDidCommit callback,
     * if any.
     */
    commitError?: Error
}

interface State {
    contents?: string
    saving: boolean

    /**
     * The lastID that we started editing from. If null, then no
     * previous versions of the settings exist, and we're creating them from
     * scratch.
     */
    editingLastID?: number | null
}

const emptySettings = '{\n  // add settings here (Ctrl+Space to see hints)\n}'

const disposableToFn = (disposable: _monaco.IDisposable) => () => disposable.dispose()

const MonacoSettingsEditor = React.lazy(async () => ({
    default: (await import('./MonacoSettingsEditor')).MonacoSettingsEditor,
}))

export class SettingsFile extends React.PureComponent<Props, State> {
    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()
    private editor?: _monaco.editor.ICodeEditor
    private monaco: typeof _monaco | null = null

    constructor(props: Props) {
        super(props)

        this.state = { saving: false }

        // Reset state upon navigation to a different subject.
        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    startWith(props),
                    map(({ settings }) => settings),
                    distinctUntilChanged()
                )
                .subscribe(() => {
                    if (this.state.contents !== undefined) {
                        this.setState({ contents: undefined })
                    }
                })
        )

        // Saving ended (in failure) if we get a commitError.
        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    map(({ commitError }) => commitError),
                    distinctUntilChanged(),
                    filter(commitError => !!commitError)
                )
                .subscribe(() => this.setState({ saving: false }))
        )

        // We are finished saving when we receive the new settings ID and it's
        // higher than the one we saved on top of.
        const refreshedAfterSave = this.componentUpdates.pipe(
            filter(({ settings }) => !!settings),
            distinctUntilChanged(
                (a, b) =>
                    (!a.settings && !!b.settings) ||
                    (!!a.settings && !b.settings) ||
                    (!!a.settings &&
                        !!b.settings &&
                        a.settings.contents === b.settings.contents &&
                        a.settings.id === b.settings.id)
            ),
            filter(
                ({ settings, commitError }) =>
                    !!settings &&
                    !commitError &&
                    ((typeof this.state.editingLastID === 'number' && settings.id > this.state.editingLastID) ||
                        (typeof settings.id === 'number' && this.state.editingLastID === null))
            )
        )
        this.subscriptions.add(
            refreshedAfterSave.subscribe(({ settings }) =>
                this.setState({
                    saving: false,
                    editingLastID: undefined,
                    contents: settings ? settings.contents : undefined,
                })
            )
        )
    }

    public componentDidMount(): void {
        // Prevent navigation when dirty.
        this.subscriptions.add(
            this.props.history.block((location: H.Location, action: H.Action) => {
                if (action === 'REPLACE') {
                    return undefined
                }
                if (this.state.saving || this.dirty) {
                    return 'Discard settings changes?'
                }
                return undefined // allow navigation
            })
        )
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    private get dirty(): boolean {
        return this.state.contents !== undefined && this.state.contents !== this.getPropsSettingsContentsOrEmpty()
    }

    public render(): JSX.Element | null {
        const dirty = this.dirty
        const contents =
            this.state.contents === undefined ? this.getPropsSettingsContentsOrEmpty() : this.state.contents

        return (
            <div className="settings-file test-settings-file d-flex flex-grow-1 flex-column">
                <SaveToolbar
                    dirty={dirty}
                    error={this.props.commitError}
                    saving={this.state.saving}
                    onSave={this.save}
                    onDiscard={this.discard}
                />
                <div className="site-admin-configuration-page__action-groups">
                    <div className="site-admin-configuration-page__actions">
                        {settingsActions.map(({ id, label }) => (
                            <button
                                type="button"
                                key={id}
                                className="btn btn-secondary btn-sm site-admin-configuration-page__action"
                                onClick={() => this.runAction(id)}
                            >
                                {label}
                            </button>
                        ))}
                    </div>
                </div>
                <React.Suspense fallback={<LoadingSpinner className="icon-inline mt-2" />}>
                    <MonacoSettingsEditor
                        value={contents}
                        jsonSchema={this.props.jsonSchema}
                        onChange={this.onEditorChange}
                        readOnly={this.state.saving}
                        monacoRef={this.monacoRef}
                        isLightTheme={this.props.isLightTheme}
                        onDidSave={this.save}
                    />
                </React.Suspense>
            </div>
        )
    }

    private monacoRef = (monacoValue: typeof _monaco | null): void => {
        this.monaco = monacoValue
        if (this.monaco) {
            this.subscriptions.add(
                disposableToFn(
                    this.monaco.editor.onDidCreateEditor(editor => {
                        this.editor = editor
                    })
                )
            )
            this.subscriptions.add(
                disposableToFn(
                    this.monaco.editor.onDidCreateModel(async model => {
                        // This function can only be called if the lazy MonacoSettingsEditor component was loaded,
                        // so this import call will not incur another load.
                        const { MonacoSettingsEditor } = await import('./MonacoSettingsEditor')

                        if (this.editor && MonacoSettingsEditor.isStandaloneCodeEditor(this.editor)) {
                            for (const { id, label, run } of settingsActions) {
                                MonacoSettingsEditor.addEditorAction(
                                    this.editor,
                                    model,
                                    label,
                                    id,
                                    run,
                                    this.props.telemetryService
                                )
                            }
                        }
                    })
                )
            )
        }
    }

    private runAction(id: string): void {
        if (this.editor) {
            const action = this.editor.getAction(id)
            action.run().then(
                () => undefined,
                error => console.error(error)
            )
        } else {
            alert('Wait for editor to load before running action.')
        }
    }

    private getPropsSettingsContentsOrEmpty(settings = this.props.settings): string {
        return settings ? settings.contents : emptySettings
    }

    private getPropsSettingsID(): number | null {
        return this.props.settings ? this.props.settings.id : null
    }

    private discard = (): void => {
        if (
            this.getPropsSettingsContentsOrEmpty() === this.state.contents ||
            window.confirm('Discard settings edits?')
        ) {
            eventLogger.log('SettingsFileDiscard')
            this.setState({
                contents: undefined,
                editingLastID: undefined,
            })
            this.props.onDidDiscard()
        } else {
            eventLogger.log('SettingsFileDiscardCanceled')
        }
    }

    private onEditorChange = (newValue: string): void => {
        if (newValue !== this.getPropsSettingsContentsOrEmpty()) {
            this.setState({ editingLastID: this.getPropsSettingsID() })
        }
        this.setState({ contents: newValue })
    }

    private save = (): void => {
        eventLogger.log('SettingsFileSaved')
        this.setState({ saving: true }, () => {
            this.props.onDidCommit(this.getPropsSettingsID(), this.state.contents!)
        })
    }
}
