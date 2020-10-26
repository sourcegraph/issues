import { ProxyMarked, proxy, proxyMarker } from 'comlink'
import { Subject } from 'rxjs'
import * as sourcegraph from 'sourcegraph'
import {
    MessageActionItem,
    NotificationType,
    ShowInputParameters,
    ShowMessageRequestParameters,
    ShowNotificationParameters,
} from '../services/notifications'

/** @internal */
export interface ClientWindowsAPI extends ProxyMarked {
    $showNotification(message: string, type: sourcegraph.NotificationType): void
    $showMessage(message: string): Promise<void>
    $showInputBox(options?: sourcegraph.InputBoxOptions): Promise<string | undefined>
    $showProgress(options: sourcegraph.ProgressOptions): sourcegraph.ProgressReporter & ProxyMarked
}

/** @internal */
export class ClientWindows implements ClientWindowsAPI {
    public readonly [proxyMarker] = true

    constructor(
        /** Called when the client receives a window/showMessage notification. */
        private showMessage: (parameters: ShowNotificationParameters) => void,
        /**
         * Called when the client receives a window/showMessageRequest request and expected to return a promise
         * that resolves to the selected action.
         */
        private showMessageRequest: (parameters: ShowMessageRequestParameters) => Promise<MessageActionItem | null>,
        /**
         * Called when the client receives a window/showInput request and expected to return a promise that
         * resolves to the user's input.
         */
        private showInput: (parameters: ShowInputParameters) => Promise<string | null>,
        private createProgressReporter: (options: sourcegraph.ProgressOptions) => Subject<sourcegraph.Progress>
    ) {}

    public $showNotification(message: string, type: sourcegraph.NotificationType): void {
        this.showMessage({ type, message })
    }

    public $showMessage(message: string): Promise<void> {
        return this.showMessageRequest({ type: NotificationType.Info, message }).then(
            () =>
                // TODO(sqs): update the showInput API to unify null/undefined etc between the old internal API and the new
                // external API.
                undefined
        )
    }

    public $showInputBox(options?: sourcegraph.InputBoxOptions): Promise<string | undefined> {
        return this.showInput({
            message: options?.prompt ? options.prompt : '',
            defaultValue: options?.value,
        }).then(input =>
            // TODO(sqs): update the showInput API to unify null/undefined etc between the old internal API and the new
            // external API.
            input === null ? undefined : input
        )
    }

    public $showProgress(options: sourcegraph.ProgressOptions): sourcegraph.ProgressReporter & ProxyMarked {
        return proxy(this.createProgressReporter(options))
    }
}
