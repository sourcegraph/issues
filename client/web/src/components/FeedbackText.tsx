import * as React from 'react'

interface Props {
    headerText?: React.ReactNode
    footerText?: React.ReactNode
    className?: string
}

export const FeedbackText: React.FunctionComponent<Props> = (props: Props) => (
    <p className={`feedback-text ${props.className || ''}`}>
        {props.headerText || 'Questions/feedback?'} Contact us at{' '}
        <a href="https://twitter.com/srcgraph" target="_blank" rel="noopener noreferrer">
            @srcgraph
        </a>{' '}
        or{' '}
        <a href="mailto:support@sourcegraph.com" target="_blank" rel="noopener noreferrer">
            support@sourcegraph.com
        </a>
        , or file issues on our{' '}
        <a href="https://github.com/sourcegraph/issues/issues" target="_blank" rel="noopener noreferrer">
            public issue tracker
        </a>
        . {props.footerText}
    </p>
)
