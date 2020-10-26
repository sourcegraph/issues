import { ContentView } from '../shared/contentViews'
import { ViewResolver } from '../shared/views'

/**
 * Matches all GitHub Markdown body content, including comment bodies, issue/PR descriptions, review
 * comments, and rendered Markdown files.
 */
export const markdownBodyViewResolver: ViewResolver<ContentView> = {
    selector: '.markdown-body',
    resolveView: element => ({ element }),
}
