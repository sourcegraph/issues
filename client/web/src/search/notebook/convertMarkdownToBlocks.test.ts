import { convertMarkdownToBlocks } from './convertMarkdownToBlocks'

describe('convertMarkdownToBlocks', () => {
    it('should handle empty markdown', () => {
        expect(convertMarkdownToBlocks('')).toStrictEqual([])
    })

    it('should handle only markdown text', () => {
        const markdown = `# Title

## Second title

* L1
* L2

Paragraph`
        expect(convertMarkdownToBlocks(markdown)).toStrictEqual([{ type: 'md', input: markdown }])
    })

    it('should handle interleaved markdown and query blocks at the start', () => {
        const markdown = `# Title

\`\`\`sourcegraph
my query
\`\`\`

## Second title

Paragraph with list:

* 1
* 2
* 3

\`\`\`sourcegraph
my second query
\`\`\`

### Third title`

        expect(convertMarkdownToBlocks(markdown)).toStrictEqual([
            { type: 'md', input: '# Title\n\n' },
            { type: 'query', input: 'my query' },
            { type: 'md', input: '## Second title\n\nParagraph with list:\n\n* 1\n* 2\n* 3\n\n' },
            { type: 'query', input: 'my second query' },
            { type: 'md', input: '### Third title' },
        ])
    })
})
