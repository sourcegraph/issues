import { highlight, highlightAuto } from 'highlight.js/lib/core'
import { without } from 'lodash'
// This is the only file allowed to import this module, all other modules must use renderMarkdown() exported from here
// eslint-disable-next-line no-restricted-imports
import marked from 'marked'
import sanitize from 'sanitize-html'
import { Overwrite } from 'utility-types'

/**
 * Escapes HTML by replacing characters like `<` with their HTML escape sequences like `&lt;`
 */
const escapeHTML = (html: string): string => {
    const span = document.createElement('span')
    span.textContent = html
    return span.innerHTML
}

/**
 * Attempts to syntax-highlight the given code.
 * If the language is not given, it is auto-detected.
 * If an error occurs, the code is returned as plain text with escaped HTML entities
 *
 * @param code The code to highlight
 * @param language The language of the code, if known
 * @returns Safe HTML
 */
export const highlightCodeSafe = (code: string, language?: string): string => {
    try {
        if (language === 'plaintext' || language === 'text') {
            return escapeHTML(code)
        }
        if (language === 'sourcegraph') {
            return code
        }
        if (language) {
            return highlight(language, code, true).value
        }
        return highlightAuto(code).value
    } catch (error) {
        console.warn('Error syntax-highlighting hover markdown code block', error)
        return escapeHTML(code)
    }
}

const svgPresentationAttributes = ['fill', 'stroke', 'stroke-width'] as const
const ALL_VALUES_ALLOWED = [/.*/]

/**
 * Renders the given markdown to HTML, highlighting code and sanitizing dangerous HTML.
 * Can throw an exception on parse errors.
 *
 * @param markdown The markdown to render
 * @param options Options object for passing additional parameters
 */
export const renderMarkdown = (
    markdown: string,
    options: {
        /** Whether to render line breaks as HTML `<br>`s */
        breaks?: boolean
    } & (
        | {
              /** Strip off any HTML and return a plain text string, useful for previews */
              plainText: true
          }
        | {
              /** Following options apply to non-plaintext output only. */
              plainText?: false

              /** Allow links to data: URIs and that download files. */
              allowDataUriLinksAndDownloads?: boolean
          }
    ) = {}
): string => {
    const rendered = marked(markdown, {
        gfm: true,
        breaks: options.breaks,
        sanitize: false,
        highlight: (code, language) => highlightCodeSafe(code, language),
    })

    let sanitizeOptions: Overwrite<sanitize.IOptions, sanitize.IDefaults>
    if (options.plainText) {
        sanitizeOptions = {
            ...sanitize.defaults,
            allowedAttributes: {},
            allowedSchemes: [],
            allowedSchemesByTag: {},
            allowedTags: [],
            selfClosing: [],
        }
    } else {
        sanitizeOptions = {
            ...sanitize.defaults,
            // Ensure <object> must have type attribute set
            exclusiveFilter: ({ tag, attribs }) => tag === 'object' && !attribs.type,

            allowedTags: [
                ...without(sanitize.defaults.allowedTags, 'iframe'),
                'img',
                'picture',
                'source',
                'object',
                'svg',
                'rect',
                'defs',
                'pattern',
                'mask',
                'circle',
                'path',
                'title',
            ],
            allowedAttributes: {
                ...sanitize.defaults.allowedAttributes,
                a: [
                    ...sanitize.defaults.allowedAttributes.a,
                    'title',
                    'data-tooltip', // TODO support fancy tooltips through native titles
                    'class',
                    { name: 'rel', values: ['noopener', 'noreferrer'] },
                ],
                img: [...sanitize.defaults.allowedAttributes.img, 'alt', 'width', 'height', 'align', 'style'],
                // Support different images depending on media queries (e.g. color theme, reduced motion)
                source: ['srcset', 'media'],
                // Support SVGs for code insights.
                object: ['data', { name: 'type', values: ['image/svg+xml'] }, 'width'],
                svg: ['width', 'height', 'viewbox', 'version', 'preserveaspectratio', 'style'],
                rect: ['x', 'y', 'width', 'height', 'transform', ...svgPresentationAttributes],
                path: ['d', ...svgPresentationAttributes],
                circle: ['cx', 'cy', ...svgPresentationAttributes],
                pattern: ['id', 'width', 'height', 'patternunits', 'patterntransform'],
                mask: ['id'],
                // Allow highligh.js styles, e.g.
                // <span class="hljs-keyword">
                // <code class="language-javascript">
                span: ['class'],
                code: ['class'],
                // Support deep-linking
                h1: ['id'],
                h2: ['id'],
                h3: ['id'],
                h4: ['id'],
                h5: ['id'],
                h6: ['id'],
            },
            allowedStyles: {
                img: {
                    padding: ALL_VALUES_ALLOWED,
                    'padding-left': ALL_VALUES_ALLOWED,
                    'padding-right': ALL_VALUES_ALLOWED,
                    'padding-top': ALL_VALUES_ALLOWED,
                    'padding-bottom': ALL_VALUES_ALLOWED,
                },
                // SVGs are usually for charts in code insights.
                // Allow them to be responsive.
                svg: {
                    flex: ALL_VALUES_ALLOWED,
                    'flex-grow': ALL_VALUES_ALLOWED,
                    'flex-shrink': ALL_VALUES_ALLOWED,
                    'flex-basis': ALL_VALUES_ALLOWED,
                },
            },
        }
        if (options.allowDataUriLinksAndDownloads) {
            sanitizeOptions.allowedAttributes.a = [...sanitizeOptions.allowedAttributes.a, 'download']
            sanitizeOptions.allowedSchemesByTag = {
                ...sanitizeOptions.allowedSchemesByTag,
                a: [...(sanitizeOptions.allowedSchemesByTag.a || sanitizeOptions.allowedSchemes), 'data'],
            }
        }
    }

    return sanitize(rendered, sanitizeOptions)
}

export const markdownLexer = (markdown: string): marked.TokensList => marked.lexer(markdown)
