import { storiesOf } from '@storybook/react'
import { boolean } from '@storybook/addon-knobs'
import React from 'react'
import { FileDiffNode } from './FileDiffNode'
import { DEMO_HUNKS } from './FileDiffHunks.story'
import { FileDiffFields } from '../../graphql-operations'
import { WebStory } from '../WebStory'

export const FILE_DIFF_NODES: FileDiffFields[] = [
    {
        hunks: DEMO_HUNKS,
        internalID: 'abcdef123',
        stat: { added: 0, changed: 1, deleted: 0 },
        oldFile: null,
        newFile: {
            __typename: 'VirtualFile',
            binary: false,
            byteSize: 2,
        },
        mostRelevantFile: {
            __typename: 'VirtualFile',
            url: '/new_file.md',
        },
        newPath: 'new_file.md',
        oldPath: null,
    },
    {
        hunks: DEMO_HUNKS,
        internalID: 'abcdef123',
        stat: { added: 0, changed: 1, deleted: 0 },
        newFile: null,
        oldFile: {
            __typename: 'VirtualFile',
            binary: false,
            byteSize: 2,
        },
        mostRelevantFile: {
            __typename: 'VirtualFile',
            url: '/deleted_file.md',
        },
        newPath: null,
        oldPath: 'deleted_file.md',
    },
    {
        hunks: [],
        internalID: 'abcdef123',
        stat: { added: 0, changed: 0, deleted: 0 },
        oldFile: null,
        newFile: {
            __typename: 'VirtualFile',
            binary: true,
            byteSize: 1280,
        },
        mostRelevantFile: {
            __typename: 'VirtualFile',
            url: '/new_file.md',
        },
        newPath: 'new_file.md',
        oldPath: null,
    },
    {
        hunks: [],
        internalID: 'abcdef123',
        stat: { added: 0, changed: 0, deleted: 0 },
        newFile: null,
        oldFile: {
            __typename: 'VirtualFile',
            binary: true,
            byteSize: 1280,
        },
        mostRelevantFile: {
            __typename: 'VirtualFile',
            url: '/deleted_file.md',
        },
        newPath: null,
        oldPath: 'deleted_file.md',
    },
    {
        hunks: DEMO_HUNKS,
        internalID: 'abcdef123',
        stat: { added: 0, changed: 1, deleted: 0 },
        oldFile: {
            __typename: 'VirtualFile',
            binary: false,
            byteSize: 0,
        },
        newFile: {
            __typename: 'VirtualFile',
            binary: false,
            byteSize: 0,
        },
        mostRelevantFile: {
            __typename: 'VirtualFile',
            url: '/existing_file.md',
        },
        newPath: 'existing_file.md',
        oldPath: 'existing_file.md',
    },
    {
        hunks: DEMO_HUNKS,
        internalID: 'abcdef123',
        stat: { added: 0, changed: 1, deleted: 0 },
        oldFile: {
            __typename: 'GitBlob',
            binary: false,
            byteSize: 0,
        },
        newFile: {
            __typename: 'GitBlob',
            binary: false,
            byteSize: 0,
        },
        mostRelevantFile: {
            __typename: 'GitBlob',
            url: 'http://test.test/gitblob',
        },
        newPath: 'existing_git_file.md',
        oldPath: 'existing_git_file.md',
    },
    {
        hunks: DEMO_HUNKS,
        internalID: 'abcdef123',
        stat: { added: 0, changed: 1, deleted: 0 },
        oldFile: {
            __typename: 'VirtualFile',
            binary: false,
            byteSize: 0,
        },
        newFile: {
            __typename: 'VirtualFile',
            binary: false,
            byteSize: 0,
        },
        mostRelevantFile: {
            __typename: 'VirtualFile',
            url: '/to.md',
        },
        newPath: 'to.md',
        oldPath: 'from.md',
    },
    {
        hunks: DEMO_HUNKS,
        internalID: 'abcdef123',
        stat: { added: 0, changed: 1, deleted: 0 },
        oldFile: {
            __typename: 'VirtualFile',
            binary: false,
            byteSize: 0,
        },
        newFile: {
            __typename: 'VirtualFile',
            binary: false,
            byteSize: 0,
        },
        mostRelevantFile: {
            __typename: 'VirtualFile',
            url: 'dir2/to.md',
        },
        newPath: 'dir2/to.md',
        oldPath: 'dir1/from.md',
    },
    {
        hunks: [],
        internalID: 'abcdef123',
        stat: { added: 0, changed: 0, deleted: 0 },
        oldFile: {
            __typename: 'VirtualFile',
            binary: false,
            byteSize: 0,
        },
        newFile: {
            __typename: 'VirtualFile',
            binary: false,
            byteSize: 0,
        },
        mostRelevantFile: {
            __typename: 'VirtualFile',
            url: '/to.md',
        },
        newPath: 'to.md',
        oldPath: 'from.md',
    },
]

const { add } = storiesOf('web/diffs/FileDiffNode', module).addDecorator(story => (
    <div className="p-3 container">{story()}</div>
))

add('All file node states overview', () => (
    <WebStory>
        {webProps => (
            <>
                {FILE_DIFF_NODES.map((node, index) => (
                    <FileDiffNode
                        {...webProps}
                        key={index}
                        persistLines={boolean('persistLines', false)}
                        lineNumbers={boolean('lineNumbers', true)}
                        node={node}
                        className="abcdef"
                    />
                ))}
            </>
        )}
    </WebStory>
))
