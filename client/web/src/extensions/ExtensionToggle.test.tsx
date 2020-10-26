import React from 'react'
import renderer from 'react-test-renderer'
import { ConfiguredRegistryExtension } from '../../../shared/src/extensions/extension'
import { PlatformContext } from '../../../shared/src/platform/context'
import { ConfiguredSubjectOrError, SettingsSubject } from '../../../shared/src/settings/settings'
import { ExtensionToggle } from './ExtensionToggle'

describe('ExtensionToggle', () => {
    const NOOP_PLATFORM_CONTEXT: PlatformContext = {} as any
    const SUBJECT: ConfiguredSubjectOrError = {
        lastID: null,
        settings: {},
        // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
        subject: { __typename: 'User', id: 'u', viewerCanAdminister: true } as SettingsSubject,
    }
    const EXTENSION: Pick<ConfiguredRegistryExtension, 'id'> = {
        id: 'x/y',
    }

    test('extension not present in settings', () => {
        expect(
            renderer
                .create(
                    <ExtensionToggle
                        extensionID={EXTENSION.id}
                        enabled={false}
                        settingsCascade={{ final: { extensions: {} }, subjects: [SUBJECT] }}
                        platformContext={NOOP_PLATFORM_CONTEXT}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })

    test('extension enabled in settings', () => {
        expect(
            renderer
                .create(
                    <ExtensionToggle
                        extensionID={EXTENSION.id}
                        enabled={true}
                        settingsCascade={{ final: { extensions: { 'x/y': true } }, subjects: [SUBJECT] }}
                        platformContext={NOOP_PLATFORM_CONTEXT}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })

    test('extension disabled in settings', () => {
        expect(
            renderer
                .create(
                    <ExtensionToggle
                        extensionID={EXTENSION.id}
                        enabled={false}
                        settingsCascade={{ final: { extensions: { 'x/y': false } }, subjects: [SUBJECT] }}
                        platformContext={NOOP_PLATFORM_CONTEXT}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })
})
