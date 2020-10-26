import { action } from '@storybook/addon-actions'
import { storiesOf } from '@storybook/react'
import * as H from 'history'
import React from 'react'
import { NOOP_TELEMETRY_SERVICE } from '../telemetry/telemetryService'
import { ActionItem, ActionItemComponentProps, ActionItemProps } from './ActionItem'
import { NEVER } from 'rxjs'
import webStyles from '../../../web/src/SourcegraphWebApp.scss'
import { subtypeOf } from '../util/types'

const { add } = storiesOf('shared/ActionItem', module).addDecorator(story => (
    <>
        <div className="p-4">{story()}</div>
        <style>{webStyles}</style>
    </>
))

const EXTENSIONS_CONTROLLER: ActionItemComponentProps['extensionsController'] = {
    executeCommand: () => new Promise(resolve => setTimeout(resolve, 750)),
}

const PLATFORM_CONTEXT: ActionItemComponentProps['platformContext'] = {
    forceUpdateTooltip: () => undefined,
    settings: NEVER,
}

const LOCATION: H.Location = { hash: '', pathname: '/', search: '', state: undefined }

const ICON_URL =
    'data:image/svg+xml,' +
    encodeURIComponent(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><text y=".9em" font-size="90">🚀</text></svg>'
    )

const onDidExecute = action('onDidExecute')

const commonProps = subtypeOf<Partial<ActionItemProps>>()({
    location: LOCATION,
    extensionsController: EXTENSIONS_CONTROLLER,
    platformContext: PLATFORM_CONTEXT,
    telemetryService: NOOP_TELEMETRY_SERVICE,
    iconClassName: 'icon-inline',
})

add('Noop action', () => (
    <ActionItem
        {...commonProps}
        action={{ id: 'a', command: undefined, actionItem: { label: 'Hello' } }}
        variant="actionItem"
    />
))

add('Command action', () => (
    <ActionItem
        {...commonProps}
        action={{ id: 'a', command: 'c', title: 'Hello', iconURL: ICON_URL }}
        telemetryService={NOOP_TELEMETRY_SERVICE}
        disabledDuringExecution={true}
        showLoadingSpinnerDuringExecution={true}
        showInlineError={true}
        onDidExecute={onDidExecute}
    />
))

add('Link action', () => (
    <ActionItem
        {...commonProps}
        action={{
            id: 'a',
            command: 'open',
            commandArguments: ['javascript:alert("link clicked")'],
            actionItem: { label: 'Hello' },
        }}
        variant="actionItem"
        onDidExecute={onDidExecute}
    />
))

add('Executing', () => {
    class ActionItemExecuting extends ActionItem {
        constructor(props: ActionItem['props']) {
            super(props)
            this.state.actionOrError = 'loading'
        }
    }
    return (
        <ActionItemExecuting
            {...commonProps}
            action={{ id: 'a', command: 'c', title: 'Hello', iconURL: ICON_URL }}
            disabledDuringExecution={true}
            showLoadingSpinnerDuringExecution={true}
            showInlineError={true}
        />
    )
})

add('Error', () => {
    class ActionItemWithError extends ActionItem {
        constructor(props: ActionItem['props']) {
            super(props)
            this.state.actionOrError = new Error('e')
        }
    }
    return (
        <ActionItemWithError
            {...commonProps}
            action={{ id: 'a', command: 'c', title: 'Hello', iconURL: ICON_URL }}
            disabledDuringExecution={true}
            showLoadingSpinnerDuringExecution={true}
            showInlineError={true}
        />
    )
})
