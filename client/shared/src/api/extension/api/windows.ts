import { Remote, ProxyMarked, proxyMarker } from 'comlink'
import { sortBy } from 'lodash'
import { BehaviorSubject, Observable, of } from 'rxjs'
import * as sourcegraph from 'sourcegraph'
import { asError } from '../../../util/errors'
import { ClientWindowsAPI } from '../../client/api/windows'
import { ViewerUpdate } from '../../client/services/viewerService'
import { ExtensionCodeEditor } from './codeEditor'
import { ExtensionDocuments } from './documents'

interface WindowsProxyData {
    windows: ClientWindowsAPI
}

/**
 * @todo Send the show{Notification,Message,InputBox} requests to the same window (right now they are global).
 * @internal
 */
export class ExtensionWindow implements sourcegraph.Window {
    /** Mutable map of viewer ID to viewer. */
    private viewComponents = new Map<string, ExtensionCodeEditor | sourcegraph.DirectoryViewer>()

    constructor(private proxy: Remote<WindowsProxyData>, private documents: ExtensionDocuments, data: ViewerUpdate[]) {
        this.update(data)
    }

    public readonly activeViewComponentChanges = new BehaviorSubject<sourcegraph.ViewComponent | undefined>(undefined)

    public get visibleViewComponents(): sourcegraph.ViewComponent[] {
        const entries = [...this.viewComponents.entries()]
        return sortBy(entries, 0).map(([, viewComponent]) => viewComponent)
    }

    public get activeViewComponent(): sourcegraph.ViewComponent | undefined {
        return this.activeViewComponentChanges.value
    }

    public showNotification(message: string, type: sourcegraph.NotificationType): void {
        // eslint-disable-next-line @typescript-eslint/no-floating-promises
        this.proxy.windows.$showNotification(message, type)
    }

    public showMessage(message: string): Promise<void> {
        return this.proxy.windows.$showMessage(message)
    }

    public showInputBox(options?: sourcegraph.InputBoxOptions): Promise<string | undefined> {
        return this.proxy.windows.$showInputBox(options)
    }

    public async withProgress<R>(
        options: sourcegraph.ProgressOptions,
        task: (reporter: sourcegraph.ProgressReporter) => Promise<R>
    ): Promise<R> {
        const reporter = await this.showProgress(options)
        try {
            const result = await task(reporter)
            reporter.complete()
            return result
        } catch (error) {
            reporter.error(error)
            throw error
        }
    }

    public async showProgress(options: sourcegraph.ProgressOptions): Promise<sourcegraph.ProgressReporter> {
        const reporterProxy = await this.proxy.windows.$showProgress(options)
        return {
            next: (progress: sourcegraph.Progress) => {
                // eslint-disable-next-line @typescript-eslint/no-floating-promises
                reporterProxy.next(progress)
            },
            error: (value: any) => {
                const error = asError(value)
                // eslint-disable-next-line @typescript-eslint/no-floating-promises
                reporterProxy.error({
                    message: error.message,
                    name: error.name,
                    stack: error.stack,
                })
            },
            complete: () => {
                // eslint-disable-next-line @typescript-eslint/no-floating-promises
                reporterProxy.complete()
            },
        }
    }

    /**
     * Perform a delta update (update/add/delete) of this window's view components.
     */
    public update(data: ViewerUpdate[]): void {
        for (const update of data) {
            const { viewerId } = update
            switch (update.type) {
                case 'added': {
                    const { viewerData } = update
                    let viewer: ExtensionCodeEditor | sourcegraph.DirectoryViewer
                    switch (viewerData.type) {
                        case 'CodeEditor':
                            viewer = new ExtensionCodeEditor(
                                { viewerId, ...viewerData },
                                this.documents.get(viewerData.resource)
                            )
                            break
                        case 'DirectoryViewer':
                            viewer = {
                                type: 'DirectoryViewer',
                                // Since directories don't have any state beyond the immutable URI,
                                // we can set the model to a static object for now and don't need to track directory models in a Map.
                                directory: {
                                    uri: new URL(viewerData.resource),
                                },
                            }
                            break
                    }
                    this.viewComponents.set(viewerId, viewer)
                    if (viewerData.isActive) {
                        this.activeViewComponentChanges.next(viewer)
                    }
                    break
                }
                case 'updated': {
                    const editor = this.viewComponents.get(viewerId)
                    if (!editor) {
                        throw new Error(`Could not perform update: viewer ${viewerId} not found`)
                    }
                    if (editor.type !== 'CodeEditor') {
                        throw new Error(
                            `Could not perform update: viewer ${viewerId} is type ${editor.type}, not CodeEditor`
                        )
                    }
                    editor.update(update.viewerData)
                    break
                }
                case 'deleted': {
                    this.viewComponents.delete(viewerId)
                    break
                }
            }
        }
    }

    public toJSON(): any {
        return { visibleViewComponents: this.visibleViewComponents, activeViewComponent: this.activeViewComponent }
    }
}

/** @internal */
export interface ExtensionWindowsAPI extends ProxyMarked {
    $acceptWindowData(viewerUpdates: ViewerUpdate[]): void
}

/**
 * @internal
 * @todo Support more than 1 window.
 */
export class ExtensionWindows implements ExtensionWindowsAPI, ProxyMarked {
    public readonly [proxyMarker] = true

    public activeWindow: ExtensionWindow
    public readonly activeWindowChanges: Observable<sourcegraph.Window>

    /** @internal */
    constructor(private proxy: Remote<{ windows: ClientWindowsAPI }>, private documents: ExtensionDocuments) {
        this.activeWindow = new ExtensionWindow(this.proxy, this.documents, [])
        this.activeWindowChanges = of(this.activeWindow)
    }

    /**
     * Returns all known windows.
     *
     * @internal
     */
    public getAll(): sourcegraph.Window[] {
        return [this.activeWindow]
    }

    /** @internal */
    public $acceptWindowData(viewerUpdates: ViewerUpdate[]): void {
        this.activeWindow.update(viewerUpdates)
    }
}
