import { MarkupKind } from '@sourcegraph/extension-api-classes'
import * as H from 'history'
import React from 'react'
import renderer from 'react-test-renderer'
import { createRenderer } from 'react-test-renderer/shallow'
import { NOOP_TELEMETRY_SERVICE } from '../telemetry/telemetryService'
import { HoverOverlay, HoverOverlayProps } from './HoverOverlay'
import { NEVER } from 'rxjs'
import { subtypeOf } from '../util/types'

const renderShallow = (element: React.ReactElement<HoverOverlayProps>): React.ReactElement => {
    const renderer = createRenderer()
    renderer.render(element)
    return renderer.getRenderOutput()
}

describe('HoverOverlay', () => {
    const NOOP_EXTENSIONS_CONTROLLER = { executeCommand: () => Promise.resolve() }
    const NOOP_PLATFORM_CONTEXT = { forceUpdateTooltip: () => undefined, settings: NEVER }
    const history = H.createMemoryHistory({ keyLength: 0 })
    const commonProps = subtypeOf<HoverOverlayProps>()({
        location: history.location,
        telemetryService: NOOP_TELEMETRY_SERVICE,
        extensionsController: NOOP_EXTENSIONS_CONTROLLER,
        platformContext: NOOP_PLATFORM_CONTEXT,
        showCloseButton: false,
        hoveredToken: { repoName: 'r', commitID: 'c', revision: 'v', filePath: 'f', line: 1, character: 2 },
        overlayPosition: { left: 0, top: 0 },
        isLightTheme: false,
    })

    test('actions and hover undefined', () => {
        expect(renderer.create(<HoverOverlay {...commonProps} />).toJSON()).toMatchSnapshot()
    })

    test('actions loading', () => {
        expect(renderer.create(<HoverOverlay {...commonProps} actionsOrError="loading" />).toJSON()).toMatchSnapshot()
    })

    test('hover loading', () => {
        expect(renderer.create(<HoverOverlay {...commonProps} hoverOrError="loading" />).toJSON()).toMatchSnapshot()
    })

    test('actions and hover loading', () => {
        expect(
            renderer.create(<HoverOverlay {...commonProps} actionsOrError="loading" hoverOrError="loading" />).toJSON()
        ).toMatchSnapshot()
    })

    test('actions empty', () => {
        const component = renderer.create(<HoverOverlay {...commonProps} actionsOrError={[]} />)
        expect(component.toJSON()).toMatchSnapshot()
    })

    test('hover empty', () => {
        expect(renderer.create(<HoverOverlay {...commonProps} hoverOrError={null} />).toJSON()).toMatchSnapshot()
    })

    test('actions and hover empty', () => {
        expect(
            renderer.create(<HoverOverlay {...commonProps} actionsOrError={[]} hoverOrError={null} />).toJSON()
        ).toMatchSnapshot()
    })

    test('actions present', () => {
        expect(
            renderShallow(
                <HoverOverlay
                    {...commonProps}
                    actionsOrError={[{ action: { id: 'a', command: 'c', title: 'Some title' } }]}
                />
            )
        ).toMatchSnapshot()
    })

    test('hover present', () => {
        expect(
            renderer
                .create(
                    <HoverOverlay
                        {...commonProps}
                        hoverOrError={{ contents: [{ kind: MarkupKind.Markdown, value: 'v' }] }}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })

    test('multiple hovers present', () => {
        expect(
            renderer
                .create(
                    <HoverOverlay
                        {...commonProps}
                        hoverOrError={{
                            contents: [
                                { kind: MarkupKind.Markdown, value: 'v' },
                                { kind: MarkupKind.Markdown, value: 'v2' },
                            ],
                        }}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })

    test('actions and hover present', () => {
        expect(
            renderShallow(
                <HoverOverlay
                    {...commonProps}
                    actionsOrError={[{ action: { id: 'a', command: 'c' } }]}
                    hoverOrError={{ contents: [{ kind: MarkupKind.Markdown, value: 'v' }] }}
                />
            )
        ).toMatchSnapshot()
    })

    test('actions, hover and alert present', () => {
        expect(
            renderShallow(
                <HoverOverlay
                    {...commonProps}
                    actionsOrError={[{ action: { id: 'a', command: 'c' } }]}
                    hoverOrError={{
                        contents: [{ kind: MarkupKind.Markdown, value: 'v' }],
                        alerts: [
                            {
                                summary: {
                                    kind: MarkupKind.Markdown,
                                    value: 'Testing `markdown` rendering.',
                                },
                                type: 'test-alert-dismissalType',
                            },
                        ],
                    }}
                />
            )
        ).toMatchSnapshot()
    })

    test('actions present, hover loading', () => {
        expect(
            renderShallow(
                <HoverOverlay
                    {...commonProps}
                    actionsOrError={[{ action: { id: 'a', command: 'c' } }]}
                    hoverOrError="loading"
                />
            )
        ).toMatchSnapshot()
    })

    test('hover present, actions loading', () => {
        expect(
            renderShallow(
                <HoverOverlay
                    {...commonProps}
                    actionsOrError="loading"
                    hoverOrError={{ contents: [{ kind: MarkupKind.Markdown, value: 'v' }] }}
                />
            )
        ).toMatchSnapshot()
    })

    test('actions error', () => {
        expect(
            renderShallow(<HoverOverlay {...commonProps} actionsOrError={{ message: 'm', name: 'c' }} />)
        ).toMatchSnapshot()
    })

    test('hover error', () => {
        expect(
            renderShallow(<HoverOverlay {...commonProps} hoverOrError={{ message: 'm', name: 'c' }} />)
        ).toMatchSnapshot()
    })

    test('actions and hover error', () => {
        expect(
            renderShallow(
                <HoverOverlay
                    {...commonProps}
                    actionsOrError={{ message: 'm1', name: 'c1' }}
                    hoverOrError={{ message: 'm2', name: 'c2' }}
                />
            )
        ).toMatchSnapshot()
    })

    test('actions error, hover present', () => {
        expect(
            renderShallow(
                <HoverOverlay
                    {...commonProps}
                    actionsOrError={{ message: 'm', name: 'c' }}
                    hoverOrError={{ contents: [{ kind: MarkupKind.Markdown, value: 'v' }] }}
                />
            )
        ).toMatchSnapshot()
    })

    test('hover error, actions present', () => {
        expect(
            renderShallow(
                <HoverOverlay
                    {...commonProps}
                    actionsOrError={[{ action: { id: 'a', command: 'c' } }]}
                    hoverOrError={{ message: 'm', name: 'c' }}
                />
            )
        ).toMatchSnapshot()
    })
})
