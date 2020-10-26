import React from 'react'
import * as GQL from '../../../../shared/src/graphql/schema'
import classNames from 'classnames'
import { Link } from '../../../../shared/src/components/Link'

/**
 * Use a multi-column layout for tree entries when there are at least this many. See TreePage.scss
 * for more information.
 */
const MIN_ENTRIES_FOR_COLUMN_LAYOUT = 6

const TreeEntry: React.FunctionComponent<{
    isDir: boolean
    name: string
    parentPath: string
    url: string
}> = ({ isDir, name, parentPath, url }) => {
    const filePath = parentPath ? parentPath + '/' + name : name
    return (
        <Link
            to={url}
            className={classNames(
                'tree-entry',
                isDir && 'font-weight-bold',
                `test-tree-entry-${isDir ? 'directory' : 'file'}`
            )}
            title={filePath}
        >
            {name}
            {isDir && '/'}
        </Link>
    )
}
export const TreeEntriesSection: React.FunctionComponent<{
    parentPath: string
    entries: Pick<GQL.ITreeEntry, 'name' | 'isDirectory' | 'url' | 'path'>[]
}> = ({ parentPath, entries }) => {
    const directChildren = entries.filter(entry => entry.path === [parentPath, entry.name].filter(Boolean).join('/'))
    if (directChildren.length === 0) {
        return null
    }
    return (
        <div
            className={
                directChildren.length > MIN_ENTRIES_FOR_COLUMN_LAYOUT ? 'tree-entries-section--columns' : undefined
            }
        >
            {directChildren.map((entry, index) => (
                <TreeEntry
                    key={entry.name + String(index)}
                    isDir={entry.isDirectory}
                    name={entry.name}
                    parentPath={parentPath}
                    url={entry.url}
                />
            ))}
        </div>
    )
}
