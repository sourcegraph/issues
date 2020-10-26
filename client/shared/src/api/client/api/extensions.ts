import { Remote } from 'comlink'
import { from, Subscription } from 'rxjs'
import { bufferCount, startWith } from 'rxjs/operators'
import { ExtensionExtensionsAPI } from '../../extension/api/extensions'
import { ExecutableExtension, IExtensionsService } from '../services/extensionsService'

/** @internal */
export class ClientExtensions {
    private subscriptions = new Subscription()

    /**
     * Implements the client side of the extensions API.
     *
     * @param connection The connection to the extension host.
     * @param extensions An observable that emits the set of extensions that should be activated
     * upon subscription and whenever it changes.
     */
    constructor(private proxy: Remote<ExtensionExtensionsAPI>, extensionRegistry: IExtensionsService) {
        this.subscriptions.add(
            from(extensionRegistry.activeExtensions)
                .pipe(startWith([] as ExecutableExtension[]), bufferCount(2, 1))
                .subscribe(([oldExtensions, newExtensions]) => {
                    // Diff next state's activated extensions vs. current state's.
                    if (!newExtensions) {
                        newExtensions = oldExtensions
                    }
                    const toActivate = [...newExtensions] // clone to avoid mutating state stored by bufferCount
                    const toDeactivate: ExecutableExtension[] = []
                    const next: ExecutableExtension[] = []
                    if (oldExtensions) {
                        for (const extension of oldExtensions) {
                            const newIndex = toActivate.findIndex(({ id }) => extension.id === id)
                            if (newIndex === -1) {
                                // Extension is no longer activated
                                toDeactivate.push(extension)
                            } else {
                                // Extension is already activated.
                                toActivate.splice(newIndex, 1)
                                next.push(extension)
                            }
                        }
                    }

                    /**
                     * Deactivate extensions that are no longer in use. In practice,
                     * {@link activeExtensions} never deactivates extensions, so this will never be
                     * called (in the current implementation).
                     */
                    for (const extension of toDeactivate) {
                        this.proxy.$deactivateExtension(extension.id).catch(error => {
                            console.warn(`Error deactivating extension ${JSON.stringify(extension.id)}:`, error)
                        })
                    }

                    // Activate extensions that haven't yet been activated.
                    for (const extension of toActivate) {
                        console.log('Activating Sourcegraph extension:', extension.id)
                        this.proxy.$activateExtension(extension.id, extension.scriptURL).catch(error => {
                            console.error(`Error activating extension ${JSON.stringify(extension.id)}:`, error)
                        })
                    }
                })
        )
    }

    public unsubscribe(): void {
        this.subscriptions.unsubscribe()
    }
}
