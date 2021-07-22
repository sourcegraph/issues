import { markdownLexer } from '@sourcegraph/shared/src/util/markdown'

import { BlockInitializer } from '.'

export function convertMarkdownToBlocks(markdown: string): BlockInitializer[] {
    const blocks: BlockInitializer[] = []

    let markdownRawTokens: string[] = []
    const addMarkdownBlock = (): void => {
        if (markdownRawTokens.length === 0) {
            return
        }
        blocks.push({ type: 'md', input: markdownRawTokens.join('') })
        markdownRawTokens = []
    }

    for (const token of markdownLexer(markdown)) {
        if (token.type === 'code' && token.lang === 'sourcegraph') {
            addMarkdownBlock()
            blocks.push({ type: 'query', input: token.text })
        } else {
            markdownRawTokens.push(token.raw)
        }
    }
    addMarkdownBlock()

    return blocks
}
