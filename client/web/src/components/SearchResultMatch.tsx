import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { decode } from 'he'
import { escapeRegExp, isEqual, range } from 'lodash'
import React from 'react'
import { Link } from 'react-router-dom'
import VisibilitySensor from 'react-visibility-sensor'
import { combineLatest, of, Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, filter, switchMap } from 'rxjs/operators'
import sanitizeHtml from 'sanitize-html'
import { Markdown } from '../../../shared/src/components/Markdown'
import * as GQL from '../../../shared/src/graphql/schema'
import { highlightNode } from '../../../shared/src/util/dom'
import { highlightCode } from '../search/backend'
import { ThemeProps } from '../../../shared/src/theme'
import * as H from 'history'
import { renderMarkdown } from '../../../shared/src/util/markdown'

interface SearchResultMatchProps extends ThemeProps {
    item: GQL.ISearchResultMatch
    highlightRanges: GQL.IHighlight[]
    history: H.History
}

interface SearchResultMatchState {
    HTML?: string
}

export class SearchResultMatch extends React.Component<SearchResultMatchProps, SearchResultMatchState> {
    public state: SearchResultMatchState = {}
    private tableContainerElement: HTMLElement | null = null
    private visibilitySensorOffset = { bottom: -500 }

    private visibilityChanges = new Subject<boolean>()
    private subscriptions = new Subscription()
    private propsChanges = new Subject<SearchResultMatchProps>()

    private getLanguage(): string | undefined {
        const matches = /```(\S+)\s/.exec(this.props.item.body.text)
        if (!matches) {
            return undefined
        }
        return matches[1]
    }

    private bodyIsCode(): boolean {
        return this.props.item.body.text.startsWith('```') && this.props.item.body.text.endsWith('```')
    }

    constructor(props: SearchResultMatchProps) {
        super(props)
        // Render the match body as markdown, and syntax highlight the response if it's a code block.
        // This is a lot of network requests right now, but once extensions can run on the backend we can
        // run results through the renderer and syntax highlighter without network requests.
        this.subscriptions.add(
            combineLatest([this.propsChanges, this.visibilityChanges])
                .pipe(
                    filter(([, isVisible]) => isVisible),
                    distinctUntilChanged((a, b) => isEqual(a, b)),
                    switchMap(([props]) => {
                        const markdownHTML = props.item.body.html
                            ? sanitizeHtml(props.item.body.html)
                            : renderMarkdown(props.item.body.text)
                        if (this.bodyIsCode()) {
                            const lang = this.getLanguage() || 'txt'
                            const parser = new DOMParser()
                            // Extract the text content of the result.
                            const codeContent =
                                parser.parseFromString(markdownHTML, 'text/html').body.textContent?.trim() || ''
                            // Match the code content and any trailing newlines if any.
                            const codeContentAndAnyNewLines = new RegExp(escapeRegExp(codeContent) + '\\n*')
                            if (codeContent) {
                                return highlightCode({
                                    code: codeContent,
                                    fuzzyLanguage: lang,
                                    disableTimeout: false,
                                    isLightTheme: props.isLightTheme,
                                }).pipe(
                                    switchMap(highlightedString => {
                                        const highlightedMarkdown = decode(markdownHTML).replace(
                                            codeContentAndAnyNewLines,
                                            highlightedString
                                        )
                                        return of(highlightedMarkdown)
                                    }),
                                    // Return the rendered markdown if highlighting fails.
                                    catchError(() => of(markdownHTML))
                                )
                            }
                        }
                        return of(markdownHTML)
                    }),
                    // Return the raw body if markdown rendering fails, maintaining the text structure.
                    catchError(() => of('<pre>' + sanitizeHtml(props.item.body.text) + '</pre>'))
                )
                .subscribe(
                    string => this.setState({ HTML: string }),
                    error => console.error(error)
                )
        )
    }

    public componentDidMount(): void {
        this.propsChanges.next(this.props)
        this.highlightNodes()
    }

    public componentDidUpdate(): void {
        this.propsChanges.next(this.props)
        this.highlightNodes()
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    private highlightNodes(): void {
        if (this.tableContainerElement) {
            const visibleRows = this.tableContainerElement.querySelectorAll('table tr')
            if (visibleRows.length > 0) {
                for (const range of this.props.highlightRanges) {
                    const code = visibleRows[range.line - 1]
                    if (code) {
                        highlightNode(code as HTMLElement, range.character, range.length)
                    }
                }
            }
        }
    }

    private onChangeVisibility = (isVisible: boolean): void => {
        this.visibilityChanges.next(isVisible)
    }

    private getFirstLine(): number {
        if (this.props.highlightRanges.length === 0) {
            // If there are no highlights, the calculation below results in -Infinity.
            return 0
        }
        return Math.max(0, Math.min(...this.props.highlightRanges.map(range => range.line)) - 1)
    }

    private getLastLine(): number {
        if (this.props.highlightRanges.length === 0) {
            // If there are no highlights, the calculation below results in Infinity,
            // so we set lastLine to 5, which is a just a heuristic for a medium-sized result.
            return 5
        }
        const lastLine = Math.max(...this.props.highlightRanges.map(range => range.line)) + 1
        return this.props.highlightRanges ? Math.min(lastLine, this.props.highlightRanges.length) : lastLine
    }

    public render(): JSX.Element {
        const firstLine = this.getFirstLine()
        let lastLine = this.getLastLine()
        if (firstLine === lastLine) {
            // Some edge cases yield the same first and last line, causing the visibility sensor to break, so make sure to avoid this.
            lastLine++
        }

        return (
            <VisibilitySensor
                active={true}
                onChange={this.onChangeVisibility}
                partialVisibility={true}
                offset={this.visibilitySensorOffset}
            >
                <>
                    {this.state.HTML !== undefined ? (
                        <Link key={this.props.item.url} to={this.props.item.url} className="search-result-match">
                            {this.bodyIsCode() ? (
                                <code>
                                    <Markdown
                                        refFn={this.setTableContainerElement}
                                        className="search-result-match__markdown search-result-match__code-excerpt"
                                        dangerousInnerHTML={this.state.HTML}
                                        history={this.props.history}
                                    />
                                </code>
                            ) : (
                                <Markdown
                                    refFn={this.setTableContainerElement}
                                    className="search-result-match__markdown"
                                    dangerousInnerHTML={this.state.HTML}
                                    history={this.props.history}
                                />
                            )}
                        </Link>
                    ) : (
                        <>
                            <LoadingSpinner className="icon-inline search-result-match__loader" />
                            <table>
                                <tbody>
                                    {range(firstLine, lastLine).map(index => (
                                        <tr key={`${this.props.item.url}#${index}`}>
                                            {/* create empty space to fill viewport (as if the blob content were already fetched, otherwise we'll overfetch) */}
                                            <td className="line search-result-match__line--hidden">
                                                <code>{index}</code>
                                            </td>
                                            <td className="code"> </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </>
                    )}
                </>
            </VisibilitySensor>
        )
    }

    private setTableContainerElement = (reference: HTMLElement | null): void => {
        this.tableContainerElement = reference
    }
}
