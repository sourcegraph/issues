import * as runtime from '../../browser/runtime'
import storage from '../../browser/storage'
import { isPhabricator } from '../../context'
import { EventLogger } from '../tracking/EventLogger'

export const DEFAULT_SOURCEGRAPH_URL = 'https://sourcegraph.com'

export let eventLogger = new EventLogger()

export let sourcegraphUrl =
    window.localStorage.getItem('SOURCEGRAPH_URL') || window.SOURCEGRAPH_URL || DEFAULT_SOURCEGRAPH_URL

export let renderMermaidGraphsEnabled = false

export let inlineSymbolSearchEnabled = false

export let useExtensions = false

interface UrlCache {
    [key: string]: string
}

export const repoUrlCache: UrlCache = {}

if (window.SG_ENV === 'EXTENSION') {
    storage.getSync(items => {
        sourcegraphUrl = items.sourcegraphURL
        renderMermaidGraphsEnabled = items.featureFlags.renderMermaidGraphsEnabled
        inlineSymbolSearchEnabled = items.featureFlags.inlineSymbolSearchEnabled
        useExtensions = items.featureFlags.useExtensions
    })
}

export function setSourcegraphUrl(url: string): void {
    sourcegraphUrl = url
}

export function isBrowserExtension(): boolean {
    return window.SOURCEGRAPH_PHABRICATOR_EXTENSION || false
}

export function isSourcegraphDotCom(url: string = sourcegraphUrl): boolean {
    return url === DEFAULT_SOURCEGRAPH_URL
}

export function checkIsOnlySourcegraphDotCom(handler: (res: boolean) => void): void {
    if (window.SG_ENV === 'EXTENSION') {
        storage.getSync(items => handler(isSourcegraphDotCom(items.sourcegraphURL)))
    } else {
        handler(false)
    }
}

export function setRenderMermaidGraphsEnabled(enabled: boolean): void {
    renderMermaidGraphsEnabled = enabled
}

export function setInlineSymbolSearchEnabled(enabled: boolean): void {
    inlineSymbolSearchEnabled = enabled
}

export function setUseExtensions(value: boolean): void {
    useExtensions = value
}

/**
 * modeToHighlightJsName gets the highlight.js name of the language given a
 * mode.
 */
export function modeToHighlightJsName(mode: string): string {
    switch (mode) {
        case 'html':
            return 'xml'
        default:
            return mode
    }
}

export function getPlatformName():
    | 'phabricator-integration'
    | 'safari-extension'
    | 'firefox-extension'
    | 'chrome-extension' {
    if (window.SOURCEGRAPH_PHABRICATOR_EXTENSION) {
        return 'phabricator-integration'
    }

    if (typeof window.safari !== 'undefined') {
        return 'safari-extension'
    }

    return isFirefoxExtension() ? 'firefox-extension' : 'chrome-extension'
}

export function getExtensionVersionSync(): string {
    return runtime.getExtensionVersionSync()
}

export function getExtensionVersion(): Promise<string> {
    return runtime.getExtensionVersion()
}

export function isFirefoxExtension(): boolean {
    return window.navigator.userAgent.indexOf('Firefox') !== -1
}

export function isE2ETest(): boolean {
    return process.env.NODE_ENV === 'test'
}

/**
 * This method created a unique username based on the platform and domain the user is visiting.
 * Examples: sg_dev_phabricator:matt
 */
export function getDomainUsername(domain: string, username: string): string {
    return `${domain}:${username}`
}

/**
 * Check the DOM to see if we can determine if a repository is private or public.
 */
export function isPrivateRepository(): boolean {
    if (isPhabricator) {
        return true
    }
    const header = document.querySelector('.repohead-details-container')
    if (!header) {
        return false
    }
    return !!header.querySelector('.private')
}

export function canFetchForURL(url: string): boolean {
    if (url === DEFAULT_SOURCEGRAPH_URL && isPrivateRepository()) {
        return false
    }
    return true
}
