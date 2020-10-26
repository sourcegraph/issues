// This Source Code Form is subject to the terms of the Mozilla Public
// license, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
declare module 'webextension-polyfill' {
    export = browser
}

declare namespace browser {
    /**
     * An object that allows adding, removing and inspecting listeners.
     * Event listeners may return a value.
     */
    interface CallbackEventEmitter<F extends (...args: any) => any> {
        addListener(callback: F): void
        removeListener(callback: F): void
        hasListener(callback: F): boolean
    }

    /**
     * Simpler version for events with a single parameter that always return void.
     */
    type EventEmitter<T> = CallbackEventEmitter<(event: T) => void>
}

declare namespace browser.alarms {
    interface Alarm {
        name: string
        scheduledTime: number
        periodInMinutes?: number
    }

    interface When {
        when?: number
        periodInMinutes?: number
    }
    interface DelayInMinutes {
        delayInMinutes?: number
        periodInMinutes?: number
    }
    function create(name?: string, alarmInfo?: When | DelayInMinutes): void
    function get(name?: string): Promise<Alarm | undefined>
    function getAll(): Promise<Alarm[]>
    function clear(name?: string): Promise<boolean>
    function clearAll(): Promise<boolean>

    const onAlarm: EventEmitter<Alarm>
}

declare namespace browser.bookmarks {
    type BookmarkTreeNodeUnmodifiable = 'managed'
    type BookmarkTreeNodeType = 'bookmark' | 'folder' | 'separator'
    interface BookmarkTreeNode {
        id: string
        parentId?: string
        index?: number
        url?: string
        title: string
        dateAdded?: number
        dateGroupModified?: number
        unmodifiable?: BookmarkTreeNodeUnmodifiable
        children?: BookmarkTreeNode[]
        type?: BookmarkTreeNodeType
    }

    interface CreateDetails {
        parentId?: string
        index?: number
        title?: string
        type?: BookmarkTreeNodeType
        url?: string
    }

    function create(bookmark: CreateDetails): Promise<BookmarkTreeNode>
    function get(idOrIdList: string | string[]): Promise<BookmarkTreeNode[]>
    function getChildren(id: string): Promise<BookmarkTreeNode[]>
    function getRecent(numberOfItems: number): Promise<BookmarkTreeNode[]>
    function getSubTree(id: string): Promise<[BookmarkTreeNode]>
    function getTree(): Promise<[BookmarkTreeNode]>

    type Destination =
        | {
              parentId: string
              index?: number
          }
        | {
              index: number
              parentId?: string
          }
    function move(id: string, destination: Destination): Promise<BookmarkTreeNode>
    function remove(id: string): Promise<void>
    function removeTree(id: string): Promise<void>
    function search(
        query:
            | string
            | {
                  query?: string
                  url?: string
                  title?: string
              }
    ): Promise<BookmarkTreeNode[]>
    function update(id: string, changes: { title: string; url: string }): Promise<BookmarkTreeNode>

    const onCreated: CallbackEventEmitter<(id: string, bookmark: BookmarkTreeNode) => void>
    const onRemoved: CallbackEventEmitter<(
        id: string,
        removeInfo: {
            parentId: string
            index: number
            node: BookmarkTreeNode
        }
    ) => void>
    const onChanged: CallbackEventEmitter<(
        id: string,
        changeInfo: {
            title: string
            url?: string
        }
    ) => void>
    const onMoved: CallbackEventEmitter<(
        id: string,
        moveInfo: {
            parentId: string
            index: number
            oldParentId: string
            oldIndex: number
        }
    ) => void>
}

declare namespace browser.browserAction {
    type ColorArray = [number, number, number, number]
    type ImageDataType = ImageData

    function setTitle(details: { title: string | null; tabId?: number }): void
    function getTitle(details: { tabId?: number }): Promise<string>

    interface IconViaPath {
        path: string | { [size: number]: string }
        tabId?: number
    }

    interface IconViaImageData {
        imageData: ImageDataType | { [size: number]: ImageDataType }
        tabId?: number
    }

    interface IconReset {
        imageData?: {} | null
        path?: {} | null
        tabId?: number
    }

    function setIcon(details: IconViaPath | IconViaImageData | IconReset): Promise<void>
    function setPopup(details: { popup: string | null; tabId?: number }): void
    function getPopup(details: { tabId?: number }): Promise<string>
    function openPopup(): Promise<void>
    function setBadgeText(details: { text: string | null; tabId?: number }): void
    function getBadgeText(details: { tabId?: number }): Promise<string>
    function setBadgeBackgroundColor(details: { color: string | ColorArray | null; tabId?: number }): void
    function getBadgeBackgroundColor(details: { tabId?: number }): Promise<ColorArray>

    interface SetBadgeTextColorDetails {
        /**
         * The color, specified as one of:
         * - a string: any CSS color value, for example "red", "#FF0000", or "rgb(255,0,0)". If the string is not a valid color, the returned promise will be rejected and the text color won't be altered.
         * - a `browserAction.ColorArray` object.
         * - `null`. If a tabId is specified, it removes the tab-specific badge text color so that the tab inherits the global badge text color. Otherwise it reverts the global badge text color to the default value.
         */
        color: string | ColorArray | null
    }
    function setBadgeTextColor(details: SetBadgeTextColorDetails & { tabId?: number }): void
    // a union type would allow specifying both, which is not allowed.
    // eslint-disable-next-line @typescript-eslint/unified-signatures
    function setBadgeTextColor(details: SetBadgeTextColorDetails & { windowId?: number }): void

    function getBadgeTextColor(details: { tabId?: string }): Promise<ColorArray>
    // a union type would allow specifying both, which is not allowed.
    // eslint-disable-next-line @typescript-eslint/unified-signatures
    function getBadgeTextColor(details: { windowId?: string }): Promise<ColorArray>

    function enable(tabId?: number): void
    function disable(tabId?: number): void

    const onClicked: EventEmitter<tabs.Tab>
}

declare namespace browser.browsingData {
    interface DataTypeSet {
        cache?: boolean
        cookies?: boolean
        downloads?: boolean
        fileSystems?: boolean
        formData?: boolean
        history?: boolean
        indexedDB?: boolean
        localStorage?: boolean
        passwords?: boolean
        pluginData?: boolean
        serverBoundCertificates?: boolean
        serviceWorkers?: boolean
    }

    interface DataRemovalOptions {
        since?: number
        originTypes?: { unprotectedWeb: boolean }
    }

    function remove(removalOptions: DataRemovalOptions, dataTypes: DataTypeSet): Promise<void>
    function removeCache(removalOptions?: DataRemovalOptions): Promise<void>
    function removeCookies(removalOptions: DataRemovalOptions): Promise<void>
    function removeDownloads(removalOptions: DataRemovalOptions): Promise<void>
    function removeFormData(removalOptions: DataRemovalOptions): Promise<void>
    function removeHistory(removalOptions: DataRemovalOptions): Promise<void>
    function removePasswords(removalOptions: DataRemovalOptions): Promise<void>
    function removePluginData(removalOptions: DataRemovalOptions): Promise<void>
    function settings(): Promise<{
        options: DataRemovalOptions
        dataToRemove: DataTypeSet
        dataRemovalPermitted: DataTypeSet
    }>
}

declare namespace browser.commands {
    interface Command {
        name?: string
        description?: string
        shortcut?: string
    }

    function getAll(): Promise<Command[]>

    const onCommand: EventEmitter<string>
}

declare namespace browser.menus {
    type ContextType =
        | 'all'
        | 'audio'
        | 'bookmarks'
        | 'browser_action'
        | 'editable'
        | 'frame'
        | 'image'
        // | "launcher" unsupported
        | 'link'
        | 'page'
        | 'page_action'
        | 'password'
        | 'selection'
        | 'tab'
        | 'tools_menu'
        | 'video'

    type ItemType = 'normal' | 'checkbox' | 'radio' | 'separator'

    interface OnClickData {
        bookmarkId?: string
        checked?: boolean
        editable: boolean
        frameId?: number
        frameUrl?: string
        linkText?: string
        linkUrl?: string
        mediaType?: string
        menuItemId: number | string
        modifiers: string[]
        pageUrl?: string
        parentMenuItemId?: number | string
        selectionText?: string
        srcUrl?: string
        targetElementId?: number
        wasChecked?: boolean
    }

    const ACTION_MENU_TOP_LEVEL_LIMIT: number

    function create(
        createProperties: {
            checked?: boolean
            command?: '_execute_browser_action' | '_execute_page_action' | '_execute_sidebar_action'
            contexts?: ContextType[]
            documentUrlPatterns?: string[]
            enabled?: boolean
            icons?: object
            id?: string
            onclick?: (info: OnClickData, tab: tabs.Tab) => void
            parentId?: number | string
            targetUrlPatterns?: string[]
            title?: string
            type?: ItemType
            visible?: boolean
        },
        callback?: () => void
    ): number | string

    function getTargetElement(targetElementId: number): object | null

    function refresh(): Promise<void>

    function remove(menuItemId: number | string): Promise<void>

    function removeAll(): Promise<void>

    function update(
        id: number | string,
        updateProperties: {
            checked?: boolean
            command?: '_execute_browser_action' | '_execute_page_action' | '_execute_sidebar_action'
            contexts?: ContextType[]
            documentUrlPatterns?: string[]
            enabled?: boolean
            onclick?: (info: OnClickData, tab: tabs.Tab) => void
            parentId?: number | string
            targetUrlPatterns?: string[]
            title?: string
            type?: ItemType
            visible?: boolean
        }
    ): Promise<void>

    const onClicked: CallbackEventEmitter<(info: OnClickData, tab: tabs.Tab) => void>

    const onHidden: CallbackEventEmitter<() => void>

    const onShown: CallbackEventEmitter<(info: OnClickData, tab: tabs.Tab) => void>
}

declare namespace browser.contextualIdentities {
    type IdentityColor = 'blue' | 'turquoise' | 'green' | 'yellow' | 'orange' | 'red' | 'pink' | 'purple'
    type IdentityIcon = 'fingerprint' | 'briefcase' | 'dollar' | 'cart' | 'circle'

    interface ContextualIdentity {
        cookieStoreId: string
        color: IdentityColor
        icon: IdentityIcon
        name: string
    }

    function create(details: { name: string; color: IdentityColor; icon: IdentityIcon }): Promise<ContextualIdentity>
    function get(cookieStoreId: string): Promise<ContextualIdentity | null>
    function query(details: { name?: string }): Promise<ContextualIdentity[]>
    function update(
        cookieStoreId: string,
        details: {
            name: string
            color: IdentityColor
            icon: IdentityIcon
        }
    ): Promise<ContextualIdentity>
    function remove(cookieStoreId: string): Promise<ContextualIdentity | null>
}

declare namespace browser.cookies {
    interface Cookie {
        name: string
        value: string
        domain: string
        hostOnly: boolean
        path: string
        secure: boolean
        httpOnly: boolean
        session: boolean
        expirationDate?: number
        storeId: string
    }

    interface CookieStore {
        id: string
        tabIds: number[]
    }

    type OnChangedCause = 'evicted' | 'expired' | 'explicit' | 'expired_overwrite' | 'overwrite'

    function get(details: { url: string; name: string; storeId?: string }): Promise<Cookie | null>
    function getAll(details: {
        url?: string
        name?: string
        domain?: string
        path?: string
        secure?: boolean
        session?: boolean
        storeId?: string
    }): Promise<Cookie[]>
    function set(details: {
        url: string
        name?: string
        domain?: string
        path?: string
        secure?: boolean
        httpOnly?: boolean
        expirationDate?: number
        storeId?: string
    }): Promise<Cookie>
    function remove(details: { url: string; name: string; storeId?: string }): Promise<Cookie | null>
    function getAllCookieStores(): Promise<CookieStore[]>

    const onChanged: EventEmitter<{
        removed: boolean
        cookie: Cookie
        cause: OnChangedCause
    }>
}

declare namespace browser.contentScripts {
    interface RegisteredContentScriptOptions {
        allFrames?: boolean
        css?: ({ file: string } | { code: string })[]
        excludeGlobs?: string[]
        excludeMatches?: string[]
        includeGlobs?: string[]
        js?: ({ file: string } | { code: string })[]
        matchAboutBlank?: boolean
        matches: string[]
        runAt?: 'document_start' | 'document_end' | 'document_idle'
    }

    interface RegisteredContentScript {
        unregister: () => void
    }

    function register(contentScriptOptions: RegisteredContentScriptOptions): Promise<RegisteredContentScript>
}

declare namespace browser.devtools.inspectedWindow {
    const tabId: number

    function eval(
        expression: string
    ): Promise<[any, { isException: boolean; value: string } | { isError: boolean; code: string }]>

    function reload(reloadOptions?: { ignoreCache?: boolean; userAgent?: string; injectedScript?: string }): void
}

declare namespace browser.devtools.network {
    const onNavigated: EventEmitter<string>
}

declare namespace browser.devtools.panels {
    interface ExtensionPanel {
        onShown: EventEmitter<Window>
        onHidden: EventEmitter<void>
    }

    function create(title: string, iconPath: string, pagePath: string): Promise<ExtensionPanel>
}

declare namespace browser.downloads {
    type FilenameConflictAction = 'uniquify' | 'overwrite' | 'prompt'

    type InterruptReason =
        | 'FILE_FAILED'
        | 'FILE_ACCESS_DENIED'
        | 'FILE_NO_SPACE'
        | 'FILE_NAME_TOO_LONG'
        | 'FILE_TOO_LARGE'
        | 'FILE_VIRUS_INFECTED'
        | 'FILE_TRANSIENT_ERROR'
        | 'FILE_BLOCKED'
        | 'FILE_SECURITY_CHECK_FAILED'
        | 'FILE_TOO_SHORT'
        | 'NETWORK_FAILED'
        | 'NETWORK_TIMEOUT'
        | 'NETWORK_DISCONNECTED'
        | 'NETWORK_SERVER_DOWN'
        | 'NETWORK_INVALID_REQUEST'
        | 'SERVER_FAILED'
        | 'SERVER_NO_RANGE'
        | 'SERVER_BAD_CONTENT'
        | 'SERVER_UNAUTHORIZED'
        | 'SERVER_CERT_PROBLEM'
        | 'SERVER_FORBIDDEN'
        | 'USER_CANCELED'
        | 'USER_SHUTDOWN'
        | 'CRASH'

    type DangerType = 'file' | 'url' | 'content' | 'uncommon' | 'host' | 'unwanted' | 'safe' | 'accepted'

    type State = 'in_progress' | 'interrupted' | 'complete'

    interface DownloadItem {
        id: number
        url: string
        referrer: string
        filename: string
        incognito: boolean
        danger: string
        mime: string
        startTime: string
        endTime?: string
        estimatedEndTime?: string
        state: string
        paused: boolean
        canResume: boolean
        error?: string
        bytesReceived: number
        totalBytes: number
        fileSize: number
        exists: boolean
        byExtensionId?: string
        byExtensionName?: string
    }

    interface Delta<T> {
        current?: T
        previous?: T
    }

    type StringDelta = Delta<string>
    type DoubleDelta = Delta<number>
    type BooleanDelta = Delta<boolean>
    type DownloadTime = Date | string | number

    interface DownloadQuery {
        query?: string[]
        startedBefore?: DownloadTime
        startedAfter?: DownloadTime
        endedBefore?: DownloadTime
        endedAfter?: DownloadTime
        totalBytesGreater?: number
        totalBytesLess?: number
        filenameRegex?: string
        urlRegex?: string
        limit?: number
        orderBy?: string
        id?: number
        url?: string
        filename?: string
        danger?: DangerType
        mime?: string
        startTime?: string
        endTime?: string
        state?: State
        paused?: boolean
        error?: InterruptReason
        bytesReceived?: number
        totalBytes?: number
        fileSize?: number
        exists?: boolean
    }

    function download(options: {
        url: string
        filename?: string
        conflictAction?: string
        saveAs?: boolean
        method?: string
        headers?: { [key: string]: string }
        body?: string
    }): Promise<number>
    function search(query: DownloadQuery): Promise<DownloadItem[]>
    function pause(downloadId: number): Promise<void>
    function resume(downloadId: number): Promise<void>
    function cancel(downloadId: number): Promise<void>
    // unsupported: function getFileIcon(downloadId: number, options?: { size?: number }):
    //              Promise<string>;
    function open(downloadId: number): Promise<void>
    function show(downloadId: number): Promise<void>
    function showDefaultFolder(): void
    function erase(query: DownloadQuery): Promise<number[]>
    function removeFile(downloadId: number): Promise<void>
    // unsupported: function acceptDanger(downloadId: number): Promise<void>;
    // unsupported: function drag(downloadId: number): Promise<void>;
    // unsupported: function setShelfEnabled(enabled: boolean): void;

    const onCreated: EventEmitter<DownloadItem>
    const onErased: EventEmitter<number>
    const onChanged: EventEmitter<{
        id: number
        url?: StringDelta
        filename?: StringDelta
        danger?: StringDelta
        mime?: StringDelta
        startTime?: StringDelta
        endTime?: StringDelta
        state?: StringDelta
        canResume?: BooleanDelta
        paused?: BooleanDelta
        error?: StringDelta
        totalBytes?: DoubleDelta
        fileSize?: DoubleDelta
        exists?: BooleanDelta
    }>
}

declare namespace browser.events {
    interface UrlFilter {
        hostContains?: string
        hostEquals?: string
        hostPrefix?: string
        hostSuffix?: string
        pathContains?: string
        pathEquals?: string
        pathPrefix?: string
        pathSuffix?: string
        queryContains?: string
        queryEquals?: string
        queryPrefix?: string
        querySuffix?: string
        urlContains?: string
        urlEquals?: string
        urlMatches?: string
        originAndPathMatches?: string
        urlPrefix?: string
        urlSuffix?: string
        schemes?: string[]
        ports?: (number | number[])[]
    }
}

declare namespace browser.extension {
    type ViewType = 'tab' | 'notification' | 'popup'

    const lastError: string | null
    const inIncognitoContext: boolean

    function getURL(path: string): string
    function getViews(fetchProperties?: { type?: ViewType; windowId?: number }): Window[]
    function getBackgroundPage(): Window
    function isAllowedIncognitoAccess(): Promise<boolean>
    function isAllowedFileSchemeAccess(): Promise<boolean>
    // unsupported: events as they are deprecated
}

declare namespace browser.extensionTypes {
    type ImageFormat = 'jpeg' | 'png'
    interface ImageDetails {
        format: ImageFormat
        quality: number
    }
    type RunAt = 'document_start' | 'document_end' | 'document_idle'
    interface InjectDetails {
        allFrames?: boolean
        code?: string
        file?: string
        frameId?: number
        matchAboutBlank?: boolean
        runAt?: RunAt
    }
    type InjectDetailsCSS = InjectDetails & { cssOrigin?: 'user' | 'author' }
}

declare namespace browser.history {
    type TransitionType =
        | 'link'
        | 'typed'
        | 'auto_bookmark'
        | 'auto_subframe'
        | 'manual_subframe'
        | 'generated'
        | 'auto_toplevel'
        | 'form_submit'
        | 'reload'
        | 'keyword'
        | 'keyword_generated'

    interface HistoryItem {
        id: string
        url?: string
        title?: string
        lastVisitTime?: number
        visitCount?: number
        typedCount?: number
    }

    interface VisitItem {
        id: string
        visitId: string
        VisitTime?: number
        refferingVisitId: string
        transition: TransitionType
    }

    function search(query: {
        text: string
        startTime?: number | string | Date
        endTime?: number | string | Date
        maxResults?: number
    }): Promise<HistoryItem[]>

    function getVisits(details: { url: string }): Promise<VisitItem[]>

    function addUrl(details: {
        url: string
        title?: string
        transition?: TransitionType
        visitTime?: number | string | Date
    }): Promise<void>

    function deleteUrl(details: { url: string }): Promise<void>

    function deleteRange(range: { startTime: number | string | Date; endTime: number | string | Date }): Promise<void>

    function deleteAll(): Promise<void>

    const onVisited: EventEmitter<HistoryItem>

    // TODO: Ensure that urls is not `urls: [string]` instead
    const onVisitRemoved: EventEmitter<{ allHistory: boolean; urls: string[] }>
}

declare namespace browser.i18n {
    type LanguageCode = string

    function getAcceptLanguages(): Promise<LanguageCode[]>

    function getMessage(messageName: string, substitutions?: string | string[]): string

    function getUILanguage(): LanguageCode

    function detectLanguage(
        text: string
    ): Promise<{
        isReliable: boolean
        languages: { language: LanguageCode; percentage: number }[]
    }>
}

declare namespace browser.identity {
    function getRedirectURL(): string
    function launchWebAuthFlow(details: { url: string; interactive: boolean }): Promise<string>
}

declare namespace browser.idle {
    type IdleState = 'active' | 'idle' /* unsupported: | "locked" */

    function queryState(detectionIntervalInSeconds: number): Promise<IdleState>
    function setDetectionInterval(intervalInSeconds: number): void

    const onStateChanged: EventEmitter<IdleState>
}

declare namespace browser.management {
    interface ExtensionInfo {
        description: string
        // unsupported: disabledReason: string,
        enabled: boolean
        homepageUrl: string
        hostPermissions: string[]
        icons: { size: number; url: string }[]
        id: string
        installType: 'admin' | 'development' | 'normal' | 'sideload' | 'other'
        mayDisable: boolean
        name: string
        // unsupported: offlineEnabled: boolean,
        optionsUrl: string
        permissions: string[]
        shortName: string
        // unsupported: type: string,
        updateUrl: string
        version: string
        // unsupported: versionName: string,
    }

    function getSelf(): Promise<ExtensionInfo>
    function uninstallSelf(options: { showConfirmDialog: boolean; dialogMessage: string }): Promise<void>
}

declare namespace browser.notifications {
    type TemplateType = 'basic' /* | "image" | "list" | "progress" */

    interface NotificationOptions {
        type: TemplateType
        message: string
        title: string
        iconUrl?: string
    }

    function create(id: string | null, options: NotificationOptions): Promise<string>
    function create(options: NotificationOptions): Promise<string>

    function clear(id: string): Promise<boolean>

    function getAll(): Promise<{ [key: string]: NotificationOptions }>

    const onClosed: EventEmitter<string>

    const onClicked: EventEmitter<string>
}

declare namespace browser.omnibox {
    type OnInputEnteredDisposition = 'currentTab' | 'newForegroundTab' | 'newBackgroundTab'
    interface SuggestResult {
        content: string
        description: string
    }

    function setDefaultSuggestion(suggestion: { description: string }): void

    const onInputStarted: EventEmitter<void>
    const onInputChanged: CallbackEventEmitter<(text: string, suggest: (arg: SuggestResult[]) => void) => void>
    const onInputEntered: CallbackEventEmitter<(text: string, disposition: OnInputEnteredDisposition) => void>
    const onInputCancelled: EventEmitter<void>
}

declare namespace browser.pageAction {
    type ImageDataType = ImageData

    function show(tabId: number): void

    function hide(tabId: number): void

    function setTitle(details: { tabId: number; title: string }): void

    function getTitle(details: { tabId: number }): Promise<string>

    function setIcon(details: { tabId: number; path?: string | object; imageData?: ImageDataType }): Promise<void>

    function setPopup(details: { tabId: number; popup: string }): void

    function getPopup(details: { tabId: number }): Promise<string>

    const onClicked: EventEmitter<tabs.Tab>
}

declare namespace browser.permissions {
    type Permission =
        | 'activeTab'
        | 'alarms'
        | 'background'
        | 'bookmarks'
        | 'browsingData'
        | 'browserSettings'
        | 'clipboardRead'
        | 'clipboardWrite'
        | 'contextMenus'
        | 'contextualIdentities'
        | 'cookies'
        | 'downloads'
        | 'downloads.open'
        | 'find'
        | 'geolocation'
        | 'history'
        | 'identity'
        | 'idle'
        | 'management'
        | 'menus'
        | 'nativeMessaging'
        | 'notifications'
        | 'pkcs11'
        | 'privacy'
        | 'proxy'
        | 'sessions'
        | 'storage'
        | 'tabs'
        | 'theme'
        | 'topSites'
        | 'unlimitedStorage'
        | 'webNavigation'
        | 'webRequest'
        | 'webRequestBlocking'

    interface Permissions {
        origins?: string[]
        permissions?: Permission[]
    }

    function contains(permissions: Permissions): Promise<boolean>

    function getAll(): Promise<Permissions>

    function remove(permissions: Permissions): Promise<boolean>

    function request(permissions: Permissions): Promise<boolean>

    /**
     * Not supported yet in Firefox:
     * https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/API/permissions/onAdded#Browser_compatibility
     */
    const onAdded: EventEmitter<Permissions> | undefined

    /**
     * Not supported yet in Firefox:
     * https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/API/permissions/onAdded#Browser_compatibility
     */
    const onRemoved: EventEmitter<Permissions> | undefined
}

declare namespace browser.runtime {
    const lastError: string | null
    const id: string

    interface Port {
        /**
         * The port's name, defined in the runtime.connect() or tabs.connect() call that created it.
         * If this port is connected to a native application, its name is the name of the native application.
         */
        name: string
        disconnect(): void
        error: Error | null
        onDisconnect: EventEmitter<this>
        onMessage: EventEmitter<any>
        postMessage(message: any): void
    }
    interface PortWithSender extends Port {
        /**
         * Contains information about the sender of the message.
         * This property will only be present on ports passed to onConnect/onConnectExternal listeners.
         */
        sender: MessageSender
    }

    interface MessageSender {
        tab?: tabs.Tab
        frameId?: number
        id?: string
        url?: string
        tlsChannelId?: string
    }

    type PlatformOs = 'mac' | 'win' | 'android' | 'cros' | 'linux' | 'openbsd'
    type PlatformArch = 'arm' | 'x86-32' | 'x86-64'
    type PlatformNaclArch = 'arm' | 'x86-32' | 'x86-64'

    interface PlatformInfo {
        os: PlatformOs
        arch: PlatformArch
    }

    // type RequestUpdateCheckStatus = "throttled" | "no_update" | "update_available";
    type OnInstalledReason = 'install' | 'update' | 'chrome_update' | 'shared_module_update'
    type OnRestartRequiredReason = 'app_update' | 'os_update' | 'periodic'

    interface FirefoxSpecificProperties {
        id?: string
        strict_min_version?: string
        strict_max_version?: string
        update_url?: string
    }

    type IconPath = { [urlName: string]: string } | string

    interface Manifest {
        // Required
        manifest_version: 2
        name: string
        version: string
        /** Required in Microsoft Edge */
        author?: string

        // Optional

        // ManifestBase
        description?: string
        homepage_url?: string
        short_name?: string

        // WebExtensionManifest
        background?: {
            page: string
            script: string[]
            persistent?: boolean
        }
        content_scripts?: {
            matches: string[]
            exclude_matches?: string[]
            include_globs?: string[]
            exclude_globs?: string[]
            css?: string[]
            js?: string[]
            all_frames?: boolean
            match_about_blank?: boolean
            run_at?: 'document_start' | 'document_end' | 'document_idle'
        }[]
        content_security_policy?: string
        developer?: {
            name?: string
            url?: string
        }
        icons?: {
            [imgSize: string]: string
        }
        incognito?: 'spanning' | 'split' | 'not_allowed'
        optional_permissions?: permissions.Permission[]
        options_ui?: {
            page: string
            browser_style?: boolean
            chrome_style?: boolean
            open_in_tab?: boolean
        }
        permissions?: permissions.Permission[]
        web_accessible_resources?: string[]

        // WebExtensionLangpackManifest
        languages: {
            [langCode: string]: {
                chrome_resources: {
                    [resName: string]: string | { [urlName: string]: string }
                }
                version: string
            }
        }
        langpack_id?: string
        sources?: {
            [srcName: string]: {
                base_path: string
                paths?: string[]
            }
        }

        // Extracted from components
        browser_action?: {
            default_title?: string
            default_icon?: IconPath
            theme_icons?: {
                light: string
                dark: string
                size: number
            }[]
            default_popup?: string
            browser_style?: boolean
            default_area?: 'navbar' | 'menupanel' | 'tabstrip' | 'personaltoolbar'
        }
        commands?: {
            [keyName: string]: {
                suggested_key?: {
                    default?: string
                    mac?: string
                    linux?: string
                    windows?: string
                    chromeos?: string
                    android?: string
                    ios?: string
                }
                description?: string
            }
        }
        default_locale?: i18n.LanguageCode
        devtools_page?: string
        omnibox?: {
            keyword: string
        }
        page_action?: {
            default_title?: string
            default_icon?: IconPath
            default_popup?: string
            browser_style?: boolean
            show_matches?: string[]
            hide_matches?: string[]
        }
        sidebar_action?: {
            default_panel: string
            default_title?: string
            default_icon?: IconPath
            browser_style?: boolean
        }

        // Firefox specific
        applications?: {
            gecko?: FirefoxSpecificProperties
        }
        browser_specific_settings?: {
            gecko?: FirefoxSpecificProperties
        }
        experiment_apis?: any
        protocol_handlers?: {
            name: string
            protocol: string
            uriTemplate: string
        }

        // Opera specific
        minimum_opera_version?: string

        // Chrome specific
        action?: any
        automation?: any
        background_page?: any
        chrome_settings_overrides?: {
            homepage?: string
            search_provider?: {
                name: string
                search_url: string
                keyword?: string
                favicon_url?: string
                suggest_url?: string
                instant_url?: string
                is_default?: string
                image_url?: string
                search_url_post_params?: string
                instant_url_post_params?: string
                image_url_post_params?: string
                alternate_urls?: string[]
                prepopulated_id?: number
            }
        }
        chrome_ui_overrides?: {
            bookmarks_ui?: {
                remove_bookmark_shortcut?: true
                remove_button?: true
            }
        }
        chrome_url_overrides?: {
            newtab?: string
            bookmarks?: string
            history?: string
        }
        content_capabilities?: any
        converted_from_user_script?: any
        current_locale?: any
        declarative_net_request?: any
        event_rules?: any[]
        export?: {
            whitelist?: string[]
        }
        externally_connectable?: {
            ids?: string[]
            matches?: string[]
            accepts_tls_channel_id?: boolean
        }
        file_browser_handlers?: {
            id: string
            default_title: string
            file_filters: string[]
        }[]
        file_system_provider_capabilities?: {
            source: 'file' | 'device' | 'network'
            configurable?: boolean
            multiple_mounts?: boolean
            watchable?: boolean
        }
        import?: {
            id: string
            minimum_version?: string
        }[]
        input_components?: any
        key?: string
        minimum_chrome_version?: string
        nacl_modules?: {
            path: string
            mime_type: string
        }[]
        oauth2?: any
        offline_enabled?: boolean
        options_page?: string
        platforms?: any
        requirements?: any
        sandbox?: {
            pages: string[]
            content_security_policy?: string
        }[]
        signature?: any
        spellcheck?: any
        storage?: {
            managed_schema: string
        }
        system_indicator?: any
        tts_engine?: {
            voice: {
                voice_name: string
                lang?: string
                gender?: 'male' | 'female'
                event_types: ('start' | 'word' | 'sentence' | 'marker' | 'end' | 'error')[]
            }[]
        }
        update_url?: string
        version_name?: string
    }

    /**
     * Only defined in the background page
     */
    const getBackgroundPage: (() => Promise<Window>) | undefined

    function openOptionsPage(): Promise<void>
    function getManifest(): Manifest

    function getURL(path: string): string
    function setUninstallURL(url: string): Promise<void>
    function reload(): void
    // Will not exist: https://bugzilla.mozilla.org/show_bug.cgi?id=1314922
    // function RequestUpdateCheck(): Promise<RequestUpdateCheckStatus>;

    interface ConnectInfo {
        name?: string
        includeTlsChannelId?: boolean
    }
    /**
     * @param connectInfo Details of the connection
     */
    function connect(connectInfo?: ConnectInfo): Port
    /**
     * @param extensionId The ID of the extension to connect to. If the target has set an ID explicitly using the applications key in manifest.json, then extensionId should have that value. Otherwise it should have the ID that was generated for the target.
     * @param connectInfo Details of the connection
     */
    function connect(extensionId?: string, connectInfo?: ConnectInfo): Port

    function connectNative(application: string): Port

    function sendMessage(
        message: any,
        options?: { includeTlsChannelId?: boolean; toProxyScript?: boolean }
    ): Promise<any>
    function sendMessage(
        extensionId: string,
        message: any,
        options?: { includeTlsChannelId?: boolean; toProxyScript?: boolean }
    ): Promise<any>

    function sendNativeMessage(application: string, message: object): Promise<object | void>
    function getPlatformInfo(): Promise<PlatformInfo>
    function getBrowserInfo(): Promise<{
        name: string
        vendor: string
        version: string
        buildID: string
    }>
    // Unsupported: https://bugzilla.mozilla.org/show_bug.cgi?id=1339407
    // function getPackageDirectoryEntry(): Promise<any>;

    const onStartup: EventEmitter<void>
    const onInstalled: EventEmitter<{
        reason: OnInstalledReason
        previousVersion?: string
        id?: string
    }>
    // Unsupported
    // const onSuspend: Listener<void>;
    // const onSuspendCanceled: Listener<void>;
    // const onBrowserUpdateAvailable: Listener<void>;
    // const onRestartRequired: Listener<OnRestartRequiredReason>;
    const onUpdateAvailable: EventEmitter<{ version: string }>
    const onConnect: EventEmitter<PortWithSender>

    const onConnectExternal: EventEmitter<PortWithSender>

    type OnMessageHandler = (message: any, sender: MessageSender) => void | Promise<any>

    const onMessage: CallbackEventEmitter<OnMessageHandler>

    const onMessageExternal: CallbackEventEmitter<OnMessageHandler>
}

declare namespace browser.sessions {
    interface Filter {
        maxResults?: number
    }

    interface Session {
        lastModified: number
        tab: tabs.Tab
        window: windows.Window
    }

    const MAX_SESSION_RESULTS: number

    function getRecentlyClosed(filter?: Filter): Promise<Session[]>

    function restore(sessionId: string): Promise<Session>

    function setTabValue(tabId: number, key: string, value: string | object): Promise<void>

    function getTabValue(tabId: number, key: string): Promise<void | string | object>

    function removeTabValue(tabId: number, key: string): Promise<void>

    function setWindowValue(windowId: number, key: string, value: string | object): Promise<void>

    function getWindowValue(windowId: number, key: string): Promise<void | string | object>

    function removeWindowValue(windowId: number, key: string): Promise<void>

    const onChanged: CallbackEventEmitter<() => void>
}

declare namespace browser.sidebarAction {
    type ImageDataType = ImageData

    function setPanel(details: { panel: string; tabId?: number }): void

    function getPanel(details: { tabId?: number }): Promise<string>

    function setTitle(details: { title: string; tabId?: number }): void

    function getTitle(details: { tabId?: number }): Promise<string>

    interface IconViaPath {
        path: string | { [index: number]: string }
        tabId?: number
    }

    interface IconViaImageData {
        imageData: ImageDataType | { [index: number]: ImageDataType }
        tabId?: number
    }

    function setIcon(details: IconViaPath | IconViaImageData): Promise<void>

    function open(): Promise<void>

    function close(): Promise<void>
}

declare namespace browser.storage {
    /**
     * Example for type-safe usage:
     *
     * ```ts
     * interface MyStorageItems {
     *  foo: number
     * }
     *
     * (browser.storage.sync as browser.storage.StorageArea<MyStorageItems>).get('foo')
     * ```
     */
    interface StorageArea<T extends object = any> {
        get(): Promise<Partial<T>>
        get<K extends keyof T>(keys: K[] | K): Promise<{ [k in K]?: T[k] }>

        /**
         * Stores one or more items in the storage area, or update existing items.
         *
         * When you store or update a value using this API, the storage.onChanged event will fire.
         *
         * @param items An object containing one or more key/value pairs to be stored in storage. If an item already exists, its value will be updated.
         * Values may be primitive types (such as numbers, booleans, and strings) or Array types.
         * It's generally not possible to store other types, such as Function, Date, RegExp, Set, Map, ArrayBuffer and so on. Some of these unsupported types will restore as an empty object, and some cause set() to throw an error. The exact behavior here is browser-specific.
         * If a value is `undefined`, it will not be changed.
         * If a value is `null`, it will be set to `null`.
         */
        set(items: Partial<T>): Promise<void>
        remove(keys: keyof T | (keyof T)[]): Promise<void>
        clear(): Promise<void>

        // unsupported: getBytesInUse: (keys: string|string[]|null) => Promise<number>,
    }

    interface StorageChange<T> {
        oldValue?: T
        newValue?: T
    }

    const sync: StorageArea
    const local: StorageArea
    const managed: StorageArea

    type ChangeDict<T> = { [K in keyof T]?: StorageChange<T[K]> }
    type AreaName = 'sync' | 'local' | 'managed'

    const onChanged: CallbackEventEmitter<(changes: ChangeDict<any>, areaName: AreaName) => void>
}

declare namespace browser.tabs {
    type MutedInfoReason = 'capture' | 'extension' | 'user'
    interface MutedInfo {
        muted: boolean
        extensionId?: string
        reason: MutedInfoReason
    }
    // TODO: Specify PageSettings properly.
    type PageSettings = object
    interface Tab {
        active: boolean
        audible?: boolean
        autoDiscardable?: boolean
        cookieStoreId?: string
        discarded?: boolean
        favIconUrl?: string
        height?: number
        hidden: boolean
        highlighted: boolean
        id?: number
        incognito: boolean
        index: number
        isArticle: boolean
        isInReaderMode: boolean
        lastAccessed: number
        mutedInfo?: MutedInfo
        openerTabId?: number
        pinned: boolean
        selected: boolean
        sessionId?: string
        status?: string
        title?: string
        url?: string
        width?: number
        windowId: number
    }

    type TabStatus = 'loading' | 'complete'
    type WindowType = 'normal' | 'popup' | 'panel' | 'devtools'
    type ZoomSettingsMode = 'automatic' | 'disabled' | 'manual'
    type ZoomSettingsScope = 'per-origin' | 'per-tab'
    interface ZoomSettings {
        defaultZoomFactor?: number
        mode?: ZoomSettingsMode
        scope?: ZoomSettingsScope
    }

    const TAB_ID_NONE: number

    function connect(tabId: number, connectInfo?: { name?: string; frameId?: number }): runtime.Port
    function create(createProperties: {
        active?: boolean
        cookieStoreId?: string
        index?: number
        openerTabId?: number
        pinned?: boolean
        // deprecated: selected: boolean,
        url?: string
        windowId?: number
    }): Promise<Tab>
    function captureTab(tabId?: number, options?: extensionTypes.ImageDetails): Promise<string>
    function captureVisibleTab(windowId?: number, options?: extensionTypes.ImageDetails): Promise<string>
    function detectLanguage(tabId?: number): Promise<string>
    function duplicate(tabId: number): Promise<Tab>
    function executeScript(tabId: number | undefined, details: extensionTypes.InjectDetails): Promise<object[]>
    function get(tabId: number): Promise<Tab>
    // deprecated: function getAllInWindow(): x;
    function getCurrent(): Promise<Tab>
    // deprecated: function getSelected(windowId?: number): Promise<browser.tabs.Tab>;
    function getZoom(tabId?: number): Promise<number>
    function getZoomSettings(tabId?: number): Promise<ZoomSettings>
    function hide(tabIds: number | number[]): Promise<number[]>
    // unsupported: function highlight(highlightInfo: {
    //     windowId?: number,
    //     tabs: number[]|number,
    // }): Promise<browser.windows.Window>;
    function insertCSS(tabId: number | undefined, details: extensionTypes.InjectDetailsCSS): Promise<void>
    function removeCSS(tabId: number | undefined, details: extensionTypes.InjectDetails): Promise<void>
    function move(
        tabIds: number | number[],
        moveProperties: {
            windowId?: number
            index: number
        }
    ): Promise<Tab | Tab[]>
    function print(): Promise<void>
    function printPreview(): Promise<void>
    function query(queryInfo: {
        active?: boolean
        audible?: boolean
        // unsupported: autoDiscardable?: boolean,
        cookieStoreId?: string
        currentWindow?: boolean
        discarded?: boolean
        hidden?: boolean
        highlighted?: boolean
        index?: number
        muted?: boolean
        lastFocusedWindow?: boolean
        pinned?: boolean
        status?: TabStatus
        title?: string
        url?: string | string[]
        windowId?: number
        windowType?: WindowType
    }): Promise<Tab[]>
    function reload(tabId?: number, reloadProperties?: { bypassCache?: boolean }): Promise<void>
    function remove(tabIds: number | number[]): Promise<void>
    function saveAsPDF(
        pageSettings: PageSettings
    ): Promise<'saved' | 'replaced' | 'canceled' | 'not_saved' | 'not_replaced'>
    function sendMessage<T = any, U = object>(
        tabId: number,
        message: T,
        options?: { frameId?: number }
    ): Promise<U | void>
    // deprecated: function sendRequest(): x;
    function setZoom(tabId: number | undefined, zoomFactor: number): Promise<void>
    function setZoomSettings(tabId: number | undefined, zoomSettings: ZoomSettings): Promise<void>
    function show(tabIds: number | number[]): Promise<void>
    function toggleReaderMode(tabId?: number): Promise<void>

    interface UpdateProperties {
        active?: boolean
        // unsupported: autoDiscardable?: boolean,
        // unsupported: highlighted?: boolean,
        // unsupported: hidden?: boolean;
        loadReplace?: boolean
        muted?: boolean
        openerTabId?: number
        pinned?: boolean
        // deprecated: selected?: boolean,
        url?: string
    }
    function update(tabId: number | undefined, updateProperties: UpdateProperties): Promise<Tab>
    function update(updateProperties: UpdateProperties): Promise<Tab>

    const onActivated: EventEmitter<{ tabId: number; windowId: number }>
    const onAttached: CallbackEventEmitter<(
        tabId: number,
        attachInfo: {
            newWindowId: number
            newPosition: number
        }
    ) => void>
    const onCreated: EventEmitter<Tab>
    const onDetached: CallbackEventEmitter<(
        tabId: number,
        detachInfo: {
            oldWindowId: number
            oldPosition: number
        }
    ) => void>
    const onHighlighted: EventEmitter<{ windowId: number; tabIds: number[] }>
    const onMoved: CallbackEventEmitter<(
        tabId: number,
        moveInfo: {
            windowId: number
            fromIndex: number
            toIndex: number
        }
    ) => void>
    const onRemoved: CallbackEventEmitter<(
        tabId: number,
        removeInfo: {
            windowId: number
            isWindowClosing: boolean
        }
    ) => void>
    const onReplaced: CallbackEventEmitter<(addedTabId: number, removedTabId: number) => void>
    const onUpdated: CallbackEventEmitter<(
        tabId: number,
        changeInfo: {
            audible?: boolean
            discarded?: boolean
            favIconUrl?: string
            mutedInfo?: MutedInfo
            pinned?: boolean
            status?: string
            title?: string
            url?: string
        },
        tab: Tab
    ) => void>
    const onZoomChanged: EventEmitter<{
        tabId: number
        oldZoomFactor: number
        newZoomFactor: number
        zoomSettings: ZoomSettings
    }>
}

declare namespace browser.topSites {
    interface MostVisitedURL {
        title: string
        url: string
    }
    function get(): Promise<MostVisitedURL[]>
}

declare namespace browser.webNavigation {
    type TransitionType = 'link' | 'auto_subframe' | 'form_submit' | 'reload'
    // unsupported: | "typed" | "auto_bookmark" | "manual_subframe"
    //              | "generated" | "start_page" | "keyword"
    //              | "keyword_generated";

    type TransitionQualifier = 'client_redirect' | 'server_redirect' | 'forward_back'
    // unsupported: "from_address_bar";

    function getFrame(details: {
        tabId: number
        processId: number
        frameId: number
    }): Promise<{ errorOccured: boolean; url: string; parentFrameId: number }>

    function getAllFrames(details: {
        tabId: number
    }): Promise<
        {
            errorOccured: boolean
            processId: number
            frameId: number
            parentFrameId: number
            url: string
        }[]
    >

    interface NavListener<T> {
        addListener: (
            callback: (arg: T) => void,
            filter?: {
                url: events.UrlFilter[]
            }
        ) => void
        removeListener: (callback: (arg: T) => void) => void
        hasListener: (callback: (arg: T) => void) => boolean
    }

    type DefaultNavListener = NavListener<{
        tabId: number
        url: string
        processId: number
        frameId: number
        timeStamp: number
    }>

    type TransitionNavListener = NavListener<{
        tabId: number
        url: string
        processId: number
        frameId: number
        timeStamp: number
        transitionType: TransitionType
        transitionQualifiers: TransitionQualifier[]
    }>

    const onBeforeNavigate: NavListener<{
        tabId: number
        url: string
        processId: number
        frameId: number
        parentFrameId: number
        timeStamp: number
    }>

    const onCommitted: TransitionNavListener

    const onCreatedNavigationTarget: NavListener<{
        sourceFrameId: number
        // Unsupported: sourceProcessId: number,
        sourceTabId: number
        tabId: number
        timeStamp: number
        url: string
        windowId: number
    }>

    const onDOMContentLoaded: DefaultNavListener

    const onCompleted: DefaultNavListener

    const onErrorOccurred: DefaultNavListener // error field unsupported

    const onReferenceFragmentUpdated: TransitionNavListener

    const onHistoryStateUpdated: TransitionNavListener
}

declare namespace browser.webRequest {
    type ResourceType =
        | 'main_frame'
        | 'sub_frame'
        | 'stylesheet'
        | 'script'
        | 'image'
        | 'object'
        | 'xmlhttprequest'
        | 'xbl'
        | 'xslt'
        | 'ping'
        | 'beacon'
        | 'xml_dtd'
        | 'font'
        | 'media'
        | 'websocket'
        | 'csp_report'
        | 'imageset'
        | 'web_manifest'
        | 'other'

    interface RequestFilter {
        urls: string[]
        types?: ResourceType[]
        tabId?: number
        windowId?: number
    }

    interface StreamFilter {
        onstart: (event: any) => void
        ondata: (event: { data: ArrayBuffer }) => void
        onstop: (event: any) => void
        onerror: (event: any) => void

        close(): void
        disconnect(): void
        resume(): void
        suspend(): void
        write(data: Uint8Array | ArrayBuffer): void

        error: string
        status:
            | 'uninitialized'
            | 'transferringdata'
            | 'finishedtransferringdata'
            | 'suspended'
            | 'closed'
            | 'disconnected'
            | 'failed'
    }

    type HttpHeaders = (
        | { name: string; binaryValue: number[]; value?: string }
        | { name: string; value: string; binaryValue?: number[] }
    )[]

    interface BlockingResponse {
        cancel?: boolean
        redirectUrl?: string
        requestHeaders?: HttpHeaders
        responseHeaders?: HttpHeaders
        authCredentials?: { username: string; password: string }
    }

    interface UploadData {
        bytes?: ArrayBuffer
        file?: string
    }

    const MAX_HANDLER_BEHAVIOR_CHANGED_CALLS_PER_10_MINUTES: number

    function handlerBehaviorChanged(): Promise<void>

    // TODO: Enforce the return result of the addListener call in the contract
    //       Use an intersection type for all the default properties
    interface ReqListener<T, U> {
        addListener: (
            callback: (arg: T) => void,
            filter: RequestFilter,
            extraInfoSpec?: U[]
        ) => BlockingResponse | Promise<BlockingResponse>
        removeListener: (callback: (arg: T) => void) => void
        hasListener: (callback: (arg: T) => void) => boolean
    }

    const onBeforeRequest: ReqListener<
        {
            requestId: string
            url: string
            method: string
            frameId: number
            parentFrameId: number
            requestBody?: {
                error?: string
                formData?: { [key: string]: string[] }
                raw?: UploadData[]
            }
            tabId: number
            type: ResourceType
            timeStamp: number
            originUrl: string
        },
        'blocking' | 'requestBody'
    >

    const onBeforeSendHeaders: ReqListener<
        {
            requestId: string
            url: string
            method: string
            frameId: number
            parentFrameId: number
            tabId: number
            type: ResourceType
            timeStamp: number
            originUrl: string
            requestHeaders?: HttpHeaders
        },
        'blocking' | 'requestHeaders'
    >

    const onSendHeaders: ReqListener<
        {
            requestId: string
            url: string
            method: string
            frameId: number
            parentFrameId: number
            tabId: number
            type: ResourceType
            timeStamp: number
            originUrl: string
            requestHeaders?: HttpHeaders
        },
        'requestHeaders'
    >

    const onHeadersReceived: ReqListener<
        {
            requestId: string
            url: string
            method: string
            frameId: number
            parentFrameId: number
            tabId: number
            type: ResourceType
            timeStamp: number
            originUrl: string
            statusLine: string
            responseHeaders?: HttpHeaders
            statusCode: number
        },
        'blocking' | 'responseHeaders'
    >

    const onAuthRequired: ReqListener<
        {
            requestId: string
            url: string
            method: string
            frameId: number
            parentFrameId: number
            tabId: number
            type: ResourceType
            timeStamp: number
            scheme: string
            realm?: string
            challenger: { host: string; port: number }
            isProxy: boolean
            responseHeaders?: HttpHeaders
            statusLine: string
            statusCode: number
        },
        'blocking' | 'responseHeaders'
    >

    const onResponseStarted: ReqListener<
        {
            requestId: string
            url: string
            method: string
            frameId: number
            parentFrameId: number
            tabId: number
            type: ResourceType
            timeStamp: number
            originUrl: string
            ip?: string
            fromCache: boolean
            statusLine: string
            responseHeaders?: HttpHeaders
            statusCode: number
        },
        'responseHeaders'
    >

    const onBeforeRedirect: ReqListener<
        {
            requestId: string
            url: string
            method: string
            frameId: number
            parentFrameId: number
            tabId: number
            type: ResourceType
            timeStamp: number
            originUrl: string
            ip?: string
            fromCache: boolean
            statusCode: number
            redirectUrl: string
            statusLine: string
            responseHeaders?: HttpHeaders
        },
        'responseHeaders'
    >

    const onCompleted: ReqListener<
        {
            requestId: string
            url: string
            method: string
            frameId: number
            parentFrameId: number
            tabId: number
            type: ResourceType
            timeStamp: number
            originUrl: string
            ip?: string
            fromCache: boolean
            statusCode: number
            statusLine: string
            responseHeaders?: HttpHeaders
        },
        'responseHeaders'
    >

    const onErrorOccurred: ReqListener<
        {
            requestId: string
            url: string
            method: string
            frameId: number
            parentFrameId: number
            tabId: number
            type: ResourceType
            timeStamp: number
            originUrl: string
            ip?: string
            fromCache: boolean
            error: string
        },
        void
    >

    function filterResponseData(requestId: string): StreamFilter
}

declare namespace browser.windows {
    type WindowType = 'normal' | 'popup' | 'panel' | 'devtools'

    type WindowState = 'normal' | 'minimized' | 'maximized' | 'fullscreen' | 'docked'

    interface Window {
        id?: number
        focused: boolean
        top?: number
        left?: number
        width?: number
        height?: number
        tabs?: tabs.Tab[]
        incognito: boolean
        type?: WindowType
        state?: WindowState
        alwaysOnTop: boolean
        sessionId?: string
    }

    type CreateType = 'normal' | 'popup' | 'panel' | 'detached_panel'

    const WINDOW_ID_NONE: number

    const WINDOW_ID_CURRENT: number

    function get(
        windowId: number,
        getInfo?: {
            populate?: boolean
            windowTypes?: WindowType[]
        }
    ): Promise<Window>

    function getCurrent(getInfo?: { populate?: boolean; windowTypes?: WindowType[] }): Promise<Window>

    function getLastFocused(getInfo?: { populate?: boolean; windowTypes?: WindowType[] }): Promise<Window>

    function getAll(getInfo?: { populate?: boolean; windowTypes?: WindowType[] }): Promise<Window[]>

    // TODO: url and tabId should be exclusive
    function create(createData?: {
        allowScriptsToClose?: boolean
        url?: string | string[]
        tabId?: number
        left?: number
        top?: number
        width?: number
        height?: number
        // unsupported: focused?: boolean,
        incognito?: boolean
        titlePreface?: string
        type?: CreateType
        state?: WindowState
    }): Promise<Window>

    function update(
        windowId: number,
        updateInfo: {
            left?: number
            top?: number
            width?: number
            height?: number
            focused?: boolean
            drawAttention?: boolean
            state?: WindowState
        }
    ): Promise<Window>

    function remove(windowId: number): Promise<void>

    const onCreated: EventEmitter<Window>

    const onRemoved: EventEmitter<number>

    const onFocusChanged: EventEmitter<number>
}

declare namespace browser.theme {
    interface Theme {
        images: ThemeImages
        colors: ThemeColors
        properties?: ThemeProperties
    }

    interface ThemeImages {
        headerURL: string
        theme_frame?: string
        additional_backgrounds?: string[]
    }

    interface ThemeColors {
        accentcolor: string
        textcolor: string
        frame?: [number, number, number]
        tab_text?: [number, number, number]
        toolbar?: string
        toolbar_text?: string
        toolbar_field?: string
        toolbar_field_text?: string
    }

    interface ThemeProperties {
        additional_backgrounds_alignment: Alignment[]
        additional_backgrounds_tiling: Tiling[]
    }

    type Alignment =
        | 'bottom'
        | 'center'
        | 'left'
        | 'right'
        | 'top'
        | 'center bottom'
        | 'center center'
        | 'center top'
        | 'left bottom'
        | 'left center'
        | 'left top'
        | 'right bottom'
        | 'right center'
        | 'right top'

    type Tiling = 'no-repeat' | 'repeat' | 'repeat-x' | 'repeat-y'

    function getCurrent(windowId?: number): Promise<Theme>
    function update(theme: Theme): Promise<void>
    function update(windowId: number, theme: Theme): Promise<void>
    function reset(windowId?: number): Promise<void>
}
