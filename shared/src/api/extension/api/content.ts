import * as comlink from 'comlink'
import { Unsubscribable } from 'rxjs'
import { LinkPreviewProvider } from 'sourcegraph'
import { ClientContentAPI } from '../../client/api/content'
import { synchronousSubscription } from '../../util'
import { toProxyableSubscribable } from './common'

/** @internal */
export class ExtContent {
    constructor(private proxy: comlink.Remote<ClientContentAPI>) {}

    public registerLinkPreviewProvider(urlMatchPattern: string, provider: LinkPreviewProvider): Unsubscribable {
        const providerFunction: comlink.Local<
            Parameters<ClientContentAPI['$registerLinkPreviewProvider']>[1]
        > = comlink.proxy((url: string) =>
            toProxyableSubscribable(provider.provideLinkPreview(new URL(url)), preview => preview)
        )
        return synchronousSubscription(this.proxy.$registerLinkPreviewProvider(urlMatchPattern, providerFunction))
    }
}
