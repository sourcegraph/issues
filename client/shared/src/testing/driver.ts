import expect from 'expect'
import { percySnapshot as realPercySnapshot } from '@percy/puppeteer'
import * as jsonc from '@sqs/jsonc-parser'
import * as jsoncEdit from '@sqs/jsonc-parser/lib/edit'
import * as os from 'os'
import puppeteer, {
    PageEventObj,
    Page,
    Serializable,
    LaunchOptions,
    PageFnOptions,
    ConsoleMessage,
    Target,
} from 'puppeteer'
import { Key } from 'ts-key-enum'
import { dataOrThrowErrors, gql, GraphQLResult } from '../graphql/graphql'
import { IMutation, IQuery, IRepository } from '../graphql/schema'
import { readEnvironmentBoolean, retry } from './utils'
import { formatPuppeteerConsoleMessage } from './console'
import * as path from 'path'
import { escapeRegExp } from 'lodash'
import { readFile, appendFile, mkdir } from 'mz/fs'
import { Settings } from '../settings/settings'
import { from, fromEvent, merge, Subscription } from 'rxjs'
import { filter, map, concatAll, mergeMap, mergeAll, takeUntil } from 'rxjs/operators'
import getFreePort from 'get-port'
import puppeteerFirefox from 'puppeteer-firefox'
import webExt from 'web-ext'
import { isDefined } from '../util/types'
import { getConfig } from './config'
import { ExternalServiceKind } from '../graphql-operations'
import delay from 'delay'

/**
 * Returns a Promise for the next emission of the given event on the given Puppeteer page.
 */
export const oncePageEvent = <E extends keyof PageEventObj>(page: Page, eventName: E): Promise<PageEventObj[E]> =>
    new Promise(resolve => page.once(eventName, resolve))

export const percySnapshot = readEnvironmentBoolean({ variable: 'PERCY_ON', defaultValue: false })
    ? realPercySnapshot
    : () => Promise.resolve()

export const BROWSER_EXTENSION_DEV_ID = 'bmfbcejdknlknpncfpeloejonjoledha'

/**
 * Specifies how to select the content of the element. No
 * single method works in all cases:
 *
 * - Meta+A doesn't work in input boxes https://github.com/GoogleChrome/puppeteer/issues/1313
 * - selectall doesn't work in the Monaco editor
 */
type SelectTextMethod = 'selectall' | 'keyboard'

/**
 * Specifies how to enter text. Typing is preferred in cases where it's important to test
 * the process of manually typing out the text to enter. Pasting is preferred in cases
 * where typing would be too slow or we explicitly want to test paste behavior.
 */
type EnterTextMethod = 'type' | 'paste'

interface FindElementOptions {
    /**
     * Filter candidate elements to those with the specified CSS selector
     */
    selector?: string

    /**
     * Log the XPath quer(y|ies) used to find the element.
     */
    log?: boolean

    /**
     * Specifies how exact the search criterion is.
     */
    fuzziness?: 'exact' | 'prefix' | 'space-prefix' | 'contains'

    /**
     * Specifies whether to wait (and how long) for the element to appear.
     */
    wait?: PageFnOptions | boolean
}

function findElementRegexpStrings(
    text: string,
    { fuzziness = 'space-prefix' }: Pick<FindElementOptions, 'fuzziness'>
): string[] {
    //  Escape regexp special chars. Copied from
    //  https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Regular_Expressions
    const escapedText = escapeRegExp(text)
    const regexps = [`^${escapedText}$`]
    if (fuzziness === 'exact') {
        return regexps
    }
    regexps.push(`^${escapedText}\\b`)
    if (fuzziness === 'prefix') {
        return regexps
    }
    regexps.push(`^\\s+${escapedText}$`) // Still prefer exact
    regexps.push(`^\\s+${escapedText}\\b`)
    if (fuzziness === 'space-prefix') {
        return regexps
    }
    regexps.push(escapedText)
    return regexps
}

function findElementMatchingRegexps(tag: string, regexps: string[]): HTMLElement | null {
    for (const regexpString of regexps) {
        const regexp = new RegExp(regexpString)
        for (const element of document.querySelectorAll<HTMLElement>(tag)) {
            if (!element.offsetParent) {
                // Ignore hidden elements
                continue
            }
            if (element.textContent?.match(regexp)) {
                return element
            }
        }
    }
    return null
}

function getDebugExpressionFromRegexp(tag: string, regexp: string): string {
    return `Array.from(document.querySelectorAll(${JSON.stringify(
        tag
    )})).filter(e => e.innerText && e.innerText.match(/${regexp}/))`
}

export class Driver {
    /** The pages that were visited since the creation of the driver. */
    public visitedPages: Readonly<URL>[] = []

    public sourcegraphBaseUrl: string
    private keepBrowser: boolean
    private subscriptions = new Subscription()

    constructor(public browser: puppeteer.Browser, public page: puppeteer.Page, options: DriverOptions) {
        this.sourcegraphBaseUrl = options.sourcegraphBaseUrl
        this.keepBrowser = !!options.keepBrowser

        // Record visited pages
        this.subscriptions.add(
            merge(fromEvent<Target>(browser, 'targetchanged'), fromEvent<Target>(browser, 'targetcreated'))
                .pipe(filter(target => target.type() === 'page'))
                .subscribe(target => {
                    this.visitedPages.push(new URL(target.url()))
                })
        )

        // Log browser console
        if (options.logBrowserConsole) {
            this.subscriptions.add(
                merge(
                    from(browser.pages()).pipe(mergeAll()),
                    fromEvent<Target>(browser, 'targetcreated').pipe(
                        mergeMap(target => target.page()),
                        filter(isDefined)
                    )
                )
                    .pipe(
                        mergeMap(page =>
                            fromEvent<ConsoleMessage>(page, 'console').pipe(
                                filter(
                                    message =>
                                        !message.text().includes('Download the React DevTools') &&
                                        !message.text().includes('[HMR]') &&
                                        !message.text().includes('[WDS]') &&
                                        !message
                                            .text()
                                            .includes('Warning: componentWillReceiveProps has been renamed') &&
                                        !message.text().includes('React-Hot-Loader') &&
                                        // These requests are expected to fail, we use them to check if the browser extension is installed.
                                        message.location().url !== 'chrome-extension://invalid/'
                                ),
                                // Immediately format remote handles to strings, but maintain order.
                                map(message => formatPuppeteerConsoleMessage(page, message)),
                                concatAll(),
                                takeUntil(fromEvent(page, 'close'))
                            )
                        )
                    )
                    .subscribe(formattedLine => console.log(formattedLine))
            )
        }
    }

    public async ensureLoggedIn({
        username,
        password,
        email,
    }: {
        username: string
        password: string
        email?: string
    }): Promise<void> {
        await this.page.goto(this.sourcegraphBaseUrl)
        await this.page.evaluate(() => {
            localStorage.setItem('has-dismissed-browser-ext-toast', 'true')
            localStorage.setItem('has-dismissed-integrations-toast', 'true')
            localStorage.setItem('has-dismissed-survey-toast', 'true')
        })
        const url = new URL(this.page.url())
        if (url.pathname === '/site-admin/init') {
            await this.page.waitForSelector('.test-signup-form')
            if (email) {
                await this.page.type('input[name=email]', email)
            }
            await this.page.type('input[name=username]', username)
            await this.page.type('input[name=password]', password)
            await this.page.waitForSelector('button[type=submit]:not(:disabled)')
            // TODO(uwedeportivo): investigate race condition between puppeteer clicking this very fast and
            // background gql client request fetching ViewerSettings. this race condition results in the gql request
            // "winning" sometimes without proper credentials which confuses the login state machine and it navigates
            // you back to the login page
            await delay(1000)
            await this.page.click('button[type=submit]')
            await this.page.waitForNavigation({ timeout: 3 * 10000 })
        } else if (url.pathname === '/sign-in') {
            await this.page.waitForSelector('.test-signin-form')
            await this.page.type('input', username)
            await this.page.type('input[name=password]', password)
            // TODO(uwedeportivo): see comment above, same reason
            await delay(1000)
            await this.page.click('button[type=submit]')
            await this.page.waitForNavigation({ timeout: 3 * 10000 })
        }
    }

    /**
     * Navigates to the Sourcegraph browser extension options page and sets the sourcegraph URL.
     */
    public async setExtensionSourcegraphUrl(): Promise<void> {
        await this.page.goto(`chrome-extension://${BROWSER_EXTENSION_DEV_ID}/options.html`)
        await this.page.waitForSelector('.test-sourcegraph-url')
        await this.replaceText({ selector: '.test-sourcegraph-url', newText: this.sourcegraphBaseUrl })
        await this.page.keyboard.press(Key.Enter)
        await this.page.waitForFunction(
            () => {
                const element = document.querySelector('.test-connection-status')
                return element?.textContent?.includes('Connected')
            },
            { timeout: 5000 }
        )
    }

    public async close(): Promise<void> {
        this.subscriptions.unsubscribe()
        if (!this.keepBrowser) {
            await this.browser.close()
        }
        console.log(
            '\nVisited routes:\n' +
                [
                    ...new Set(
                        this.visitedPages
                            .filter(url => url.href.startsWith(this.sourcegraphBaseUrl))
                            .map(url => url.pathname)
                    ),
                ].join('\n')
        )
    }

    public async newPage(): Promise<void> {
        this.page = await this.browser.newPage()
    }

    public async selectAll(method: SelectTextMethod = 'selectall'): Promise<void> {
        switch (method) {
            case 'selectall': {
                await this.page.evaluate(() => document.execCommand('selectall', false))
                break
            }
            case 'keyboard': {
                const modifier = os.platform() === 'darwin' ? Key.Meta : Key.Control
                await this.page.keyboard.down(modifier)
                await this.page.keyboard.press('a')
                await this.page.keyboard.up(modifier)
                break
            }
        }
    }

    public async enterText(method: EnterTextMethod = 'type', text: string): Promise<void> {
        // Pasting does not work on macOS. See:  https://github.com/GoogleChrome/puppeteer/issues/1313
        method = os.platform() === 'darwin' ? 'type' : method
        switch (method) {
            case 'type':
                await this.page.keyboard.type(text)
                break
            case 'paste':
                await this.paste(text)
                break
        }
    }

    public async replaceText({
        selector,
        newText,
        selectMethod = 'selectall',
        enterTextMethod = 'type',
    }: {
        selector: string
        newText: string
        selectMethod?: SelectTextMethod
        enterTextMethod?: EnterTextMethod
    }): Promise<void> {
        // The Monaco editor sometimes detaches nodes from the DOM, causing
        // `click()` to fail unpredictably.
        await retry(async () => {
            await this.page.waitForSelector(selector)
            await this.page.click(selector)
        })
        await this.selectAll(selectMethod)
        await this.page.keyboard.press(Key.Backspace)
        await this.enterText(enterTextMethod, newText)
    }

    public async acceptNextDialog(): Promise<void> {
        const dialog = await oncePageEvent(this.page, 'dialog')
        await dialog.accept()
    }

    public async ensureHasExternalService({
        kind,
        displayName,
        config,
        ensureRepos,
        alwaysCloning,
    }: {
        kind: ExternalServiceKind
        displayName: string
        config: string
        ensureRepos?: string[]
        alwaysCloning?: string[]
    }): Promise<void> {
        // Use the graphQL API to query external services on the instance.
        const { externalServices } = dataOrThrowErrors(
            await this.makeGraphQLRequest<IQuery>({
                request: gql`
                    query ExternalServicesForTests {
                        externalServices(first: 1) {
                            totalCount
                        }
                    }
                `,
                variables: {},
            })
        )
        // Delete existing external services if there are any.
        if (externalServices.totalCount !== 0) {
            await this.page.goto(this.sourcegraphBaseUrl + '/site-admin/external-services')
            await this.page.waitFor('.test-filtered-connection')
            await this.page.waitForSelector('.test-filtered-connection__loader', { hidden: true })

            // Matches buttons for deleting external services named ${displayName}.
            const deleteButtonSelector = `[data-test-external-service-name="${displayName}"] .test-delete-external-service-button`
            if (await this.page.$(deleteButtonSelector)) {
                await Promise.all([this.acceptNextDialog(), this.page.click(deleteButtonSelector)])
            }
        }

        // Navigate to the add external service page.
        console.log('Adding external service of kind', kind)
        await this.page.goto(this.sourcegraphBaseUrl + '/site-admin/external-services/new')
        await this.page.waitForSelector(`[data-test-external-service-card-link="${kind.toUpperCase()}"]`, {
            visible: true,
        })
        await this.page.evaluate((selector: string) => {
            const element = document.querySelector<HTMLElement>(selector)
            if (!element) {
                throw new Error(`Could not find element to click on for selector ${selector}`)
            }
            element.click()
        }, `[data-test-external-service-card-link="${kind.toUpperCase()}"]`)
        await this.replaceText({
            selector: '#test-external-service-form-display-name',
            newText: displayName,
        })

        await this.page.waitForSelector('.test-external-service-editor .monaco-editor')
        // Type in a new external service configuration.
        await this.replaceText({
            selector: '.test-external-service-editor .monaco-editor .view-line',
            newText: config,
            selectMethod: 'keyboard',
        })
        await Promise.all([this.page.waitForNavigation(), this.page.click('.test-add-external-service-button')])

        if (ensureRepos) {
            // Clone the repositories
            for (const slug of ensureRepos) {
                await this.page.goto(
                    this.sourcegraphBaseUrl + `/site-admin/repositories?filter=cloned&query=${encodeURIComponent(slug)}`
                )
                await this.page.waitForSelector(`.repository-node[data-test-repository='${slug}']`, { visible: true })
                // Workaround for https://github.com/sourcegraph/sourcegraph/issues/5286
                await this.page.goto(`${this.sourcegraphBaseUrl}/${slug}`)
            }
        }

        if (alwaysCloning) {
            for (const slug of alwaysCloning) {
                await this.page.goto(
                    this.sourcegraphBaseUrl +
                        `/site-admin/repositories?filter=cloning&query=${encodeURIComponent(slug)}`
                )
                await this.page.waitForSelector(`.repository-node[data-test-repository='${slug}']`, { visible: true })
                // Workaround for https://github.com/sourcegraph/sourcegraph/issues/5286
                await this.page.goto(`${this.sourcegraphBaseUrl}/${slug}`)
            }
        }
    }

    public async paste(value: string): Promise<void> {
        await this.page.evaluate(async (value: string) => {
            await navigator.clipboard.writeText(value)
        }, value)
        const modifier = os.platform() === 'darwin' ? Key.Meta : Key.Control
        await this.page.keyboard.down(modifier)
        await this.page.keyboard.press('v')
        await this.page.keyboard.up(modifier)
    }

    public async assertWindowLocation(location: string, isAbsolute = false): Promise<any> {
        const url = isAbsolute ? location : this.sourcegraphBaseUrl + location
        await retry(async () => {
            expect(await this.page.evaluate(() => window.location.href)).toEqual(url)
        })
    }

    public async assertWindowLocationPrefix(locationPrefix: string, isAbsolute = false): Promise<any> {
        const prefix = isAbsolute ? locationPrefix : this.sourcegraphBaseUrl + locationPrefix
        await retry(async () => {
            const location: string = await this.page.evaluate(() => window.location.href)
            expect(location.startsWith(prefix)).toBeTruthy()
        })
    }

    public async assertStickyHighlightedToken(label: string): Promise<void> {
        await this.page.waitForSelector('.selection-highlight-sticky', { visible: true }) // make sure matched token is highlighted
        await retry(async () =>
            expect(
                await this.page.evaluate(() => document.querySelector('.selection-highlight-sticky')!.textContent)
            ).toEqual(label)
        )
    }

    public async assertAllHighlightedTokens(label: string): Promise<void> {
        const highlightedTokens = await this.page.evaluate(() =>
            [...document.querySelectorAll('.selection-highlight')].map(element => element.textContent || '')
        )
        expect(highlightedTokens.every(txt => txt === label)).toBeTruthy()
    }

    public async assertNonemptyLocalRefs(): Promise<void> {
        // verify active group is references
        await this.page.waitForXPath(
            "//*[contains(@class, 'panel__tabs')]//*[contains(@class, 'tab-bar__tab--active') and contains(text(), 'References')]"
        )
        // verify there are some references
        await this.page.waitForSelector('.panel__tabs-content .file-match-children__item', { visible: true })
    }

    public async assertNonemptyExternalRefs(): Promise<void> {
        // verify active group is references
        await this.page.waitForXPath(
            "//*[contains(@class, 'panel__tabs')]//*[contains(@class, 'tab-bar__tab--active') and contains(text(), 'References')]"
        )
        // verify there are some references
        await this.page.waitForSelector('.panel__tabs-content .hierarchical-locations-view__item', { visible: true })
    }

    private async makeRequest<T = void>({ url, init }: { url: string; init: RequestInit & Serializable }): Promise<T> {
        const handle = await this.page.evaluateHandle(
            (url, init) => fetch(url, init).then(response => response.json()),
            url,
            init
        )
        return (await handle.jsonValue()) as T
    }

    private async makeGraphQLRequest<T, V = object>({
        request,
        variables,
    }: {
        request: string
        variables: V
    }): Promise<GraphQLResult<T>> {
        const nameMatch = request.match(/^\s*(?:query|mutation)\s+(\w+)/)
        const xhrHeaders =
            (await this.page.evaluate(
                sourcegraphBaseUrl =>
                    location.href.startsWith(sourcegraphBaseUrl) && (window as any).context.xhrHeaders,
                this.sourcegraphBaseUrl
            )) || {}
        const response = await this.makeRequest<GraphQLResult<T>>({
            url: `${this.sourcegraphBaseUrl}/.api/graphql${nameMatch ? '?' + nameMatch[1] : ''}`,
            init: {
                method: 'POST',
                body: JSON.stringify({ query: request, variables }),
                headers: {
                    ...xhrHeaders,
                    Accept: 'application/json',
                    'Content-Type': 'application/json',
                },
            },
        })
        return response
    }

    public async getRepository(name: string): Promise<Pick<IRepository, 'id'>> {
        const response = await this.makeGraphQLRequest<IQuery>({
            request: gql`
                query($name: String!) {
                    repository(name: $name) {
                        id
                    }
                }
            `,
            variables: { name },
        })
        const { repository } = dataOrThrowErrors(response)
        if (!repository) {
            throw new Error(`repository not found: ${name}`)
        }
        return repository
    }

    public async setConfig(
        path: jsonc.JSONPath,
        editFunction: (oldValue: jsonc.Node | undefined) => any
    ): Promise<void> {
        const currentConfigResponse = await this.makeGraphQLRequest<IQuery>({
            request: gql`
                query SiteForTests {
                    site {
                        id
                        configuration {
                            id
                            effectiveContents
                            validationMessages
                        }
                    }
                }
            `,
            variables: {},
        })
        const { site } = dataOrThrowErrors(currentConfigResponse)
        const currentConfig = site.configuration.effectiveContents
        const newConfig = modifyJSONC(currentConfig, path, editFunction)
        const updateConfigResponse = await this.makeGraphQLRequest<IMutation>({
            request: gql`
                mutation UpdateSiteConfigurationForTests($lastID: Int!, $input: String!) {
                    updateSiteConfiguration(lastID: $lastID, input: $input)
                }
            `,
            variables: { lastID: site.configuration.id, input: newConfig },
        })
        dataOrThrowErrors(updateConfigResponse)
    }

    public async ensureHasCORSOrigin({ corsOriginURL }: { corsOriginURL: string }): Promise<void> {
        await this.setConfig(['corsOrigin'], oldCorsOrigin => {
            const urls = oldCorsOrigin ? (oldCorsOrigin.value as string).split(' ') : []
            return (urls.includes(corsOriginURL) ? urls : [...urls, corsOriginURL]).join(' ')
        })
    }

    public async resetUserSettings(): Promise<void> {
        return this.setUserSettings({})
    }

    public async setUserSettings<S extends Settings>(settings: S): Promise<void> {
        const currentSettingsResponse = await this.makeGraphQLRequest<IQuery>({
            request: gql`
                query UserSettingsForTests {
                    currentUser {
                        id
                        latestSettings {
                            id
                            contents
                        }
                    }
                }
            `,
            variables: {},
        })

        const { currentUser } = dataOrThrowErrors(currentSettingsResponse)
        if (!currentUser) {
            throw new Error('no currentUser')
        }

        const updateConfigResponse = await this.makeGraphQLRequest<IMutation>({
            request: gql`
                mutation OverwriteSettingsForTests($subject: ID!, $lastID: Int, $contents: String!) {
                    settingsMutation(input: { subject: $subject, lastID: $lastID }) {
                        overwriteSettings(contents: $contents) {
                            empty {
                                alwaysNil
                            }
                        }
                    }
                }
            `,
            variables: {
                contents: JSON.stringify(settings),
                subject: currentUser.id,
                lastID: currentUser.latestSettings ? currentUser.latestSettings.id : null,
            },
        })
        dataOrThrowErrors(updateConfigResponse)
    }

    /**
     * Finds the first HTML element matching the text using the regular expressions returned in
     * {@link findElementRegexpStrings}.
     *
     * @param options specifies additional parameters for finding the element. If you want to wait
     * until the element appears, specify the `wait` field (which can contain additional inner
     * options for how long to wait).
     */
    public async findElementWithText(
        text: string,
        options: FindElementOptions & { action?: 'click' } = {}
    ): Promise<puppeteer.ElementHandle<Element>> {
        const { selector: tagName, fuzziness, wait } = options
        const tag = tagName || '*'
        const regexps = findElementRegexpStrings(text, { fuzziness })

        const notFoundError = (underlying?: Error): Error => {
            const debuggingExpressions = regexps.map(regexp => getDebugExpressionFromRegexp(tag, regexp))
            return new Error(
                `Could not find element with text ${JSON.stringify(text)}, options: ${JSON.stringify(options)}` +
                    (underlying ? `. Underlying error was: ${JSON.stringify(underlying.message)}.` : '') +
                    ` Debug expressions: ${debuggingExpressions.join('\n')}`
            )
        }

        return retry(
            async () => {
                const handlePromise = wait
                    ? this.page
                          .waitForFunction(
                              findElementMatchingRegexps,
                              typeof wait === 'object' ? wait : {},
                              tag,
                              regexps
                          )
                          .catch(error => {
                              throw notFoundError(error)
                          })
                    : this.page.evaluateHandle(findElementMatchingRegexps, tag, regexps)

                const element = (await handlePromise).asElement()
                if (!element) {
                    throw notFoundError()
                }

                if (options.action === 'click') {
                    await element.click()
                }
                return element
            },
            {
                retries: options.action === 'click' ? 3 : 0,
                minTimeout: 100,
                maxTimeout: 100,
                factor: 1,
                maxRetryTime: 500,
            }
        )
    }

    public async waitUntilURL(url: string, options: PageFnOptions = {}): Promise<void> {
        await this.page.waitForFunction(url => document.location.href === url, options, url)
    }
}

export function modifyJSONC(
    text: string,
    path: jsonc.JSONPath,
    editFunction: (oldValue: jsonc.Node | undefined) => any
): any {
    const old = jsonc.findNodeAtLocation(jsonc.parseTree(text), path)
    return jsonc.applyEdits(
        text,
        jsoncEdit.setProperty(text, path, editFunction(old), {
            eol: '\n',
            insertSpaces: true,
            tabSize: 2,
        })
    )
}

// Copied from node_modules/puppeteer-firefox/misc/install-preferences.js
async function getFirefoxCfgPath(): Promise<string> {
    const firefoxFolder = path.dirname(puppeteerFirefox.executablePath())
    let configPath: string
    if (process.platform === 'darwin') {
        configPath = path.join(firefoxFolder, '..', 'Resources')
    } else if (process.platform === 'linux') {
        await mkdir(path.join(firefoxFolder, 'browser', 'defaults', 'preferences'), { recursive: true })
        configPath = firefoxFolder
    } else if (process.platform === 'win32') {
        configPath = firefoxFolder
    } else {
        throw new Error('Unsupported platform: ' + process.platform)
    }
    return path.join(configPath, 'puppeteer.cfg')
}

interface DriverOptions extends LaunchOptions {
    browser?: 'chrome' | 'firefox'

    /** If true, load the Sourcegraph browser extension. */
    loadExtension?: boolean

    sourcegraphBaseUrl: string

    /** If not `false`, print browser console messages to stdout. */
    logBrowserConsole?: boolean

    /** If true, keep browser open when driver is closed */
    keepBrowser?: boolean
}

export async function createDriverForTest(options?: DriverOptions): Promise<Driver> {
    // Apply defaults
    options = {
        ...getConfig('sourcegraphBaseUrl', 'headless', 'slowMo', 'keepBrowser', 'browser', 'devtools'),
        ...options,
    }

    const { loadExtension } = options
    const args: string[] = []
    const launchOptions: puppeteer.LaunchOptions = {
        ignoreHTTPSErrors: true,
        ...options,
        args,
        defaultViewport: null,
    }
    let browser: puppeteer.Browser
    if (options.browser === 'firefox') {
        // Make sure CSP is disabled in FF preferences,
        // because Puppeteer uses new Function() to evaluate code
        // which is not allowed by the github.com CSP.
        // The pref option does not work to disable CSP for some reason.
        const cfgPath = await getFirefoxCfgPath()
        const disableCspPreference = '\npref("security.csp.enable", false);\n'
        if (!(await readFile(cfgPath, 'utf-8')).includes(disableCspPreference)) {
            await appendFile(cfgPath, disableCspPreference)
        }
        if (loadExtension) {
            const cdpPort = await getFreePort()
            const firefoxExtensionPath = path.resolve(__dirname, '..', '..', '..', 'browser', 'build', 'firefox')
            // webExt.util.logger.consoleStream.makeVerbose()
            args.push(`-juggler=${cdpPort}`)
            if (launchOptions.headless) {
                args.push('-headless')
            }
            await webExt.cmd.run(
                {
                    sourceDir: firefoxExtensionPath,
                    firefox: puppeteerFirefox.executablePath(),
                    args,
                },
                { shouldExitProgram: false }
            )
            const browserWSEndpoint = `ws://127.0.0.1:${cdpPort}`
            browser = await puppeteerFirefox.connect({ browserWSEndpoint })
        } else {
            browser = await puppeteerFirefox.launch(launchOptions)
        }
    } else {
        // Chrome
        args.push('--window-size=1280,1024')
        if (process.getuid() === 0) {
            // TODO don't run as root in CI
            console.warn('Running as root, disabling sandbox')
            args.push('--no-sandbox', '--disable-setuid-sandbox')
        }
        if (loadExtension) {
            const chromeExtensionPath = path.resolve(__dirname, '..', '..', '..', 'browser', 'build', 'chrome')
            const manifest = JSON.parse(
                await readFile(path.resolve(chromeExtensionPath, 'manifest.json'), 'utf-8')
            ) as { permissions: string[] }
            if (!manifest.permissions.includes('<all_urls>')) {
                throw new Error(
                    'Browser extension was not built with permissions for all URLs.\nThis is necessary because permissions cannot be granted by e2e tests.\nTo fix, run `EXTENSION_PERMISSIONS_ALL_URLS=true yarn run dev` inside the browser/ directory.'
                )
            }
            args.push(`--disable-extensions-except=${chromeExtensionPath}`, `--load-extension=${chromeExtensionPath}`)
        }
        browser = await puppeteer.launch(launchOptions)
    }

    const page = await browser.newPage()

    return new Driver(browser, page, options)
}
