import React from 'react'
import renderer from 'react-test-renderer'
import { ExtensionPermissionModal } from './ExtensionPermissionModal'

describe('ExtensionPermissionModal', () => {
    test('renders', () => {
        expect(
            renderer
                .create(
                    <ExtensionPermissionModal
                        extensionID="sourcegraph/typescript"
                        givePermission={() => {}}
                        denyPermission={() => {}}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })
})
