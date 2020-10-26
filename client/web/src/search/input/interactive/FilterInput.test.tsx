import React from 'react'
import { FilterInput } from './FilterInput'
import { FilterType, FiltersToTypeAndValue } from '../../../../../shared/src/search/interactive/util'
import sinon from 'sinon'
import { render, fireEvent, cleanup, getByText, getByDisplayValue } from '@testing-library/react'

const defaultFiltersInQuery: FiltersToTypeAndValue = {
    fork: {
        type: FilterType.fork,
        value: 'no',
        editable: false,
        negated: false,
    },
}
const defaultProps = {
    filtersInQuery: defaultFiltersInQuery,
    navbarQuery: { query: 'test', cursorPosition: 4 },
    mapKey: 'repo',
    value: '',
    filterType: FilterType.repo as Exclude<FilterType, FilterType.patterntype>,
    editable: true,
    negated: false,
    globbing: true,
    onSubmit: sinon.spy(),
    onFilterEdited: sinon.spy(),
    onFilterDeleted: sinon.spy(),
    toggleFilterEditable: sinon.spy(),
    toggleFilterNegated: sinon.spy(),
}

describe('FilterInput', () => {
    afterAll(cleanup)
    let container: HTMLElement
    let nextFiltersInQuery: FiltersToTypeAndValue
    let nextValue: string
    beforeEach(() => {
        defaultProps.onFilterEdited.resetHistory()
        nextFiltersInQuery = {}
        nextValue = ''
    })
    const filterHandler = (newFiltersInQuery: FiltersToTypeAndValue, value: string) => {
        nextFiltersInQuery = newFiltersInQuery
        nextValue = value
    }

    const onFilterEditedHandler = (filterKey: string, inputValue: string) => {
        const newFiltersInQuery = {
            ...defaultFiltersInQuery,
            [filterKey]: {
                ...defaultFiltersInQuery[filterKey],
                inputValue,
                editable: false,
            },
        }
        filterHandler(newFiltersInQuery, `${inputValue}`)
    }

    const toggleFilterEditableTrue = (filterKey: string) => {
        const newFiltersInQuery = {
            ...defaultFiltersInQuery,
            [filterKey]: {
                ...defaultFiltersInQuery[filterKey],
                editable: true,
            },
        }
        nextFiltersInQuery = newFiltersInQuery
        // This is a hack. It doesn't appear to be possible to simulate
        // the proper re-rendering behavior when clicking a filter input
        // to make it editable. Thus, we have to use JSON.parse to set
        // the correct next value when the filter is set as editable=true.
        nextValue = JSON.parse(nextValue)
    }

    describe('For type filters', () => {
        it('successfully updates when submitting with an empty value', () => {
            container = render(
                <FilterInput {...defaultProps} value="diff" filterType={FilterType.type} mapKey="type" />
            ).container
            const codeRadioButton = container.querySelector('.test-filter-input-radio-button-')
            expect(codeRadioButton).toBeTruthy()
            fireEvent.click(codeRadioButton!)
            const confirmButton = container.querySelector('.test-confirm-filter-button')
            expect(confirmButton).toBeTruthy()
            fireEvent.click(confirmButton!)
            expect(defaultProps.onFilterEdited.calledOnce).toBe(true)
        })
    })

    describe('For all other filters', () => {
        it('Updating filters with an empty value does not work', () => {
            container = render(<FilterInput {...defaultProps} />).container
            const inputElement = container.querySelector('.filter-input__input-field')
            expect(inputElement).toBeTruthy()
            const confirmButton = container.querySelector('.check-button__btn')
            expect(confirmButton).toBeTruthy()
            fireEvent.click(confirmButton!)
            expect(defaultProps.onFilterEdited.notCalled).toBe(true)
        })

        it('displays quoted value when uneditable', () => {
            container = render(
                <FilterInput
                    {...defaultProps}
                    mapKey="content"
                    filterType={FilterType.content}
                    onFilterEdited={onFilterEditedHandler}
                    editable={true}
                />
            ).container

            const inputElement = container.querySelector('.filter-input__input-field')
            expect(inputElement).toBeTruthy()
            fireEvent.click(inputElement!)
            fireEvent.change(inputElement!, { target: { value: 'test query' } })
            const confirmButton = container.querySelector('.check-button__btn')
            expect(confirmButton).toBeTruthy()
            fireEvent.click(confirmButton!)
            container = render(
                <FilterInput
                    {...defaultProps}
                    mapKey="content"
                    filtersInQuery={nextFiltersInQuery}
                    filterType={FilterType.content}
                    value={nextValue}
                    onFilterEdited={onFilterEditedHandler}
                    editable={false}
                />
            ).container
            expect(getByText(container, 'content:"test query"')).toBeTruthy()
        })

        it('displays quoted, escaped value when uneditable', () => {
            container = render(
                <FilterInput
                    {...defaultProps}
                    mapKey="repo"
                    filterType={FilterType.repo}
                    onFilterEdited={onFilterEditedHandler}
                    editable={true}
                />
            ).container

            const inputElement = container.querySelector('.filter-input__input-field')
            expect(inputElement).toBeTruthy()
            fireEvent.click(inputElement!)
            fireEvent.change(inputElement!, { target: { value: '"foo' } })
            const confirmButton = container.querySelector('.check-button__btn')
            expect(confirmButton).toBeTruthy()
            fireEvent.click(confirmButton!)
            container = render(
                <FilterInput
                    {...defaultProps}
                    mapKey="repo"
                    filtersInQuery={nextFiltersInQuery}
                    filterType={FilterType.repo}
                    value={nextValue}
                    onFilterEdited={onFilterEditedHandler}
                    editable={false}
                />
            ).container
            expect(getByText(container, 'repo:"\\"foo"')).toBeTruthy()
        })

        it('displays non-quoted value when editable', () => {
            container = render(
                <FilterInput
                    {...defaultProps}
                    mapKey="content"
                    filterType={FilterType.content}
                    onFilterEdited={onFilterEditedHandler}
                    toggleFilterEditable={toggleFilterEditableTrue}
                    editable={true}
                />
            ).container

            const inputElement = container.querySelector('.filter-input__input-field')
            expect(inputElement).toBeTruthy()
            fireEvent.click(inputElement!)
            fireEvent.change(inputElement!, { target: { value: 'test query' } })
            const confirmButton = container.querySelector('.check-button__btn')
            expect(confirmButton).toBeTruthy()
            fireEvent.click(confirmButton!)
            container = render(
                <FilterInput
                    {...defaultProps}
                    mapKey="content"
                    filtersInQuery={nextFiltersInQuery}
                    filterType={FilterType.content}
                    value={nextValue}
                    onFilterEdited={onFilterEditedHandler}
                    toggleFilterEditable={toggleFilterEditableTrue}
                    editable={false}
                />
            ).container
            const buttonText = container.querySelector('.filter-input__button-text')
            expect(buttonText).toBeTruthy()
            fireEvent.click(buttonText!)
            container = render(
                <FilterInput
                    {...defaultProps}
                    mapKey="content"
                    filtersInQuery={nextFiltersInQuery}
                    filterType={FilterType.content}
                    value={nextValue}
                    onFilterEdited={onFilterEditedHandler}
                    toggleFilterEditable={toggleFilterEditableTrue}
                    editable={true}
                />
            ).container
            expect(getByDisplayValue(container, 'test query')).toBeTruthy()
        })

        it('calls onFilterEdited with escaped and quoted value', () => {
            container = render(
                <FilterInput
                    {...defaultProps}
                    mapKey="content"
                    filterType={FilterType.content}
                    onFilterEdited={defaultProps.onFilterEdited}
                    editable={true}
                />
            ).container

            const inputElement = container.querySelector('.filter-input__input-field')
            expect(inputElement).toBeTruthy()
            fireEvent.click(inputElement!)
            fireEvent.change(inputElement!, { target: { value: '"""' } })
            const confirmButton = container.querySelector('.check-button__btn')
            expect(confirmButton).toBeTruthy()
            fireEvent.click(confirmButton!)
            expect(defaultProps.onFilterEdited.calledWith('content', '"\\"\\"\\""')).toBe(true)
        })

        it('calls onFilterEdited with quoted value', () => {
            container = render(
                <FilterInput
                    {...defaultProps}
                    mapKey="after"
                    filterType={FilterType.after}
                    onFilterEdited={defaultProps.onFilterEdited}
                    editable={true}
                />
            ).container

            const inputElement = container.querySelector('.filter-input__input-field')
            expect(inputElement).toBeTruthy()
            fireEvent.click(inputElement!)
            fireEvent.change(inputElement!, { target: { value: '2 months ago' } })
            const confirmButton = container.querySelector('.check-button__btn')
            expect(confirmButton).toBeTruthy()
            fireEvent.click(confirmButton!)
            expect(defaultProps.onFilterEdited.calledWith('after', '"2 months ago"')).toBe(true)
        })
    })
})
