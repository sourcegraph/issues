import React from 'react'
import { cleanup, fireEvent, getByDisplayValue, queryByTestId, render, waitFor } from '@testing-library/react'
import sinon from 'sinon'
import { QueryBuilder } from './QueryBuilder'
import { SearchPatternType } from '../../graphql-operations'

describe('QueryBuilder', () => {
    afterAll(cleanup)

    let onChange: sinon.SinonSpy<[string], void>
    let container: HTMLElement
    beforeEach(() => {
        onChange = sinon.spy((query: string) => {
            /* noop */
        })
        ;({ container } = render(
            <QueryBuilder
                onFieldsQueryChange={onChange}
                isSourcegraphDotCom={false}
                patternType={SearchPatternType.regexp}
            />
        ))
    })

    it('fires the onFieldsQueryChange prop handler with the `repo:` filter when updating the "Repository" field', () => {
        const repoField = container.querySelector('#query-builder-repo')!
        fireEvent.change(repoField, { target: { value: 'sourcegraph/sourcegraph' } })
        sinon.assert.calledOnce(onChange)
        sinon.assert.calledWith(onChange, 'repo:sourcegraph/sourcegraph')
    })

    it('fires the onFieldsQueryChange prop handler with the `file:` filter when updating the "File" field', () => {
        const fileField = container.querySelector('#query-builder-file')!
        fireEvent.change(fileField, { target: { value: 'web' } })
        sinon.assert.calledOnce(onChange)
        sinon.assert.calledWith(onChange, 'file:web')
    })

    it('fires the onFieldsQueryChange prop handler with the `case:` filter when updating the "Case" field', () => {
        const caseField = container.querySelector('#query-builder-case')!
        fireEvent.change(caseField, { target: { value: 'yes' } })
        sinon.assert.calledOnce(onChange)
        sinon.assert.calledWith(onChange, 'case:yes')
    })

    it('fires the onFieldsQueryChange prop handler with the patterns left untransformed when updating the "Patterns" field', () => {
        const patternsField = container.querySelector('#query-builder-patterns')!
        fireEvent.change(patternsField, { target: { value: '(open|close) file' } })
        sinon.assert.calledOnce(onChange)
        sinon.assert.calledWith(onChange, '(open|close) file')
    })

    it('field fires the onFieldsQueryChange prop handler with a multi-word term wrapped in double quotes when updating the "Exact match"', () => {
        const exactMatchField = container.querySelector('#query-builder-exactMatch')!
        fireEvent.change(exactMatchField, { target: { value: 'foo bar baz' } })
        sinon.assert.calledOnce(onChange)
        sinon.assert.calledWith(onChange, '"foo bar baz"')
    })

    it('field fires the onFieldsQueryChange prop handler with a single-word term wrapped in double quotes when updating the "Exact match"', () => {
        const exactMatchField = container.querySelector('#query-builder-exactMatch')!
        fireEvent.change(exactMatchField, { target: { value: 'open(' } })
        sinon.assert.calledOnce(onChange)
        sinon.assert.calledWith(onChange, '"open("')
    })

    it('checks that the "Author", "Before", "After", and "Message" fields do not exist if the search type is set to code search', () => {
        expect(queryByTestId(container, 'test-author')).toBeNull()
        expect(queryByTestId(container, 'test-after')).toBeNull()
        expect(queryByTestId(container, 'test-before')).toBeNull()
        expect(queryByTestId(container, 'test-message')).toBeNull()
    })

    it('checks that the "Author", "Before", "After", and "Message" fields exist if the search type is set to diff search', async () => {
        const typeField = getByDisplayValue(container, 'Code (default)')
        fireEvent.change(typeField, { target: { value: 'diff' } })

        await waitFor(() => queryByTestId(container, 'test-author'))
        expect(queryByTestId(container, 'test-author')).toBeTruthy()
        await waitFor(() => queryByTestId(container, 'test-after'))
        expect(queryByTestId(container, 'test-after')).toBeTruthy()
        await waitFor(() => queryByTestId(container, 'test-before'))
        expect(queryByTestId(container, 'test-before')).toBeTruthy()
        await waitFor(() => queryByTestId(container, 'test-message'))
        expect(queryByTestId(container, 'test-message')).toBeTruthy()
    })

    it('checks that the "Author", "Before", and "After" fields exist if type is commit', async () => {
        const typeField = getByDisplayValue(container, 'Code (default)')
        fireEvent.change(typeField, { target: { value: 'commit' } })

        await waitFor(() => queryByTestId(container, 'test-author'))
        expect(queryByTestId(container, 'test-author')).toBeTruthy()
        await waitFor(() => queryByTestId(container, 'test-after'))
        expect(queryByTestId(container, 'test-after')).toBeTruthy()
        await waitFor(() => queryByTestId(container, 'test-before'))
        expect(queryByTestId(container, 'test-before')).toBeTruthy()
        await waitFor(() => queryByTestId(container, 'test-message'))
        expect(queryByTestId(container, 'test-message')).toBeTruthy()
    })

    it('fires the onFieldsQueryChange prop handler with the "author:" filter when updating the "Author" field', async () => {
        const typeField = getByDisplayValue(container, 'Code (default)')
        fireEvent.change(typeField, { target: { value: 'diff' } })

        await waitFor(() => queryByTestId(container, 'test-author'))
        const authorField = container.querySelector('#query-builder-author')!
        fireEvent.change(authorField, { target: { value: 'alice' } })
        sinon.assert.calledTwice(onChange)
        sinon.assert.calledWith(onChange, 'type:diff author:alice')
    })

    it('fires the onFieldsQueryChange prop handler with the "after:" filter when updating the "After" field ', async () => {
        const typeField = getByDisplayValue(container, 'Code (default)')
        fireEvent.change(typeField, { target: { value: 'diff' } })

        await waitFor(() => queryByTestId(container, 'test-after'))
        const afterField = container.querySelector('#query-builder-after')!
        fireEvent.change(afterField, { target: { value: '1 year ago' } })

        sinon.assert.calledTwice(onChange)
        sinon.assert.calledWith(onChange, 'type:diff after:"1 year ago"')
    })

    it('fires the onFieldsQueryChange prop handler with the "before:" filter when updating the "Before" field', async () => {
        const typeField = getByDisplayValue(container, 'Code (default)')
        fireEvent.change(typeField, { target: { value: 'diff' } })

        await waitFor(() => queryByTestId(container, 'test-before'))
        const beforeField = container.querySelector('#query-builder-before')!
        fireEvent.change(beforeField, { target: { value: '1 year ago' } })

        sinon.assert.calledTwice(onChange)
        sinon.assert.calledWith(onChange, 'type:diff before:"1 year ago"')
    })

    it('fires the onFieldsQueryChange prop handler with the "message:" filter when updating the "Message" field', async () => {
        const typeField = getByDisplayValue(container, 'Code (default)')
        fireEvent.change(typeField, { target: { value: 'diff' } })

        await waitFor(() => queryByTestId(container, 'test-message'))
        const messageField = container.querySelector('#query-builder-message')!
        fireEvent.change(messageField, { target: { value: 'fix issue' } })

        sinon.assert.calledTwice(onChange)
        sinon.assert.calledWith(onChange, 'type:diff message:"fix issue"')
    })
})

describe('QueryBuilder in literal mode', () => {
    afterAll(cleanup)

    let onChange: sinon.SinonSpy<[string], void>
    let container: HTMLElement
    beforeEach(() => {
        onChange = sinon.spy((query: string) => {
            /* noop */
        })
        ;({ container } = render(
            <QueryBuilder
                onFieldsQueryChange={onChange}
                isSourcegraphDotCom={false}
                patternType={SearchPatternType.literal}
            />
        ))
    })

    it('in literal mode, fires the onFieldsQueryChange prop handler with a single-word term not wrapped in double quotes when updating the "Exact match"', () => {
        const exactMatchField = container.querySelector('#query-builder-exactMatch')!
        fireEvent.change(exactMatchField, { target: { value: 'open(' } })
        sinon.assert.calledOnce(onChange)
        sinon.assert.calledWith(onChange, 'open(')
    })
    it('in literal mode, fires the onFieldsQueryChange prop handler with a multi-word term not wrapped in double-quotes when updating the "Exact match"', () => {
        const exactMatchField = container.querySelector('#query-builder-exactMatch')!
        fireEvent.change(exactMatchField, { target: { value: 'foo bar' } })
        sinon.assert.calledOnce(onChange)
        sinon.assert.calledWith(onChange, 'foo bar')
    })
})
