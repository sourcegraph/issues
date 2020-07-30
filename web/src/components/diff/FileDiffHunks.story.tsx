import { storiesOf } from '@storybook/react'
import { radios, boolean } from '@storybook/addon-knobs'
import React from 'react'
import { FileDiffHunks } from './FileDiffHunks'
import { createMemoryHistory } from 'history'
import webStyles from '../../SourcegraphWebApp.scss'
import { DiffHunkLineType, FileDiffHunk } from '../../../../shared/src/graphql/schema'

export const DEMO_HUNKS: FileDiffHunk[] = [
    {
        __typename: 'FileDiffHunk',
        oldRange: { __typename: 'FileDiffHunkRange', lines: 7, startLine: 3 },
        newRange: { __typename: 'FileDiffHunkRange', lines: 7, startLine: 3 },
        oldNoNewlineAt: false,
        section: 'func awesomeness(param string) (int, error) {',
        highlight: {
            __typename: 'HighlightedDiffHunkBody',
            aborted: false,
            lines: [
                {
                    kind: DiffHunkLineType.UNCHANGED,
                    html: '    v, err := makeAwesome()',
                    __typename: 'HighlightedDiffHunkLine',
                },
                {
                    kind: DiffHunkLineType.UNCHANGED,
                    html: '    if err != nil {',
                    __typename: 'HighlightedDiffHunkLine',
                },
                {
                    kind: DiffHunkLineType.UNCHANGED,
                    html: '        fmt.Printf("wow: %v", err)',
                    __typename: 'HighlightedDiffHunkLine',
                },
                {
                    kind: DiffHunkLineType.DELETED,
                    html: '        return err',
                    __typename: 'HighlightedDiffHunkLine',
                },
                {
                    kind: DiffHunkLineType.ADDED,
                    html: '        return nil, err',
                    __typename: 'HighlightedDiffHunkLine',
                },
                { kind: DiffHunkLineType.UNCHANGED, html: '    }', __typename: 'HighlightedDiffHunkLine' },
                {
                    kind: DiffHunkLineType.UNCHANGED,
                    html: '    return v.Score, nil',
                    __typename: 'HighlightedDiffHunkLine',
                },
                { kind: DiffHunkLineType.UNCHANGED, html: '}', __typename: 'HighlightedDiffHunkLine' },
            ],
        },
        body: '',
    },
]

const { add } = storiesOf('web/FileDiffHunks', module).addDecorator(story => {
    // TODO find a way to do this globally for all stories and storybook itself.
    const theme = radios('Theme', { Light: 'light', Dark: 'dark' }, 'light')
    document.body.classList.toggle('theme-light', theme === 'light')
    document.body.classList.toggle('theme-dark', theme === 'dark')
    return (
        <>
            <style>{webStyles}</style>
            <div className="p-3 container">{story()}</div>
        </>
    )
})

add('One diff hunk', () => (
    <FileDiffHunks
        persistLines={boolean('persistLines', false)}
        fileDiffAnchor="abc"
        lineNumbers={boolean('lineNumbers', true)}
        isLightTheme={true}
        hunks={DEMO_HUNKS}
        className="abcdef"
        location={createMemoryHistory().location}
        history={createMemoryHistory()}
    />
))
