import { Remote, ProxyMarked, proxyMarker } from 'comlink'
import { LinkPreview, Unsubscribable } from 'sourcegraph'
import { ProxySubscribable } from '../../extension/api/common'
import { LinkPreviewProviderRegistry } from '../services/linkPreview'
import { registerRemoteProvider } from './common'

/** @internal */
export interface ClientContentAPI extends ProxyMarked {
    $registerLinkPreviewProvider(
        urlMatchPattern: string,
        provider: Remote<((url: string) => ProxySubscribable<LinkPreview | null | undefined>) & ProxyMarked>
    ): Unsubscribable & ProxyMarked
}

/** @internal */
export function createClientContent(registry: LinkPreviewProviderRegistry): ClientContentAPI & ProxyMarked {
    return {
        [proxyMarker]: true,
        $registerLinkPreviewProvider: (urlMatchPattern, providerFunction) =>
            registerRemoteProvider(registry, { urlMatchPattern }, providerFunction),
    }
}
