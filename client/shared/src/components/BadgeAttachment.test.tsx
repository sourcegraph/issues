import * as React from 'react'
import { cleanup, render } from '@testing-library/react'
import { BadgeAttachment } from './BadgeAttachment'
import { BadgeAttachmentRenderOptions } from 'sourcegraph'

const base64icon =
    'data:image/svg+xml;base64,IDxzdmcgeG1sbnM9J2h0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnJyBzdHlsZT0id2lkdGg6MjRweDtoZWlnaHQ6MjRweCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSIjZmZmZmZmIj4gPHBhdGggZD0iIE0xMSwgOUgxM1Y3SDExTTEyLCAyMEM3LjU5LCAyMCA0LCAxNi40MSA0LCAxMkM0LCA3LjU5IDcuNTksIDQgMTIsIDRDMTYuNDEsIDQgMjAsIDcuNTkgMjAsIDEyQzIwLCAxNi40MSAxNi40MSwgMjAgMTIsIDIwTTEyLCAyQTEwLCAxMCAwIDAsIDAgMiwgMTJBMTAsIDEwIDAgMCwgMCAxMiwgMjJBMTAsIDEwIDAgMCwgMCAyMiwgMTJBMTAsIDEwIDAgMCwgMCAxMiwgMk0xMSwgMTdIMTNWMTFIMTFWMTdaIiAvPiA8L3N2Zz4g'

export const base64ImageBadge: Omit<BadgeAttachmentRenderOptions, 'kind'> = {
    icon: base64icon,
    light: { icon: base64icon },
    hoverMessage:
        'Search-based results - click to see how these results are calculated and how to get precise intelligence with LSIF.',
    linkURL: 'https://docs.sourcegraph.com/code_intelligence/explanations/basic_code_intelligence',
}

export const badgeWithKind: BadgeAttachmentRenderOptions = {
    ...base64ImageBadge,
    kind: 'info',
}

describe('BadgeAttachment', () => {
    afterAll(cleanup)

    it('renders an img element with a base64 icon', () => {
        const { container } = render(
            <BadgeAttachment attachment={base64ImageBadge as BadgeAttachmentRenderOptions} isLightTheme={true} />
        )
        expect(container).toMatchSnapshot()
    })

    it('renders an svg element with a predefined icon', () => {
        const { container } = render(<BadgeAttachment attachment={badgeWithKind} isLightTheme={true} />)
        expect(container).toMatchSnapshot()
    })
})
