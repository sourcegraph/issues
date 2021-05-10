import { range, isEqual } from 'lodash'
import ErrorIcon from 'mdi-react/ErrorIcon'
import React from 'react'
import VisibilitySensor from 'react-visibility-sensor'
import { combineLatest, Observable, Subject, Subscription } from 'rxjs'
import { catchError, filter, switchMap, map, distinctUntilChanged } from 'rxjs/operators'

import * as GQL from '../graphql/schema'
import { highlightNode } from '../util/dom'
import { asError, ErrorLike, isErrorLike } from '../util/errors'
import { Repo } from '../util/url'

export interface FetchFileParameters {
    repoName: string
    commitID: string
    filePath: string
    disableTimeout?: boolean
    ranges: GQL.IHighlightLineRange[]
}

interface Props extends Repo {
    commitID: string
    filePath: string
    highlightRanges: HighlightRange[]
    /** The 0-based (inclusive) line number that this code excerpt starts at */
    startLine: number
    /** The 0-based (exclusive) line number that this code excerpt ends at */
    endLine: number
    /** Whether or not this is the first result being shown or not. */
    isFirst: boolean
    className?: string
    /** A function to fetch the range of lines this code excerpt will display. It will be provided
     * the same start and end lines properties that were provided as component props */
    fetchHighlightedFileRangeLines: (isFirst: boolean, startLine: number, endLine: number) => Observable<string[]>
}

interface HighlightRange {
    /**
     * The 0-based line number that this highlight appears in
     */
    line: number
    /**
     * The 0-based character offset to start highlighting at
     */
    character: number
    /**
     * The number of characters to highlight
     */
    highlightLength: number
}

interface State {
    blobLinesOrError?: string[] | ErrorLike
}

/**
 * A code excerpt that displays syntax highlighting and match range highlighting.
 */
export class CodeExcerpt extends React.PureComponent<Props, State> {
    public state: State = {}
    private tableContainerElement: HTMLElement | null = null
    private propsChanges = new Subject<Props>()
    private visibilityChanges = new Subject<boolean>()
    private subscriptions = new Subscription()
    private visibilitySensorOffset = { bottom: -500 }

    constructor(props: Props) {
        super(props)
        this.subscriptions.add(
            combineLatest([this.propsChanges, this.visibilityChanges])
                .pipe(
                    filter(([, isVisible]) => isVisible),
                    map(([{ repoName, filePath, commitID, isFirst, startLine, endLine }]) => ({
                        repoName,
                        filePath,
                        commitID,
                        isFirst,
                        startLine,
                        endLine,
                    })),
                    distinctUntilChanged((a, b) => isEqual(a, b)),
                    switchMap(({ repoName, filePath, commitID, isFirst, startLine, endLine }) =>
                        props.fetchHighlightedFileRangeLines(isFirst, startLine, endLine)
                    ),
                    catchError(error => [asError(error)])
                )
                .subscribe(blobLinesOrError => {
                    this.setState({ blobLinesOrError })
                })
        )
    }

    public componentDidMount(): void {
        this.propsChanges.next(this.props)
    }

    public componentDidUpdate(): void {
        this.propsChanges.next(this.props)

        if (this.tableContainerElement) {
            const visibleRows = this.tableContainerElement.querySelectorAll('table tr')
            for (const highlight of this.props.highlightRanges) {
                // Select the HTML row in the excerpt that corresponds to the line to be highlighted.
                // highlight.line is the 0-indexed line number in the code file, and this.props.startLine is the 0-indexed
                // line number of the first visible line in the excerpt. So, subtract this.props.startLine
                // from highlight.line to get the correct 0-based index in visibleRows that holds the HTML row.
                const tableRow = visibleRows[highlight.line - this.props.startLine]
                if (tableRow) {
                    // Take the lastChild of the row to select the code portion of the table row (each table row consists of the line number and code).
                    const code = tableRow.lastChild as HTMLTableDataCellElement
                    highlightNode(code, highlight.character, highlight.highlightLength)
                }
            }
        }
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    private onChangeVisibility = (isVisible: boolean): void => {
        this.visibilityChanges.next(isVisible)
    }

    public render(): JSX.Element | null {
        return (
            <VisibilitySensor
                onChange={this.onChangeVisibility}
                partialVisibility={true}
                offset={this.visibilitySensorOffset}
            >
                <code
                    className={`code-excerpt ${this.props.className || ''}${
                        isErrorLike(this.state.blobLinesOrError) ? ' code-excerpt-error' : ''
                    }`}
                >
                    {this.state.blobLinesOrError && !isErrorLike(this.state.blobLinesOrError) && (
                        <div
                            ref={this.setTableContainerElement}
                            dangerouslySetInnerHTML={{ __html: this.makeTableHTML(this.state.blobLinesOrError) }}
                        />
                    )}
                    {this.state.blobLinesOrError && isErrorLike(this.state.blobLinesOrError) && (
                        <div className="code-excerpt-alert">
                            <ErrorIcon className="icon-inline mr-2" />
                            {this.state.blobLinesOrError.message}
                        </div>
                    )}
                    {!this.state.blobLinesOrError && (
                        <table>
                            <tbody>
                                {range(this.props.startLine, this.props.endLine).map(index => (
                                    <tr key={index}>
                                        <td className="line">{index + 1}</td>
                                        {/* create empty space to fill viewport (as if the blob content were already fetched, otherwise we'll overfetch) */}
                                        <td className="code"> </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    )}
                </code>
            </VisibilitySensor>
        )
    }

    private setTableContainerElement = (reference: HTMLElement | null): void => {
        this.tableContainerElement = reference
    }

    private makeTableHTML(blobLines: string[]): string {
        return '<table>' + blobLines.join('') + '</table>'
    }
}
