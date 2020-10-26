import { take } from 'rxjs/operators'
import { PlatformContext } from '../../../../shared/src/platform/context'
import { buildSearchURLQuery } from '../../../../shared/src/util/url'
import { createSuggestionFetcher } from '../backend/search'
import { observeSourcegraphURL, getAssetsURL, DEFAULT_SOURCEGRAPH_URL } from '../util/context'
import { createPlatformContext } from '../platform/context'
import { from } from 'rxjs'
import { isDefined, isNot } from '../../../../shared/src/util/types'
import { ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import { Settings } from '../../../../shared/src/settings/settings'
import { SearchPatternType } from '../../graphql-operations'

const isURL = /^https?:\/\//
const IS_EXTENSION = true // This feature is only supported in browser extension

export class SearchCommand {
    public description = 'Enter a search query'

    private suggestionFetcher = createSuggestionFetcher(20, this.requestGraphQL)

    private prev: { query: string; suggestions: browser.omnibox.SuggestResult[] } = { query: '', suggestions: [] }

    constructor(private requestGraphQL: PlatformContext['requestGraphQL']) {}

    public getSuggestions = (query: string): Promise<browser.omnibox.SuggestResult[]> =>
        new Promise(resolve => {
            if (this.prev.query === query) {
                resolve(this.prev.suggestions)
                return
            }

            this.suggestionFetcher({
                query,
                handler: async suggestions => {
                    const sourcegraphURL = await observeSourcegraphURL(IS_EXTENSION).pipe(take(1)).toPromise()
                    const built = suggestions.map(({ title, url, urlLabel }) => ({
                        content: `${sourcegraphURL}${url}`,
                        description: `${title} - ${urlLabel}`,
                    }))

                    this.prev = {
                        query,
                        suggestions: built,
                    }

                    resolve(built)
                },
            })
        })

    public action = async (query: string, disposition?: string): Promise<void> => {
        const sourcegraphURL = await observeSourcegraphURL(IS_EXTENSION).pipe(take(1)).toPromise()

        const patternType = await this.getDefaultSearchPatternType(sourcegraphURL)

        const props = {
            url: isURL.test(query)
                ? query
                : `${sourcegraphURL}/search?${buildSearchURLQuery(query, patternType, false)}&utm_source=omnibox`,
        }

        switch (disposition) {
            case 'newForegroundTab':
                await browser.tabs.create(props)
                break
            case 'newBackgroundTab':
                await browser.tabs.create({ ...props, active: false })
                break
            case 'currentTab':
            default:
                await browser.tabs.update(props)
                break
        }
    }

    private lastSourcegraphUrl = ''
    private settingsTimeoutHandler = 0
    private defaultPatternType = SearchPatternType.literal
    private readonly settingsTimeoutDuration = 60 * 60 * 1000 // one hour

    private async getDefaultSearchPatternType(sourcegraphURL: string): Promise<SearchPatternType> {
        try {
            // Refresh settings when either:
            // - First search
            // - Sourcegraph URL changes
            // - Over an hour has passed since last refresh

            if (this.lastSourcegraphUrl !== sourcegraphURL || this.settingsTimeoutHandler === 0) {
                clearTimeout(this.settingsTimeoutHandler)
                this.settingsTimeoutHandler = 0

                const platformContext = createPlatformContext(
                    { urlToFile: undefined, getContext: undefined },
                    { sourcegraphURL, assetsURL: getAssetsURL(DEFAULT_SOURCEGRAPH_URL) },
                    IS_EXTENSION
                )

                await platformContext.refreshSettings()
                const settings = (await from(platformContext.settings).pipe(take(1)).toPromise()).final

                if (isDefined(settings) && isNot<ErrorLike | Settings, ErrorLike>(isErrorLike)(settings)) {
                    this.defaultPatternType =
                        (settings['search.defaultPatternType'] as SearchPatternType) || this.defaultPatternType
                }

                this.lastSourcegraphUrl = sourcegraphURL
                this.settingsTimeoutHandler = window.setTimeout(() => {
                    this.settingsTimeoutHandler = 0
                }, this.settingsTimeoutDuration)
            }
        } catch {
            // Ignore errors trying to get settings, fall to return default below
        }

        return this.defaultPatternType
    }
}
