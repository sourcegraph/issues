import { renderMarkdown } from './markdown'
import { registerHighlightContributions } from '../highlight/contributions'

registerHighlightContributions()

describe('renderMarkdown', () => {
    it('renders code blocks, with syntax highlighting', () => {
        const markdown = [
            '# This is a heading',
            '',
            '## This is a subheading',
            '',
            'Some text',
            'in the same paragraph',
            'with a [link](./destination).',
            '',
            '```ts',
            'const someTypeScriptCode = funcCall()',
            '```',
            '',
            '- bullet list item 1',
            '- bullet list item 2',
            '',
            '1. item 1',
            '  ```ts',
            '  const codeInsideTheBulletPoint = "string"',
            '  ```',
            '1. item 2',
            '',
            '> quoted',
            '> text',
            '',
            '| col 1 | col 2 |',
            '|-------|-------|',
            '| A     | B     |',
            '',
            '![image alt text](./src.jpg)',
        ].join('\n')
        expect(renderMarkdown(markdown)).toMatchInlineSnapshot(`
            "<h1 id=\\"this-is-a-heading\\">This is a heading</h1>
            <h2 id=\\"this-is-a-subheading\\">This is a subheading</h2>
            <p>Some text
            in the same paragraph
            with a <a href=\\"./destination\\">link</a>.</p>
            <pre><code class=\\"language-ts\\"><span class=\\"hljs-keyword\\">const</span> someTypeScriptCode = funcCall()</code></pre>
            <ul>
            <li>bullet list item 1</li>
            <li>bullet list item 2</li>
            </ul>
            <ol>
            <li>item 1<pre><code class=\\"language-ts\\"><span class=\\"hljs-keyword\\">const</span> codeInsideTheBulletPoint = <span class=\\"hljs-string\\">\\"string\\"</span></code></pre>
            </li>
            <li>item 2</li>
            </ol>
            <blockquote>
            <p>quoted
            text</p>
            </blockquote>
            <table>
            <thead>
            <tr>
            <th>col 1</th>
            <th>col 2</th>
            </tr>
            </thead>
            <tbody><tr>
            <td>A</td>
            <td>B</td>
            </tr>
            </tbody></table>
            <p><img src=\\"./src.jpg\\" alt=\\"image alt text\\" /></p>
            "
        `)
    })
    it('renders to plain text with plainText: true', () => {
        expect(renderMarkdown('A **b**', { plainText: true })).toBe('A b\n')
    })
    it('sanitizes script tags', () => {
        expect(renderMarkdown('<script>evil();</script>')).toBe('')
    })
    it('sanitizes event handlers', () => {
        expect(renderMarkdown('<svg><rect onclick="evil()"></rect></svg>')).toBe('<p><svg><rect></rect></svg></p>\n')
    })
    it('sanitizes non-SVG <object> tags', () => {
        expect(renderMarkdown('<object data="something"></object>')).toBe('<p></p>\n')
    })
    it('allows SVG <object> tags', () => {
        expect(renderMarkdown('<object data="something" type="image/svg+xml"></object>')).toBe(
            '<p><object data="something" type="image/svg+xml"></object></p>\n'
        )
    })
    it('allows <svg> tags', () => {
        const input =
            '<svg viewbox="10 10 10 10" width="100"><rect x="37.5" y="7.5" width="675.0" height="16.875" fill="#e05d44" stroke="white" stroke-width="1"><title>/</title></rect></svg>'
        expect(renderMarkdown(input)).toBe(`<p>${input}</p>\n`)
    })

    describe('allowDataUriLinksAndDownloads option', () => {
        const MARKDOWN_WITH_DOWNLOAD = '<a href="data:text/plain,foobar" download>D</a>\n[D2](data:text/plain,foobar)'
        test('default disabled', () => {
            expect(renderMarkdown(MARKDOWN_WITH_DOWNLOAD)).toBe('<p><a>D</a>\n<a>D2</a></p>\n')
        })
        test('enabled', () => {
            expect(renderMarkdown(MARKDOWN_WITH_DOWNLOAD, { allowDataUriLinksAndDownloads: true })).toBe(
                '<p><a href="data:text/plain,foobar" download>D</a>\n<a href="data:text/plain,foobar">D2</a></p>\n'
            )
        })
    })
})
