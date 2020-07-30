import * as React from 'react'
import VisibilitySensor from 'react-visibility-sensor'
import { LinkOrSpan } from '../../../shared/src/components/LinkOrSpan'
import * as GQL from '../../../shared/src/graphql/schema'
import { highlightNode } from '../../../shared/src/util/dom'
import { HighlightRange } from './SearchResult'

interface Props {
    /**
     * A CSS class name to add to this component's element.
     */
    className?: string

    /**
     * The code string (or array of lines) to display.
     */
    value: string | string[]

    /**
     * The highlights for the lines.
     */
    highlights?: GQL.Highlight[] | HighlightRange[]

    /**
     * A list of classes to apply to 1-indexed line numbers.
     */
    lineClasses?: { line: number; className: string; url?: string }[]

    /**
     * Called when the mousedown event is triggered on the element.
     */
    onMouseDown?: () => void
}

interface DecoratedLine {
    value: string
    highlights?: (GQL.Highlight | HighlightRange)[]
    classNames?: string[]
    url?: string
}

interface State {
    visible: boolean
    lines: DecoratedLine[]
}

/**
 * A simple component for displaying lines of text, with optional
 * highlighted ranges (not syntax highlighting, only e.g. query match
 * highlighting).
 */
export class DecoratedTextLines extends React.PureComponent<Props, State> {
    private tableContainerElement: HTMLElement | null = null

    constructor(props: Props) {
        super(props)
        this.state = {
            ...this.getStateForProps(props),
            visible: false,
        }
    }

    public componentDidMount(): void {
        this.updateHighlights()
    }

    public componentDidUpdate(previousProps: Props): void {
        if (
            this.props.value !== previousProps.value ||
            this.props.highlights !== previousProps.highlights ||
            this.props.lineClasses !== previousProps.lineClasses
        ) {
            // eslint-disable-next-line react/no-did-update-set-state
            this.setState(this.getStateForProps(this.props))
        }
        this.updateHighlights()
    }

    private updateHighlights(): void {
        if (this.state.visible && this.tableContainerElement) {
            const rows = this.tableContainerElement.querySelectorAll('table tr')
            for (const [index, row] of rows.entries()) {
                const element = row.firstChild as HTMLTableDataCellElement
                const data = this.state.lines[index]
                if (data.highlights && data.highlights.length > 0) {
                    // TODO(sqs): only supports 1 highlight per line
                    const highlight = data.highlights[0]
                    highlightNode(element, highlight.character, highlight.length)
                }
            }
        }
    }

    private getStateForProps(props: Props): { lines: DecoratedLine[] } {
        const lineValues = typeof props.value === 'string' ? props.value.split('\n') : props.value
        const lines: DecoratedLine[] = lineValues.map(line => ({ value: line }))
        if (props.highlights) {
            for (const highlight of props.highlights) {
                if (highlight.line > lines.length - 1) {
                    continue
                }
                const line = lines[highlight.line - 1]
                if (!line.highlights) {
                    line.highlights = []
                }
                line.highlights.push(highlight)
            }
        }
        if (props.lineClasses) {
            for (const { line: lineNumber, className, url } of props.lineClasses) {
                const line = lines[lineNumber - 1]
                if (!line.classNames) {
                    line.classNames = []
                }
                line.classNames.push(className)
                if (url) {
                    line.url = url
                }
            }
        }
        return { lines }
    }

    public render(): JSX.Element | null {
        return (
            <VisibilitySensor onChange={this.onChangeVisibility} partialVisibility={true}>
                <code className={`decorated-text-lines code-excerpt ${this.props.className || ''}`}>
                    <table ref={this.setTableContainerElement}>
                        <tbody>
                            {this.state.lines.map((line, index) => (
                                <tr key={index} className={line.classNames ? line.classNames.join(' ') : undefined}>
                                    <td className="code" onMouseDown={this.props.onMouseDown}>
                                        <LinkOrSpan to={line.url}>{line.value}</LinkOrSpan>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </code>
            </VisibilitySensor>
        )
    }

    public onChangeVisibility = (): void => {
        this.setState({ visible: true })
    }

    private setTableContainerElement = (reference: HTMLElement | null): void => {
        this.tableContainerElement = reference
    }
}
