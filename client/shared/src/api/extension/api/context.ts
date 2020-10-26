import { Remote } from 'comlink'
import { ContextValues } from 'sourcegraph'
import { ClientContextAPI } from '../../client/api/context'

/** @internal */
export class ExtensionContext {
    constructor(private proxy: Remote<ClientContextAPI>) {}

    public updateContext(updates: ContextValues): void {
        // eslint-disable-next-line @typescript-eslint/no-floating-promises
        this.proxy.$acceptContextUpdates(updates)
    }
}
