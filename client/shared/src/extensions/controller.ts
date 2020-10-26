import { from, Observable, Subject, Subscription, Unsubscribable } from 'rxjs'
import { map, publishReplay, refCount } from 'rxjs/operators'
import { Services } from '../api/client/services'
import { ExecuteCommandParameters } from '../api/client/services/command'
import { ContributionRegistry, parseContributionExpressions } from '../api/client/services/contribution'
import { ExtensionsService } from '../api/client/services/extensionsService'
import { NotificationType } from '../api/client/services/notifications'
import { InitData } from '../api/extension/extensionHost'
import { registerBuiltinClientCommands } from '../commands/commands'
import { Notification } from '../notifications/notification'
import { PlatformContext } from '../platform/context'
import { asError, ErrorLike, isErrorLike } from '../util/errors'
import { isDefined } from '../util/types'
import { createExtensionHostClientConnection } from '../api/client/connection'
import { Remote } from 'comlink'
import { FlatExtensionHostAPI } from '../api/contract'

export interface Controller extends Unsubscribable {
    /**
     * Global notification messages that should be displayed to the user, from the following sources:
     *
     * - window/showMessage notifications from extensions
     * - Errors thrown or returned in command invocation
     */
    readonly notifications: Observable<Notification>

    services: Services

    /**
     * Executes the command (registered in the CommandRegistry) specified in params. If an error is thrown, the
     * error is returned *and* emitted on the {@link Controller#notifications} observable.
     *
     * All callers should execute commands using this method instead of calling
     * {@link sourcegraph:CommandRegistry#executeCommand} directly (to ensure errors are emitted as notifications).
     *
     * @param suppressNotificationOnError By default, if command execution throws (or rejects with) an error, the
     * error will be shown in the global notification UI component. Pass suppressNotificationOnError as true to
     * skip this. The error is always returned to the caller.
     */
    executeCommand(parameters: ExecuteCommandParameters, suppressNotificationOnError?: boolean): Promise<any>

    /**
     * Frees all resources associated with this client.
     */
    unsubscribe(): void

<<<<<<< HEAD:shared/src/extensions/controller.ts
    extensionHostAPI: Promise<Remote<FlatExtHostAPI>>
=======
    extHostAPI: Promise<Remote<FlatExtensionHostAPI>>
>>>>>>> main:client/shared/src/extensions/controller.ts
}

/**
 * React props or state containing the client. There should be only a single client for the whole
 * application.
 */
export interface ExtensionsControllerProps<K extends keyof Controller = keyof Controller> {
    /**
     * The client, which is used to communicate with and manage extensions.
     */
    extensionsController: Pick<Controller, K>
}

function messageFromExtension(message: string): string {
    return `From extension:\n\n${message}`
}

/**
 * Creates the controller, which handles all communication between the client application and extensions.
 *
 * There should only be a single controller for the entire client application. The controller's model represents
 * all of the client application state that the client needs to know.
 */
export function createController(context: PlatformContext): Controller {
    const subscriptions = new Subscription()

    const services = new Services(context)
    const initData: Omit<InitData, 'initialSettings'> = {
        sourcegraphURL: context.sourcegraphURL,
        clientApplication: context.clientApplication,
    }
    const extensionHostClientPromise = createExtensionHostClientConnection(
        context.createExtensionHost(),
        services,
        initData,
        context
    )

    subscriptions.add(() => extensionHostClientPromise.then(({ subscription }) => subscription.unsubscribe()))

    const notifications = new Subject<Notification>()

    subscriptions.add(registerBuiltinClientCommands(services, context, extensionHostClientPromise))
    subscriptions.add(registerExtensionContributions(services.contribution, services.extensions))

    // Show messages (that don't need user input) as global notifications.
    subscriptions.add(
        services.notifications.showMessages.subscribe(({ message, type }) => notifications.next({ message, type }))
    )
    subscriptions.add(
        services.notifications.progresses.subscribe(({ title, progress }) => {
            notifications.next({ message: title, progress, type: NotificationType.Log })
        })
    )
    subscriptions.add(
        services.notifications.showMessageRequests.subscribe(({ message, actions, resolve }) => {
            if (!actions || actions.length === 0) {
                alert(messageFromExtension(message))
                resolve(null)
                return
            }
            const value = prompt(
                messageFromExtension(
                    `${message}\n\nValid responses: ${actions.map(({ title }) => JSON.stringify(title)).join(', ')}`
                ),
                actions[0].title
            )
            resolve(actions.find(a => a.title === value) || null)
        })
    )
    subscriptions.add(
        services.notifications.showInputs.subscribe(({ message, defaultValue, resolve }) =>
            resolve(prompt(messageFromExtension(message), defaultValue))
        )
    )

    return {
        notifications,
        services,
        executeCommand: (parameters, suppressNotificationOnError) =>
            services.commands.executeCommand(parameters).catch(error => {
                if (!suppressNotificationOnError) {
                    notifications.next({
                        message: asError(error).message,
                        type: NotificationType.Error,
                        source: parameters.command,
                    })
                }
                return Promise.reject(error)
            }),
        extensionHostAPI: extensionHostClientPromise.then(({ api }) => api),
        unsubscribe: () => subscriptions.unsubscribe(),
    }
}

export function registerExtensionContributions(
    contributionRegistry: Pick<ContributionRegistry, 'registerContributions'>,
    { activeExtensions }: Pick<ExtensionsService, 'activeExtensions'>
): Unsubscribable {
    const contributions = from(activeExtensions).pipe(
        map(extensions =>
            extensions
                .map(({ manifest }) => manifest)
                .filter(
                    (manifest): manifest is Exclude<typeof manifest, ErrorLike | null> =>
                        manifest !== null && !isErrorLike(manifest)
                )
                .map(({ contributes }) => contributes)
                .filter(isDefined)
                .map(contributions => {
                    try {
                        return parseContributionExpressions(contributions)
                    } catch (error) {
                        // An error during evaluation causes all of the contributions in the same entry to be
                        // discarded.
                        console.warn('Discarding contributions: parsing expressions or templates failed.', {
                            contributions,
                            error,
                        })
                        return {}
                    }
                })
        ),
        // Perf optimization: only parse all the context expression once if there are multiple Subscribers.
        // This does not change the behaviour of the Observable, it always emits the current value on Subscription.
        publishReplay(1),
        refCount()
    )
    return contributionRegistry.registerContributions({ contributions })
}
