import React, { useCallback, useState } from 'react'
import { ButtonDropdown, DropdownItem, DropdownMenu, DropdownToggle } from 'reactstrap'
import { EXTENSION_CATEGORIES, ExtensionCategory } from '../../../shared/src/schema/extensionSchema'
import { ExtensionsEnablement } from './ExtensionRegistry'

interface Props {
    selectedCategories: ExtensionCategory[]

    onSelectCategories: (
        categoriesOrCallback: ExtensionCategory[] | ((categories: ExtensionCategory[]) => ExtensionCategory[])
    ) => void

    enablementFilter: ExtensionsEnablement

    setEnablementFilter: React.Dispatch<React.SetStateAction<ExtensionsEnablement>>
}

const enablementFilterToLabel: Record<ExtensionsEnablement, string> = {
    all: 'Show all',
    enabled: 'Show enabled extensions',
    disabled: 'Show disabled extensions',
}

/**
 * Displays buttons to be rendered alongside the extension registry list query input field.
 */
export const ExtensionsQueryInputToolbar: React.FunctionComponent<Props> = ({
    selectedCategories,
    onSelectCategories,
    enablementFilter,
    setEnablementFilter,
}) => {
    const [isOpen, setIsOpen] = useState(false)
    const toggleIsOpen = useCallback(() => setIsOpen(open => !open), [])

    return (
        <div className="d-flex flex-wrap justify-content-between mb-2">
            <div>
                {EXTENSION_CATEGORIES.map(category => {
                    const selected = selectedCategories.includes(category)
                    return (
                        <button
                            type="button"
                            className={`btn btn-sm mr-2 ${selected ? 'btn-secondary' : 'btn-outline-secondary'}`}
                            data-test-extension-category={category}
                            key={category}
                            onClick={() =>
                                onSelectCategories(selectedCategories =>
                                    selected
                                        ? selectedCategories.filter(selectedCategory => selectedCategory !== category)
                                        : [...selectedCategories, category]
                                )
                            }
                        >
                            {category}
                        </button>
                    )
                })}
            </div>

            <ButtonDropdown isOpen={isOpen} toggle={toggleIsOpen}>
                <DropdownToggle className="btn-sm" caret={true} color="outline-secondary">
                    {enablementFilterToLabel[enablementFilter]}
                </DropdownToggle>
                <DropdownMenu right={true}>
                    <DropdownItem
                        // eslint-disable-next-line react/jsx-no-bind
                        onClick={() => setEnablementFilter('all')}
                        disabled={enablementFilter === 'all'}
                    >
                        Show all
                    </DropdownItem>
                    <DropdownItem
                        // eslint-disable-next-line react/jsx-no-bind
                        onClick={() => setEnablementFilter('enabled')}
                        disabled={enablementFilter === 'enabled'}
                    >
                        Show enabled extensions
                    </DropdownItem>
                    <DropdownItem
                        // eslint-disable-next-line react/jsx-no-bind
                        onClick={() => setEnablementFilter('disabled')}
                        disabled={enablementFilter === 'disabled'}
                    >
                        Show disabled extensions
                    </DropdownItem>
                </DropdownMenu>
            </ButtonDropdown>
        </div>
    )
}
