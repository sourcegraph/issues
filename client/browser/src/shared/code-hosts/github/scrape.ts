import { commitIDFromPermalink } from '../../util/dom'

/**
 * Get the commit ID from the permalink element on the page.
 */
export function getCommitIDFromPermalink(): string {
    return commitIDFromPermalink({
        selector: '.js-permalink-shortcut',
        hrefRegex: /^\/.*?\/.*?\/(?:blob|tree)\/([\da-f]{40})/,
    })
}
