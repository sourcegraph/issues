import { action } from '@storybook/addon-actions'
import { radios, text } from '@storybook/addon-knobs'
import { storiesOf } from '@storybook/react'
import React, { useState } from 'react'
import { CompletionList } from 'sourcegraph'
import { CompletionWidget, CompletionWidgetProps } from './CompletionWidget'
import webStyles from '../../../../web/src/main.scss'

const onSelectItem = action('onSelectItem')

// Disable keyboard shortcuts because in CompletionWidget the cursor is in a contenteditable element,
// which Storybook doesn't consider to be an input, so it intercepts keyboard shortcuts instead of
// propagating them to the CompletionWidget element.
const { add } = storiesOf('shared/CompletionWidget', module)
    .addParameters({ options: { enableShortcuts: false } })
    .addDecorator(story => {
        const theme = radios('Theme', { Light: 'light', Dark: 'dark' }, 'light')
        document.body.classList.toggle('theme-light', theme === 'light')
        document.body.classList.toggle('theme-dark', theme === 'dark')

        return (
            <>
                <div>{story()}</div>
                <style>{webStyles}</style>
            </>
        )
    })

const completionWidgetListItemClassName = 'completion-widget-dropdown__item d-flex align-items-center p-2'
const StyledCompletionWidget: React.FunctionComponent<CompletionWidgetProps> = props => (
    <CompletionWidget
        {...props}
        listClassName="completion-widget-dropdown d-block list-unstyled rounded border p-0 m-0 mt-3"
        listItemClassName={completionWidgetListItemClassName}
        selectedListItemClassName="completion-widget-dropdown__item--selected"
        loadingClassName={completionWidgetListItemClassName}
        noResultsClassName={completionWidgetListItemClassName}
    />
)
StyledCompletionWidget.displayName = 'StyledCompletionWidget'

add('interactive', () => {
    const CompletionWidgetInteractive: React.FunctionComponent = () => {
        const [element, setElement] = useState<HTMLTextAreaElement | null>(null)
        return (
            <div className="position-relative p-5">
                {element && (
                    <StyledCompletionWidget
                        completionListOrError={
                            { items: [{ label: 'alice' }, { label: 'bob' }, { label: 'carol' }] } as CompletionList
                        }
                        textArea={element}
                        onSelectItem={onSelectItem}
                        listClassName="list-unstyled"
                    />
                )}
                <textarea
                    ref={setElement}
                    rows={8}
                    className="w-100"
                    defaultValue={text('Initial value', 'hello, world!')}
                />
            </div>
        )
    }
    return <CompletionWidgetInteractive />
})
