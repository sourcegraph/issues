import * as H from 'history'
import React from 'react'
import renderer from 'react-test-renderer'
import { createBarrier } from '../api/integration-test/testHelpers'
import { NOOP_TELEMETRY_SERVICE } from '../telemetry/telemetryService'
import { ActionItem } from './ActionItem'
import { NEVER } from 'rxjs'

jest.mock('mdi-react/OpenInNewIcon', () => 'OpenInNewIcon')

describe('ActionItem', () => {
    const NOOP_EXTENSIONS_CONTROLLER = { executeCommand: () => Promise.resolve(undefined) }
    const NOOP_PLATFORM_CONTEXT = { forceUpdateTooltip: () => undefined, settings: NEVER }
    const history = H.createMemoryHistory()

    test('non-actionItem variant', () => {
        const component = renderer.create(
            <ActionItem
                action={{ id: 'c', command: 'c', title: 't', description: 'd', iconURL: 'u', category: 'g' }}
                telemetryService={NOOP_TELEMETRY_SERVICE}
                location={history.location}
                extensionsController={NOOP_EXTENSIONS_CONTROLLER}
                platformContext={NOOP_PLATFORM_CONTEXT}
            />
        )
        expect(component.toJSON()).toMatchSnapshot()
    })

    test('actionItem variant', () => {
        const component = renderer.create(
            <ActionItem
                action={{ id: 'c', command: 'c', title: 't', description: 'd', iconURL: 'u', category: 'g' }}
                telemetryService={NOOP_TELEMETRY_SERVICE}
                variant="actionItem"
                location={history.location}
                extensionsController={NOOP_EXTENSIONS_CONTROLLER}
                platformContext={NOOP_PLATFORM_CONTEXT}
            />
        )
        expect(component.toJSON()).toMatchSnapshot()
    })

    test('noop command', () => {
        const component = renderer.create(
            <ActionItem
                action={{ id: 'c', title: 't', description: 'd', iconURL: 'u', category: 'g' }}
                telemetryService={NOOP_TELEMETRY_SERVICE}
                location={history.location}
                extensionsController={NOOP_EXTENSIONS_CONTROLLER}
                platformContext={NOOP_PLATFORM_CONTEXT}
            />
        )
        expect(component.toJSON()).toMatchSnapshot()
    })

    test('pressed toggle actionItem', () => {
        const component = renderer.create(
            <ActionItem
                action={{ id: 'a', command: 'c', actionItem: { pressed: true, label: 'b' } }}
                telemetryService={NOOP_TELEMETRY_SERVICE}
                variant="actionItem"
                location={history.location}
                extensionsController={NOOP_EXTENSIONS_CONTROLLER}
                platformContext={NOOP_PLATFORM_CONTEXT}
            />
        )
        expect(component.toJSON()).toMatchSnapshot()
    })

    test('non-pressed actionItem', () => {
        const component = renderer.create(
            <ActionItem
                action={{ id: 'a', command: 'c', actionItem: { pressed: false, label: 'b' } }}
                telemetryService={NOOP_TELEMETRY_SERVICE}
                variant="actionItem"
                location={history.location}
                extensionsController={NOOP_EXTENSIONS_CONTROLLER}
                platformContext={NOOP_PLATFORM_CONTEXT}
            />
        )
        expect(component.toJSON()).toMatchSnapshot()
    })

    test('title element', () => {
        const component = renderer.create(
            <ActionItem
                action={{ id: 'c', command: 'c', title: 't', description: 'd', iconURL: 'u', category: 'g' }}
                telemetryService={NOOP_TELEMETRY_SERVICE}
                variant="actionItem"
                title={<span>t2</span>}
                location={history.location}
                extensionsController={NOOP_EXTENSIONS_CONTROLLER}
                platformContext={NOOP_PLATFORM_CONTEXT}
            />
        )
        expect(component.toJSON()).toMatchSnapshot()
    })

    test('run command', async () => {
        const { wait, done } = createBarrier()

        const component = renderer.create(
            <ActionItem
                action={{ id: 'c', command: 'c', title: 't', description: 'd', iconURL: 'u', category: 'g' }}
                telemetryService={NOOP_TELEMETRY_SERVICE}
                variant="actionItem"
                disabledDuringExecution={true}
                location={history.location}
                extensionsController={{ ...NOOP_EXTENSIONS_CONTROLLER, executeCommand: () => wait }}
                platformContext={NOOP_PLATFORM_CONTEXT}
            />
        )

        // Run command and wait for execution to finish.
        let tree = component.toJSON()
        tree!.props.onClick({ preventDefault: () => undefined, currentTarget: { blur: () => undefined } })
        tree = component.toJSON()
        expect(tree).toMatchSnapshot()

        // Finish execution. (Use setTimeout to wait for the executeCommand resolution to result in the setState
        // call.)
        done()
        await new Promise<void>(resolve => setTimeout(resolve))
        tree = component.toJSON()
        expect(tree).toMatchSnapshot()
    })

    test('run command with showLoadingSpinnerDuringExecution', async () => {
        const { wait, done } = createBarrier()

        const component = renderer.create(
            <ActionItem
                action={{ id: 'c', command: 'c', title: 't', description: 'd', iconURL: 'u', category: 'g' }}
                telemetryService={NOOP_TELEMETRY_SERVICE}
                variant="actionItem"
                showLoadingSpinnerDuringExecution={true}
                location={history.location}
                extensionsController={{ ...NOOP_EXTENSIONS_CONTROLLER, executeCommand: () => wait }}
                platformContext={NOOP_PLATFORM_CONTEXT}
            />
        )

        // Run command and wait for execution to finish.
        let tree = component.toJSON()
        tree!.props.onClick({ preventDefault: () => undefined, currentTarget: { blur: () => undefined } })
        tree = component.toJSON()
        expect(tree).toMatchSnapshot()

        // Finish execution. (Use setTimeout to wait for the executeCommand resolution to result in the setState
        // call.)
        done()
        await new Promise<void>(resolve => setTimeout(resolve))
        tree = component.toJSON()
        expect(tree).toMatchSnapshot()
    })

    test('run command with error', async () => {
        const component = renderer.create(
            <ActionItem
                action={{ id: 'c', command: 'c', title: 't', description: 'd', iconURL: 'u', category: 'g' }}
                telemetryService={NOOP_TELEMETRY_SERVICE}
                variant="actionItem"
                disabledDuringExecution={true}
                location={history.location}
                extensionsController={{
                    ...NOOP_EXTENSIONS_CONTROLLER,
                    executeCommand: () => Promise.reject(new Error('x')),
                }}
                platformContext={NOOP_PLATFORM_CONTEXT}
            />
        )

        // Run command (which will reject with an error). (Use setTimeout to wait for the executeCommand resolution
        // to result in the setState call.)
        let tree = component.toJSON()
        tree!.props.onClick({ preventDefault: () => undefined, currentTarget: { blur: () => undefined } })
        await new Promise<void>(resolve => setTimeout(resolve))
        tree = component.toJSON()
        expect(tree).toMatchSnapshot()
    })

    test('run command with error with showInlineError', async () => {
        const component = renderer.create(
            <ActionItem
                action={{ id: 'c', command: 'c', title: 't', description: 'd', iconURL: 'u', category: 'g' }}
                telemetryService={NOOP_TELEMETRY_SERVICE}
                variant="actionItem"
                showInlineError={true}
                location={history.location}
                extensionsController={{
                    ...NOOP_EXTENSIONS_CONTROLLER,
                    executeCommand: () => Promise.reject(new Error('x')),
                }}
                platformContext={NOOP_PLATFORM_CONTEXT}
            />
        )

        // Run command (which will reject with an error). (Use setTimeout to wait for the executeCommand resolution
        // to result in the setState call.)
        let tree = component.toJSON()
        tree!.props.onClick({ preventDefault: () => undefined, currentTarget: { blur: () => undefined } })
        await new Promise<void>(resolve => setTimeout(resolve))
        tree = component.toJSON()
        expect(tree).toMatchSnapshot()
    })

    describe('"open" command', () => {
        it('renders as link', () => {
            jsdom.reconfigure({ url: 'https://example.com/foo' })

            const component = renderer.create(
                <ActionItem
                    action={{ id: 'c', command: 'open', commandArguments: ['https://example.com/bar'], title: 't' }}
                    telemetryService={NOOP_TELEMETRY_SERVICE}
                    location={history.location}
                    extensionsController={NOOP_EXTENSIONS_CONTROLLER}
                    platformContext={NOOP_PLATFORM_CONTEXT}
                />
            )
            expect(component.toJSON()).toMatchSnapshot()
        })

        it('renders as link with icon and opens a new tab for a different origin', () => {
            jsdom.reconfigure({ url: 'https://example.com/foo' })

            const component = renderer.create(
                <ActionItem
                    action={{ id: 'c', command: 'open', commandArguments: ['https://other.com/foo'], title: 't' }}
                    telemetryService={NOOP_TELEMETRY_SERVICE}
                    location={history.location}
                    extensionsController={NOOP_EXTENSIONS_CONTROLLER}
                    platformContext={NOOP_PLATFORM_CONTEXT}
                />
            )
            expect(component.toJSON()).toMatchSnapshot()
        })

        it('renders as link that opens in a new tab, but without icon for a different origin as the alt action and a primary action defined', () => {
            jsdom.reconfigure({ url: 'https://example.com/foo' })

            const component = renderer.create(
                <ActionItem
                    action={{ id: 'c1', command: 'whatever', title: 'primary' }}
                    altAction={{ id: 'c2', command: 'open', commandArguments: ['https://other.com/foo'], title: 'alt' }}
                    telemetryService={NOOP_TELEMETRY_SERVICE}
                    location={history.location}
                    extensionsController={NOOP_EXTENSIONS_CONTROLLER}
                    platformContext={NOOP_PLATFORM_CONTEXT}
                />
            )
            expect(component.toJSON()).toMatchSnapshot()
        })
    })
})
