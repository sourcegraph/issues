import { storiesOf } from '@storybook/react'
import React from 'react'
import { action } from '@storybook/addon-actions'
import { boolean } from '@storybook/addon-knobs'
import { createMemoryHistory } from 'history'
import { HoverOverlay, HoverOverlayClassProps } from './HoverOverlay'
import { MarkupKind } from '@sourcegraph/extension-api-classes'
import { NOOP_TELEMETRY_SERVICE } from '../telemetry/telemetryService'
import { of } from 'rxjs'
import { registerHighlightContributions } from '../highlight/contributions'
import { PlatformContext } from '../platform/context'

import webStyles from '../../../web/src/SourcegraphWebApp.scss'
import bitbucketStyles from '@atlassian/aui/dist/aui/css/aui.css'
import browserExtensionStyles from '../../../browser/src/app.scss'
import { BadgeAttachmentRenderOptions, MarkupContent, Badged } from 'sourcegraph'

registerHighlightContributions()

const { add } = storiesOf('shared/HoverOverlay', module)

const history = createMemoryHistory()
const NOOP_EXTENSIONS_CONTROLLER = { executeCommand: () => Promise.resolve() }
const NOOP_PLATFORM_CONTEXT: Pick<PlatformContext, 'forceUpdateTooltip' | 'settings'> = {
    forceUpdateTooltip: () => undefined,
    settings: of({ final: {}, subjects: [] }),
}

const commonProps = () => ({
    showCloseButton: boolean('showCloseButton', true),
    location: history.location,
    telemetryService: NOOP_TELEMETRY_SERVICE,
    extensionsController: NOOP_EXTENSIONS_CONTROLLER,
    platformContext: NOOP_PLATFORM_CONTEXT,
    isLightTheme: true,
    overlayPosition: { top: 16, left: 16 },
    onAlertDismissed: action('onAlertDismissed'),
    onCloseButtonClick: action('onCloseButtonClick'),
})
const webHoverOverlayClassProps: HoverOverlayClassProps = {
    className: 'card',
    iconClassName: 'icon-inline',
    iconButtonClassName: 'btn btn-icon',
    actionItemClassName: 'btn btn-secondary',
    infoAlertClassName: 'alert alert-info',
    errorAlertClassName: 'alert alert-danger',
}
const bitbucketClassProps: HoverOverlayClassProps = {
    className: 'aui-dialog',
    actionItemClassName: 'aui-button hover-action-item--bitbucket-server',
    iconButtonClassName: 'aui-button btn-icon--bitbucket-server',
    infoAlertClassName: 'aui-message aui-message-info',
    errorAlertClassName: 'aui-message aui-message-error',
    iconClassName: 'aui-icon',
}

const FIXTURE_BADGE: BadgeAttachmentRenderOptions = {
    kind: 'info',
    hoverMessage:
        'Search-based results - click to see how these results are calculated and how to get precise intelligence with LSIF.',
    linkURL: 'https://docs.sourcegraph.com/code_intelligence/explanations/basic_code_intelligence',
}

const LEGACY_FIXTURE_BADGE = {
    icon:
        'data:image/svg+xml;base64,IDxzdmcgeG1sbnM9J2h0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnJyBzdHlsZT0id2lkdGg6MjRweDtoZWlnaHQ6MjRweCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSIjZmZmZmZmIj4gPHBhdGggZD0iIE0xMSwgOUgxM1Y3SDExTTEyLCAyMEM3LjU5LCAyMCA0LCAxNi40MSA0LCAxMkM0LCA3LjU5IDcuNTksIDQgMTIsIDRDMTYuNDEsIDQgMjAsIDcuNTkgMjAsIDEyQzIwLCAxNi40MSAxNi40MSwgMjAgMTIsIDIwTTEyLCAyQTEwLCAxMCAwIDAsIDAgMiwgMTJBMTAsIDEwIDAgMCwgMCAxMiwgMjJBMTAsIDEwIDAgMCwgMCAyMiwgMTJBMTAsIDEwIDAgMCwgMCAxMiwgMk0xMSwgMTdIMTNWMTFIMTFWMTdaIiAvPiA8L3N2Zz4g',
    light: {
        icon:
            'data:image/svg+xml;base64,IDxzdmcgeG1sbnM9J2h0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnJyBzdHlsZT0id2lkdGg6MjRweDtoZWlnaHQ6MjRweCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSIjMDAwMDAwIj4gPHBhdGggZD0iIE0xMSwgOUgxM1Y3SDExTTEyLCAyMEM3LjU5LCAyMCA0LCAxNi40MSA0LCAxMkM0LCA3LjU5IDcuNTksIDQgMTIsIDRDMTYuNDEsIDQgMjAsIDcuNTkgMjAsIDEyQzIwLCAxNi40MSAxNi40MSwgMjAgMTIsIDIwTTEyLCAyQTEwLCAxMCAwIDAsIDAgMiwgMTJBMTAsIDEwIDAgMCwgMCAxMiwgMjJBMTAsIDEwIDAgMCwgMCAyMiwgMTJBMTAsIDEwIDAgMCwgMCAxMiwgMk0xMSwgMTdIMTNWMTFIMTFWMTdaIiAvPiA8L3N2Zz4g',
    },
    hoverMessage:
        'Search-based results - click to see how these results are calculated and how to get precise intelligence with LSIF.',
    linkURL: 'https://docs.sourcegraph.com/code_intelligence/explanations/basic_code_intelligence',
} as BadgeAttachmentRenderOptions

const FIXTURE_CONTENT: Badged<MarkupContent> = {
    value:
        '```typescript\nexport interface TestInterface<A, B, C>\n```\n\n' +
        '---\n\nVeniam voluptate quis magna mollit aliqua enim id ea fugiat. Aliqua anim eiusmod nisi excepteur.\n',
    kind: MarkupKind.Markdown,
    badge: FIXTURE_BADGE,
}

const FIXTURE_CONTENT_LONG_CODE = {
    ...FIXTURE_CONTENT,
    value:
        '```typescript\nexport interface LongTestInterface<A, B, C, D, E, F, G, H, I, J, K, L, M, N, O, P, Q, R, S, T, U, V, W, X, Y, Z, A, B, C, D, E, F, G, H, I, J, K, L, M, N, O, P, Q, R, S, T, U, V, W, X, Y, Z>\n```\n\n' +
        '---\n\nNisi id deserunt culpa dolore aute pariatur ut amet veniam. Proident id Lorem reprehenderit veniam sunt velit.\n',
}

const FIXTURE_CONTENT_LONG_TEXT_ONLY = {
    ...FIXTURE_CONTENT,
    value:
        'Mollit ea esse magna incididunt aliquip mollit non reprehenderit veniam anim. Veniam in dolor elit sint aliqua non cillum. Est sit pariatur ut cupidatat magna dolore. Sint et culpa voluptate ad sit eu ea dolor. Dolore Lorem cillum esse pariatur elit dolore dolor quis fugiat labore non. Elit nostrud minim aliqua adipisicing laborum ad sunt velit amet. In voluptate est voluptate labore consectetur proident. Nostrud exercitation ut officia enim minim tempor qui adipisicing sunt et occaecat anim irure. Culpa irure reprehenderit reprehenderit dolore sint aliquip non ex excepteur ipsum dolor. Et qui anim officia magna enim laboris enim exercitation pariatur. Cillum consequat elit dolore tempor magna exercitation ad laborum consequat aute consequat.',
}

const FIXTURE_ACTIONS = [
    {
        action: {
            id: 'goToDefinition.preloaded',
            title: 'Go to definition',
            command: 'open',
            commandArguments: ['/github.com/sourcegraph/codeintellify/-/blob/src/hoverifier.ts?subtree=true#L57:1'],
        },
    },
    {
        action: {
            id: 'findReferences',
            title: 'Find references',
            command: 'open',
            commandArguments: [
                '/github.com/sourcegraph/codeintellify/-/blob/src/hoverifier.ts?subtree=true#L57:18&tab=references',
            ],
        },
    },
]

add('Loading', () => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">
            <HoverOverlay
                {...commonProps()}
                {...webHoverOverlayClassProps}
                hoverOrError="loading"
                actionsOrError={FIXTURE_ACTIONS}
            />
        </div>
    </>
))

add('Error', () => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">
            <HoverOverlay
                {...commonProps()}
                {...webHoverOverlayClassProps}
                hoverOrError={
                    new Error(
                        'Something terrible happened: Eiusmod voluptate deserunt in sint cillum pariatur laborum eiusmod.'
                    )
                }
                actionsOrError={FIXTURE_ACTIONS}
            />
        </div>
    </>
))

add('Common content', () => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">
            <HoverOverlay
                {...commonProps()}
                {...webHoverOverlayClassProps}
                hoverOrError={{
                    contents: [FIXTURE_CONTENT],
                }}
                actionsOrError={FIXTURE_ACTIONS}
            />
        </div>
    </>
))

add('Legacy badge', () => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">
            <HoverOverlay
                {...commonProps()}
                {...webHoverOverlayClassProps}
                hoverOrError={{
                    contents: [{ ...FIXTURE_CONTENT, badge: LEGACY_FIXTURE_BADGE }],
                }}
                actionsOrError={FIXTURE_ACTIONS}
            />
        </div>
    </>
))

add('Only actions', () => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">
            <HoverOverlay
                {...commonProps()}
                {...webHoverOverlayClassProps}
                hoverOrError={null}
                actionsOrError={FIXTURE_ACTIONS}
            />
        </div>
    </>
))

add('Long code', () => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">
            <HoverOverlay
                {...commonProps()}
                {...webHoverOverlayClassProps}
                hoverOrError={{
                    contents: [FIXTURE_CONTENT_LONG_CODE],
                }}
                actionsOrError={FIXTURE_ACTIONS}
            />
        </div>
    </>
))

add('Long text only', () => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">
            <HoverOverlay
                {...commonProps()}
                {...webHoverOverlayClassProps}
                hoverOrError={{
                    contents: [FIXTURE_CONTENT_LONG_TEXT_ONLY],
                }}
                actionsOrError={FIXTURE_ACTIONS}
            />
        </div>
    </>
))

add('Multiple MarkupContents', () => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">
            <HoverOverlay
                {...commonProps()}
                {...webHoverOverlayClassProps}
                hoverOrError={{
                    contents: [FIXTURE_CONTENT, FIXTURE_CONTENT, FIXTURE_CONTENT],
                }}
                actionsOrError={FIXTURE_ACTIONS}
            />
        </div>
    </>
))

add('With small-text alert', () => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">
            <HoverOverlay
                {...commonProps()}
                {...webHoverOverlayClassProps}
                hoverOrError={{
                    contents: [FIXTURE_CONTENT],
                    alerts: [
                        {
                            summary: {
                                kind: MarkupKind.Markdown,
                                value:
                                    '<small>This is a test alert. Enim esse quis commodo ex. Pariatur tempor laborum officiairure est do est laborum nostrud cillum. Cupidatat id consectetur et eiusmod Loremproident cupidatat ullamco dolor nostrud. Cupidatat sit do dolor aliqua labore adlaboris cillum deserunt dolor. Sunt labore veniam Lorem reprehenderit quis occaecatsint do mollit aliquip. Consectetur mollit mollit magna eiusmod duis ex. Sint nisilabore labore nulla laboris.</small>',
                            },
                            type: 'test-alert-type',
                        },
                    ],
                }}
                actionsOrError={FIXTURE_ACTIONS}
                onAlertDismissed={action('onAlertDismissed')}
            />
        </div>
    </>
))

add('With one-line alert', () => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">
            <HoverOverlay
                {...commonProps()}
                {...webHoverOverlayClassProps}
                hoverOrError={{
                    contents: [FIXTURE_CONTENT],
                    alerts: [
                        {
                            summary: {
                                kind: MarkupKind.PlainText,
                                value: 'This is a test alert.',
                            },
                        },
                    ],
                }}
                actionsOrError={FIXTURE_ACTIONS}
                onAlertDismissed={action('onAlertDismissed')}
            />
        </div>
    </>
))

add('With badged alert', () => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">
            <HoverOverlay
                {...commonProps()}
                {...webHoverOverlayClassProps}
                hoverOrError={{
                    contents: [FIXTURE_CONTENT],
                    alerts: [
                        {
                            summary: {
                                kind: MarkupKind.PlainText,
                                value: 'This is a test alert.',
                            },
                            badge: {
                                kind: 'info',
                            },
                        },
                    ],
                }}
                actionsOrError={FIXTURE_ACTIONS}
                onAlertDismissed={action('onAlertDismissed')}
            />
        </div>
    </>
))

add('With badged dismissible alert', () => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">
            <HoverOverlay
                {...commonProps()}
                {...webHoverOverlayClassProps}
                hoverOrError={{
                    contents: [FIXTURE_CONTENT],
                    alerts: [
                        {
                            summary: {
                                kind: MarkupKind.Markdown,
                                value: 'This is a test alert.',
                            },
                            type: 'test-alert-type',
                            badge: {
                                kind: 'info',
                            },
                        },
                    ],
                }}
                actionsOrError={FIXTURE_ACTIONS}
                onAlertDismissed={action('onAlertDismissed')}
            />
        </div>
    </>
))

add('With long markdown text badged dismissible alert.', () => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">
            <HoverOverlay
                {...commonProps()}
                {...webHoverOverlayClassProps}
                hoverOrError={{
                    contents: [FIXTURE_CONTENT],
                    alerts: [
                        {
                            summary: {
                                kind: MarkupKind.Markdown,
                                value:
                                    'This is a test alert. [It uses Markdown.](https://sourcegraph.com) `To render things easily`. *Cool!*',
                            },
                            type: 'test-alert-type',
                            badge: {
                                kind: 'info',
                            },
                        },
                    ],
                }}
                actionsOrError={FIXTURE_ACTIONS}
                onAlertDismissed={action('onAlertDismissed')}
            />
        </div>
    </>
))

add('Bitbucket styles', () => (
    <>
        <style>{bitbucketStyles}</style>
        <style>{browserExtensionStyles}</style>
        <HoverOverlay
            {...commonProps()}
            {...bitbucketClassProps}
            hoverOrError={{
                contents: [FIXTURE_CONTENT],
            }}
            actionsOrError={FIXTURE_ACTIONS}
        />
    </>
))
