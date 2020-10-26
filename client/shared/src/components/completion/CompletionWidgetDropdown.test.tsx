import React from 'react'
import renderer from 'react-test-renderer'
import { CompletionItem } from 'sourcegraph'
import { CompletionWidgetDropdown } from './CompletionWidgetDropdown'

const COMPLETION_ITEM_2: CompletionItem = { label: 'b' }

describe('CompletionWidgetDropdown', () => {
    test('simple', () =>
        expect(
            renderer
                .create(
                    <CompletionWidgetDropdown
                        completionListOrError={{ items: [{ label: 'a' }, COMPLETION_ITEM_2] }}
                        highlightedIndex={1}
                        onClickOutside={() => undefined}
                        onDownshiftStateChange={() => undefined}
                        onItemSelected={() => undefined}
                        selectedItem={COMPLETION_ITEM_2}
                        listClassName="list-class-name"
                        listItemClassName="list-item-class-name"
                        loadingClassName="loading-class-name"
                        noResultsClassName="no-results-class-name"
                        selectedListItemClassName="selected-list-item-class-name"
                    />
                )
                .toJSON()
        ).toMatchSnapshot())
})
