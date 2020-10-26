import React from 'react'
import renderer from 'react-test-renderer'
import { ExtensionsQueryInputToolbar } from './ExtensionsQueryInputToolbar'

describe('ExtensionsQueryInputToolbar', () => {
    test('renders', () => {
        expect(
            renderer
                .create(
                    <ExtensionsQueryInputToolbar
                        selectedCategories={[]}
                        onSelectCategories={() => {}}
                        enablementFilter="all"
                        setEnablementFilter={() => {}}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })

    test('shows category in query as selected', () => {
        expect(
            renderer
                .create(
                    <ExtensionsQueryInputToolbar
                        selectedCategories={[]}
                        onSelectCategories={() => {}}
                        enablementFilter="all"
                        setEnablementFilter={() => {}}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })
})
