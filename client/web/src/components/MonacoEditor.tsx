import classNames from 'classnames'
import * as monaco from 'monaco-editor'
import * as React from 'react'
import { ThemeProps } from '../../../shared/src/theme'
import { Subscription, Subject } from 'rxjs'
import { map, distinctUntilChanged } from 'rxjs/operators'
import { KeyboardShortcut } from '../../../shared/src/keyboardShortcuts'
import { Shortcut } from '@slimsag/react-shortcuts'

const SOURCEGRAPH_LIGHT = 'sourcegraph-light'

const SOURCEGRAPH_DARK = 'sourcegraph-dark'

monaco.editor.defineTheme(SOURCEGRAPH_DARK, {
    base: 'vs-dark',
    inherit: true,
    colors: {
        background: '#0E121B',
        'textLink.activeBackground': '#2a3a51',
        'editor.background': '#0E121B',
        'editor.foreground': '#f2f4f8',
        'editorCursor.foreground': '#ffffff',
        'editorSuggestWidget.background': '#1c2736',
        'editorSuggestWidget.foreground': '#F2F4F8',
        'editorSuggestWidget.highlightForeground': '#569cd6',
        'editorSuggestWidget.selectedBackground': '#2a3a51',
        'list.hoverBackground': '#2a3a51',
        'editorSuggestWidget.border': '#2b3750',
        'editorHoverWidget.background': '#1c2736',
        'editorHoverWidget.foreground': '#F2F4F8',
        'editorHoverWidget.border': '#2b3750',
    },
    rules: [
        { token: 'identifier', foreground: '#f2f4f8' },
        { token: 'keyword', foreground: '#569cd6' },
        { token: 'operator', foreground: '#da77f2' },
        { token: 'comment', foreground: '#ffa94d' },
    ],
})

monaco.editor.defineTheme(SOURCEGRAPH_LIGHT, {
    base: 'vs',
    inherit: true,
    colors: {
        background: '#ffffff',
        'editor.background': '#ffffff',
        'editor.foreground': '#2b3750',
        'editorCursor.foreground': '#2b3750',
        'editorSuggestWidget.background': '#ffffff',
        'editorSuggestWidget.foreground': '#2b3750',
        'editorSuggestWidget.border': '#cad2e2',
        'editorSuggestWidget.highlightForeground': '#268bd2',
        'editorSuggestWidget.selectedBackground': '#f2f4f8',
        'list.hoverBackground': '#f2f4f8',
        'editorHoverWidget.background': '#ffffff',
        'editorHoverWidget.foreground': '#2b3750',
        'editorHoverWidget.border': '#cad2e2',
    },
    rules: [
        { token: 'identifier', foreground: '#2b3750' },
        { token: 'keyword', foreground: '#268bd2' },
        { token: 'operator', foreground: '#ae3ec9' },
        { token: 'comment', foreground: '#d9480f' },
    ],
})

interface Props extends ThemeProps {
    /** The contents of the document. */
    value?: string

    /** The language of the document. */
    language?: string

    /** The DOM element ID to use when rendering the component. Use for a11y, not DOM manipulation. */
    id?: string

    /** The height (in px) of the Monaco editor. */
    height: number

    /** Called when the editor has mounted. */
    editorWillMount: (editor: typeof monaco) => void

    /** Called when a standalone code editor has been created with the given props */
    onEditorCreated?: (editor: monaco.editor.IStandaloneCodeEditor) => void

    /** Options for the editor. */
    options: monaco.editor.IEditorOptions

    /** An optional className to add to the editor. */
    className?: string

    /** Whether to add a border to the Monaco editor. Default: true. */
    border?: boolean

    /** Keyboard shortcut to focus the Monaco editor. */
    keyboardShortcutForFocus?: KeyboardShortcut
}

interface State {}

export class MonacoEditor extends React.PureComponent<Props, State> {
    private subscriptions = new Subscription()

    private componentUpdates = new Subject<Props>()

    private editor: monaco.editor.ICodeEditor | undefined

    private setRef = (element: HTMLElement | null): void => {
        if (!element) {
            return
        }
        this.props.editorWillMount(monaco)
        const editor = monaco.editor.create(element, {
            value: this.props.value,
            language: this.props.language,
            theme: this.props.isLightTheme ? SOURCEGRAPH_LIGHT : SOURCEGRAPH_DARK,
            ...this.props.options,
        })
        if (this.props.onEditorCreated) {
            this.props.onEditorCreated(editor)
        }
        this.editor = editor
    }

    public componentDidUpdate(previousProps: Props): void {
        if (this.props.value !== previousProps.value && this.editor && this.editor.getValue() !== this.props.value) {
            this.editor.setValue(this.props.value || '')
        }
        this.componentUpdates.next(this.props)
    }

    public componentDidMount(): void {
        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    map(({ isLightTheme }) => (isLightTheme ? SOURCEGRAPH_LIGHT : SOURCEGRAPH_DARK)),
                    distinctUntilChanged()
                )
                .subscribe(theme => monaco.editor.setTheme(theme))
        )
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
        if (this.editor) {
            this.editor.dispose()

            // HACK: Clean up ARIA container that Monaco apparently forgets to remove.
            for (const element of document.querySelectorAll('.monaco-aria-container')) {
                element.remove()
            }
        }
    }

    public render(): JSX.Element | null {
        return (
            <>
                <div
                    // eslint-disable-next-line react/forbid-dom-props
                    style={{ height: `${this.props.height}px`, position: 'relative' }}
                    ref={this.setRef}
                    id={this.props.id}
                    className={classNames(this.props.className, this.props.border !== false && 'border')}
                />
                {this.props.keyboardShortcutForFocus?.keybindings.map((keybinding, index) => (
                    <Shortcut key={index} {...keybinding} onMatch={this.focusInput} />
                ))}
            </>
        )
    }

    private focusInput = (): void => {
        if (
            this.editor &&
            !!document.activeElement &&
            !['INPUT', 'TEXTAREA'].includes(document.activeElement.nodeName)
        ) {
            this.editor.focus()
        }
    }
}

if (!window.MonacoEnvironment) {
    window.MonacoEnvironment = {
        getWorkerUrl(moduleId: string, label: string): string {
            if (label === 'json') {
                return window.context.assetsRoot + '/scripts/json.worker.bundle.js'
            }
            return window.context.assetsRoot + '/scripts/editor.worker.bundle.js'
        },
    }
}
