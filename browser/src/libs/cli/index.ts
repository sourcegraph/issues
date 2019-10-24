import { PlatformContext } from '../../../../shared/src/platform/context'
import { SearchCommand } from './search'

export function initializeOmniboxInterface(requestGraphQL: PlatformContext['requestGraphQL']): void {
    const searchCommand = new SearchCommand(requestGraphQL)
    browser.omnibox.onInputChanged.addListener(async (query, suggest) => {
        try {
            const suggestions = await searchCommand.getSuggestions(query)
            suggest(suggestions)
        } catch (err) {
            console.error('error getting suggestions', err)
        }
    })

    browser.omnibox.onInputEntered.addListener(async (query, disposition) => {
        await searchCommand.action(query, disposition)
    })
}
