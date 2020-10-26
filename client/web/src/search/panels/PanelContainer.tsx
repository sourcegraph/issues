import * as React from 'react'
import classNames from 'classnames'

interface Props {
    title: string
    state: 'loading' | 'populated' | 'empty'
    // the content displayed when state is 'loading'
    loadingContent?: JSX.Element
    // the content displayed when state is 'empty'
    emptyContent?: JSX.Element
    // the content displayed when state is 'populated'
    populatedContent: JSX.Element
    actionButtons?: JSX.Element
    className?: string
}

export const PanelContainer: React.FunctionComponent<Props> = ({
    title,
    state,
    loadingContent = <></>,
    emptyContent = <></>,
    populatedContent,
    actionButtons,
    className,
}) => (
    <div className={classNames(className, 'panel-container', 'd-flex', 'flex-column')}>
        <div className="panel-container__header d-flex border-bottom">
            <h4 className="panel-container__header-text">{title}</h4>
            {actionButtons}
        </div>

        <div className="panel-container__content h-100">
            {state === 'loading' && loadingContent}
            {state === 'populated' && populatedContent}
            {state === 'empty' && emptyContent}
        </div>
    </div>
)
