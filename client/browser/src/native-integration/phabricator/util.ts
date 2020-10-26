/**
 * This hacks javelin Stratcom to ignore command + click actions on sg-clickable tokens.
 * Without this, two windows open when a user command + clicks on a token.
 *
 * TODO could this be eliminated with shadow DOM?
 */
export function metaClickOverride(): void {
    const javelin = (window as any).JX
    if (javelin.Stratcom._dispatchProxyPreMeta) {
        return
    }
    javelin.Stratcom._dispatchProxyPreMeta = javelin.Stratcom._dispatchProxy
    javelin.Stratcom._dispatchProxy = (proxyEvent: {
        __auto__type: string
        __auto__rawEvent: KeyboardEvent
        __auto__target: HTMLElement
    }) => {
        if (
            proxyEvent.__auto__type === 'click' &&
            proxyEvent.__auto__rawEvent.metaKey &&
            proxyEvent.__auto__target.classList.contains('sg-clickable')
        ) {
            return
        }
        return javelin.Stratcom._dispatchProxyPreMeta(proxyEvent)
    }
}
