import * as H from 'history'
import isEqual from 'lodash/isEqual'
import omit from 'lodash/omit'
import * as React from 'react'
import 'rxjs/add/observable/fromEvent'
import 'rxjs/add/observable/fromPromise'
import 'rxjs/add/observable/interval'
import 'rxjs/add/observable/merge'
import 'rxjs/add/operator/catch'
import 'rxjs/add/operator/do'
import 'rxjs/add/operator/filter'
import 'rxjs/add/operator/map'
import 'rxjs/add/operator/switchMap'
import 'rxjs/add/operator/take'
import 'rxjs/add/operator/takeUntil'
import 'rxjs/add/operator/zip'
import { Observable } from 'rxjs/Observable'
import { Subject } from 'rxjs/Subject'
import { Subscription } from 'rxjs/Subscription'
import { fetchHover, fetchJumpURL, isEmptyHover } from '../backend/lsp'
import { triggerBlame } from '../blame'
import { events } from '../tracking/events'
import { getPathExtension, supportedExtensions } from '../util'
import { parseHash, toAbsoluteBlobURL, toPrettyBlobURL } from '../util/url'
import { AbsoluteRepoFile, AbsoluteRepoFilePosition, getCodeCell } from './index'
import { convertNode, createTooltips, findElementWithOffset, getTableDataCell, getTargetLineAndOffset, hideTooltip, TooltipData, updateTooltip } from './tooltips'

/**
 * Highlights a <td> element and updates the page URL if necessary.
 */
function updateLine(cell: HTMLElement, history: H.History, ctx: AbsoluteRepoFilePosition, clickEvent?: MouseEvent): void {
    triggerBlame(ctx, clickEvent)

    const currentlyHighlighted = document.querySelectorAll('.sg-highlighted') as NodeListOf<HTMLElement>
    for (const cellElem of currentlyHighlighted) {
        cellElem.classList.remove('sg-highlighted')
        cellElem.style.backgroundColor = 'inherit'
    }

    cell.style.backgroundColor = 'rgb(34, 44, 58)'
    cell.classList.add('sg-highlighted')

    // Check URL change first, since this function can be called in response to
    // onhashchange.
    const newUrl = toPrettyBlobURL(ctx)
    if (newUrl === (window.location.pathname + window.location.hash)) {
        // Avoid double-pushing the same URL
        return
    }

    history.push(toPrettyBlobURL(ctx))
}

/**
 * The same as updateLine, but also scrolls the blob.l
 */
function updateAndScrollToLine(cell: HTMLElement, history: H.History, ctx: AbsoluteRepoFilePosition, clickEvent?: MouseEvent): void {
    updateLine(cell, history, ctx, clickEvent)

    // Scroll to the line.
    scrollToCell(cell)
}

function scrollToCell(cell: HTMLElement): void {
    // Scroll to the line.
    const scrollingElement = document.querySelector('.blob')!
    const viewportBound = scrollingElement.getBoundingClientRect()
    const blobTable = document.querySelector('.blob > table')! // table that we're positioning tooltips relative to.
    const tableBound = blobTable.getBoundingClientRect() // tables bounds
    const targetBound = cell.getBoundingClientRect() // our target elements bounds

    scrollingElement.scrollTop = targetBound.top - tableBound.top - (viewportBound.height / 2) + (targetBound.height / 2)
}

interface Props extends AbsoluteRepoFile {
    html: string
    location: H.Location
    history: H.History
}

interface State {
    fixedTooltip?: TooltipData
}

export class Blob extends React.Component<Props, State> {
    public state: State = {}
    private blobElement: HTMLElement | null = null
    private fixedTooltip = new Subject<Props>()
    private subscriptions = new Subscription()

    public componentWillReceiveProps(nextProps: Props): void {
        const hash = parseHash(nextProps.location.hash)
        if (this.props.location.pathname !== nextProps.location.pathname && !hash.line) {
            if (this.blobElement) {
                this.blobElement.scrollTop = 0
            }
        }
        const thisHash = parseHash(this.props.location.hash)
        const nextHash = parseHash(nextProps.location.hash)
        if (thisHash.modal !== nextHash.modal && this.props.location.pathname === nextProps.location.pathname) {
            // When updating references mode in the same file, scroll. Wait just a moment to make sure the references
            // panel is shown, since the scroll offset is calculated based on the height of the blob.
            setTimeout(() => this.scrollToLine(nextProps), 10)
        }

        if (this.props.location.pathname === nextProps.location.pathname && (
            thisHash.character !== nextHash.character ||
            thisHash.line !== nextHash.line ||
            thisHash.modal !== nextHash.modal
        )) {
            if (!nextHash.modal) {
                this.fixedTooltip.next(nextProps)
            } else {
                // If showing modal, remove any tooltip then highlight the element for the given position.
                this.setFixedTooltip()
                if (nextHash.line && nextHash.character) {
                    const el = findElementWithOffset(getCodeCell(nextHash.line!).childNodes[1]! as HTMLElement, nextHash.character!)
                    if (el) {
                        el.classList.add('selection-highlight-sticky')
                    }
                }
            }
        }

        if (this.props.html !== nextProps.html) {
            // Hide the previous tooltip, if it exists.
            hideTooltip()

            this.subscriptions.unsubscribe()
            this.subscriptions = new Subscription()
            if (this.blobElement) {
                this.addTooltipEventListeners(this.blobElement)
            }
            this.setFixedTooltip()
        }
    }

    public shouldComponentUpdate(nextProps: Props): boolean {
        // Only update the blob if the inner HTML content changes.
        if (this.props.html !== nextProps.html) {
            return true
        }

        if (isEqual(omit(this.props, 'rev'), omit(nextProps, 'rev'))) {
            // nextProps is a new location, but we don't have new HTML.
            // We *only* want lifeycle hooks when the html is changed.
            // This prevents e.g. scrolling to a line that doesn't exist
            // yet when file has changed but html hasn't been resolved.
            return false
        }

        const prevHash = parseHash(this.props.location.hash)
        const nextHash = parseHash(nextProps.location.hash)
        if (prevHash.line !== nextHash.line && nextProps.history.action === 'POP') {
            // If we don't need an update (the file hasn't changed, and we will *not* get into componentDidUpdate).
            // We still want to scroll if the hash is changed, but only on 'back' and 'forward' browser events (and not e.g. on each line click).
            this.scrollToLine(nextProps)
        }
        return false
    }

    public componentDidUpdate(prevProps: Props, prevState: State): void {
        hideTooltip()
        createTooltips()

        const parsedHash = parseHash(this.props.location.hash)
        if (!parsedHash.modal) {
            // Show fixed tooltip if necessary iff not showing a modal.
            this.fixedTooltip.next(this.props)
        }
        if (this.props.history.action === 'POP') {
            // The contents were updated on a mounted component and we did a 'back' or 'forward' event;
            // scroll to the appropariate line after the new table is created.
            this.scrollToLine(this.props)
        } else if (this.props.location.state && this.props.location.state.referencesClick) {
            // We do not want to scroll on all 'PUSH' events (otherwise every time a user clicks
            // a line the page would scroll). However, to allow some <Link>'s in external components
            // to trigger a scroll, we let those <Link> components set push state. The references
            // panel results are one such example, and set the `referencesClick` value on location state.
            this.scrollToLine(this.props)
        }
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return (
            <div className='blob' ref={this.onBlobRef} dangerouslySetInnerHTML={{ __html: this.props.html }} />
        )
    }

    private onBlobRef = (ref: HTMLElement | null) => {
        this.blobElement = ref
        if (ref) {
            // This is the first time the component is ever mounted. We need to set initial scroll.
            this.scrollToLine(this.props)
            createTooltips()
            if (supportedExtensions.has(getPathExtension(this.props.filePath))) {
                this.addTooltipEventListeners(ref)
            }
            const parsedHash = parseHash(this.props.location.hash)
            if (parsedHash.line && parsedHash.character) {
                this.fixedTooltip.next(this.props)
            }
        }
    }

    private addTooltipEventListeners = (ref: HTMLElement): void => {
        this.subscriptions.add(
            this.fixedTooltip
                .filter(props => {
                    const parsed = parseHash(props.location.hash)
                    if (parsed.line && parsed.character) {
                        const td = getCodeCell(parsed.line).childNodes[1] as HTMLTableDataCellElement
                        if (td && !td.classList.contains('annotated')) {
                            td.classList.add('annotated')
                            convertNode(td)
                        }
                        if (!parsed.modal) {
                            return true
                        }
                        // Don't show a tooltip when there is a modal (but do highlight the token)
                        // TODO(john): this can probably be simplified.
                        const el = findElementWithOffset(getCodeCell(parsed.line!).childNodes[1]! as HTMLElement, parsed.character!)
                        if (el) {
                            el.classList.add('selection-highlight-sticky')
                            return false
                        }
                    }
                    this.setFixedTooltip()
                    return false
                })
                .map(props => parseHash(props.location.hash))
                .map(pos => findElementWithOffset(getCodeCell(pos.line!).childNodes[1]! as HTMLElement, pos.character!))
                .filter(el => !!el)
                .map((target: HTMLElement) => {
                    const data = { target, loc: getTargetLineAndOffset(target!, false) }
                    if (!data.loc) {
                        return null
                    }
                    const ctx = { ...this.props, position: data.loc! }
                    return { target: data.target, ctx }
                })
                .switchMap(data => {
                    if (data === null) {
                        return [null]
                    }
                    const { target, ctx } = data
                    return this.getTooltip(target, ctx)
                        .zip(this.getDefinition(ctx))
                        .map(([tooltip, defUrl]) => ({ ...tooltip, defUrl: defUrl || undefined } as TooltipData))
                        .catch(e => {
                            const data: TooltipData = { target, ctx }
                            return [data]
                        })
                })
                .subscribe(data => {
                    if (!data) {
                        this.setFixedTooltip()
                        return
                    }

                    const contents = data.contents
                    if (!contents || isEmptyHover({ contents })) {
                        this.setFixedTooltip()
                        return
                    }

                    this.setFixedTooltip(data)
                    updateTooltip(data, true, this.tooltipActions(data.ctx))
                })
        )

        this.subscriptions.add(
            Observable.fromEvent<MouseEvent>(ref, 'mouseover')
                .map(e => e.target as HTMLElement)
                .do(target => {
                    const td = getTableDataCell(target)
                    if (td && !td.classList.contains('annotated')) {
                        td.classList.add('annotated')
                        convertNode(td)
                    }
                })
                .map(target => ({ target, loc: getTargetLineAndOffset(target, false) }))
                .filter(data => Boolean(data.loc))
                .map(data => ({ target: data.target, ctx: { ...this.props, position: data.loc! } }))
                .switchMap(({ target, ctx }) => {
                    const tooltip = this.getTooltip(target, ctx)
                    const tooltipWithJ2D: Observable<TooltipData> = tooltip.zip(this.getDefinition(ctx))
                        .map(([tooltip, defUrl]) => ({ ...tooltip, defUrl: defUrl || undefined }))
                    const loading = this.getLoadingTooltip(target, ctx, tooltip)
                    return Observable.merge(loading, tooltip, tooltipWithJ2D).catch(e => {
                        const data: TooltipData = { target, ctx }
                        return [data]
                    })
                })
                .subscribe(data => {
                    if (!this.state.fixedTooltip) {
                        updateTooltip(data, false, this.tooltipActions(data.ctx))
                    }
                })
        )
        this.subscriptions.add(
            Observable.fromEvent<MouseEvent>(ref, 'mouseout')
                .subscribe(e => {
                    for (const el of document.querySelectorAll('.blob .selection-highlight')) {
                        el.classList.remove('selection-highlight')
                    }
                    if (!this.state.fixedTooltip) {
                        hideTooltip()
                    }
                })
        )
        this.subscriptions.add(
            Observable.fromEvent<MouseEvent>(ref, 'click')
                .filter(e => {
                    const target = e.target as HTMLElement
                    if (!target) {
                        return false
                    }
                    const tooltip = document.querySelector('.sg-tooltip')
                    if (tooltip && tooltip.contains(target)) {
                        return false
                    }
                    return true
                })
                .subscribe(e => {
                    const target = e.target as HTMLElement
                    const row = (target as Element).closest('tr') as HTMLTableRowElement | null
                    if (!row) {
                        return
                    }
                    const line = parseInt(row.firstElementChild!.getAttribute('data-line')!, 10)
                    const data = { target, loc: getTargetLineAndOffset(target, false) }
                    if (!data.loc) {
                        return updateLine(row, this.props.history, {
                            repoPath: this.props.repoPath,
                            rev: this.props.rev,
                            commitID: this.props.commitID,
                            filePath: this.props.filePath,
                            position: { line, character: 0 }
                        }, e)
                    }
                    const ctx = { ...this.props, position: { line, character: data.loc!.character } }
                    updateLine(row, this.props.history, ctx, e)
                })
        )
    }

    private setFixedTooltip = (data?: TooltipData) => {
        for (const el of document.querySelectorAll('.blob .selection-highlight')) {
            el.classList.remove('selection-highlight')
        }
        for (const el of document.querySelectorAll('.blob .selection-highlight-sticky')) {
            el.classList.remove('selection-highlight-sticky')
        }
        if (data) {
            events.TooltipDocked.log()
            data.target.classList.add('selection-highlight-sticky')
        } else {
            hideTooltip()
        }
        this.setState({ fixedTooltip: data || undefined })
    }

    private scrollToLine = (props: Props) => {
        const parsed = parseHash(props.location.hash)
        const { line, character, modalMode } = parsed
        if (line) {
            updateAndScrollToLine(getCodeCell(line), props.history, {
                repoPath: props.repoPath,
                rev: props.rev,
                commitID: props.commitID,
                filePath: props.filePath,
                position: { line, character: character || 0 },
                referencesMode: modalMode
            })
        }
    }

    /**
     * getTooltip wraps the asynchronous fetch of tooltip data from the Sourcegraph API.
     * This Observable will emit exactly one value before it completes. If the resolved
     * tooltip is defined, it will update the target styling.
     */
    private getTooltip(target: HTMLElement, ctx: AbsoluteRepoFilePosition): Observable<TooltipData> {
        return Observable.fromPromise(fetchHover(ctx))
            .do(data => {
                if (isEmptyHover(data)) {
                    // short-cirtuit, no tooltip data
                    return
                }
                target.style.cursor = 'pointer'
                target.classList.add('selection-highlight')
            })
            .map(data => ({ target, ctx, ...data }))
    }
    /**
     * getDefinition wraps the asynchronous fetch of tooltip data from the Sourcegraph API.
     * This Observable will emit exactly one value before it completes.
     */
    private getDefinition(ctx: AbsoluteRepoFilePosition): Observable<string | null> {
        return Observable.fromPromise(fetchJumpURL(ctx))
    }

    /**
     * getLoadingTooltip emits "loading" tooltip data after a delay,
     * iff the other Observable hasn't already emitted a value.
     */
    private getLoadingTooltip(target: HTMLElement, ctx: AbsoluteRepoFilePosition, tooltip: Observable<TooltipData>): Observable<TooltipData> {
        return Observable.interval(500)
            .take(1)
            .takeUntil(tooltip)
            .map(() => ({ target, ctx, loading: true }))
    }

    private handleGoToDefinition = (defCtx: AbsoluteRepoFilePosition) =>
        (e: MouseEvent) => {
            events.GoToDefClicked.log()
            if (e.shiftKey || e.ctrlKey || e.altKey || e.metaKey) {
                return
            }
            e.preventDefault()
            const lastHash = parseHash(this.props.location.hash)
            hideTooltip()
            if (defCtx.position.line && this.props.repoPath === defCtx.repoPath && this.props.rev === defCtx.rev && lastHash.line !== defCtx.position.line) {
                // Handles URL update + scroll to file (for j2d within same file).
                // Since the defCtx rev/commitID may be undefined, use the resolved rev
                // for the current file.
                const ctx = { ...defCtx, commitID: this.props.commitID }
                updateAndScrollToLine(getCodeCell(ctx.position.line), this.props.history, ctx)
            } else {
                this.setFixedTooltip()
                this.props.history.push(toAbsoluteBlobURL(defCtx))
            }
        }

    private handleFindReferences = (ctx: AbsoluteRepoFilePosition) =>
        (e: MouseEvent) => {
            events.FindRefsClicked.log()
            if (e.shiftKey || e.ctrlKey || e.altKey || e.metaKey) {
                return
            }
            e.preventDefault()
            this.props.history.push(toPrettyBlobURL({ ...ctx, rev: this.props.rev, referencesMode: 'local' }))
            hideTooltip()
            scrollToCell(getCodeCell(ctx.position.line))
        }

    private handleDismiss = () => {
        const parsed = parseHash(this.props.location.hash)
        if (parsed.line) {
            // Remove the character position so the fixed tooltip goes away.
            const ctx = { ...this.props, position: { line: parsed.line, character: 0 } }
            this.props.history.push(toPrettyBlobURL(ctx))
        } else {
            // Unset fixed tooltip if it exists (no URL update necessary).
            this.setFixedTooltip()
        }
    }

    private tooltipActions = (ctx: AbsoluteRepoFilePosition) =>
        ({ definition: this.handleGoToDefinition, references: this.handleFindReferences, dismiss: this.handleDismiss })
}
