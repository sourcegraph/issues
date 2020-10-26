import classNames from 'classnames'
import React, { useMemo } from 'react'
import { highlightCodeSafe } from '../../../shared/src/util/markdown'

interface CodeSnippetProps {
    /** The code to be displayed. */
    code: string
    /** Hint to the language, used for syntax-highlighting the code-snippet. */
    language: string

    className?: string
}

export const CodeSnippet: React.FunctionComponent<CodeSnippetProps> = ({ code, language, className }) => {
    const highlightedInput = useMemo(() => ({ __html: highlightCodeSafe(code, language) }), [code, language])
    return (
        <pre className={classNames('bg-code rounded p-3', className)}>
            <code dangerouslySetInnerHTML={highlightedInput} />
        </pre>
    )
}
