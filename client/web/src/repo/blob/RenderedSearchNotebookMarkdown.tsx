import { noop } from 'lodash'
import React, { useMemo } from 'react'

import { convertMarkdownToBlocks } from '../../search/notebook/convertMarkdownToBlocks'
import { SearchNotebook, SearchNotebookProps } from '../../search/notebook/SearchNotebook'

import styles from './RenderedSearchNotebookMarkdown.module.scss'

export const SEARCH_NOTEBOOK_FILE_EXTENSION = '.snb.md'

interface RenderedSearchNotebookMarkdownProps extends Omit<SearchNotebookProps, 'onSerializeBlocks' | 'blocks'> {
    markdown: string
}

export const RenderedSearchNotebookMarkdown: React.FunctionComponent<RenderedSearchNotebookMarkdownProps> = ({
    markdown,
    ...props
}) => {
    const blocks = useMemo(() => convertMarkdownToBlocks(markdown), [markdown])
    return (
        <div className={styles.renderedSearchNotebookMarkdownWrapper}>
            <div className={styles.renderedSearchNotebookMarkdown}>
                <SearchNotebook isReadOnly={true} blocks={blocks} {...props} onSerializeBlocks={noop} />
            </div>
        </div>
    )
}
