import * as H from 'history'
import { storiesOf } from '@storybook/react'
import React from 'react'
import { ActivationDropdown, ActivationDropdownProps } from './ActivationDropdown'
import { Activation } from './Activation'
import { boolean } from '@storybook/addon-knobs'
import { action } from '@storybook/addon-actions'
import webMainStyles from '../../../../web/src/SourcegraphWebApp.scss'
import { subtypeOf } from '../../util/types'

const { add } = storiesOf('shared/ActivationDropdown', module).addDecorator(story => (
    <>
        <style>{webMainStyles}</style>
        <div className="theme-light">{story()}</div>
    </>
))

const baseActivation: Activation = {
    steps: [
        {
            id: 'ConnectedCodeHost',
            title: 'Add repositories',
            detail: 'Configure Sourcegraph to talk to your code host and fetch a list of your repositories.',
        },
        {
            id: 'DidSearch',
            title: 'Search your code',
            detail: (
                <span>
                    Head to the <a href="/search">homepage</a> and perform a search query on your code.{' '}
                    <strong>Example:</strong> type 'lang:' and select a language
                </span>
            ),
        },
        {
            id: 'FoundReferences',
            title: 'Find some references',
            detail:
                'To find references of a token, navigate to a code file in one of your repositories, hover over a token to activate the tooltip, and then click "Find references".',
        },
        {
            id: 'EnabledSharing',
            title: 'Configure SSO or share with teammates',
            detail: 'Configure a single-sign on (SSO) provider or have at least one other teammate sign up.',
        },
    ],
    refetch: action('Refetch'),
    update: action('Update'),
    completed: undefined,
}
const history = H.createMemoryHistory({ keyLength: 0 })
const commonProps = subtypeOf<Partial<ActivationDropdownProps>>()({
    alwaysShow: true,
    history,
    // Make sure the dropdown is not rendered outside the theme-light container
    portal: false,
})

add('Loading', () => <ActivationDropdown {...commonProps} activation={baseActivation} />)
add('0/4 completed', () => (
    <ActivationDropdown
        {...commonProps}
        activation={{
            ...baseActivation,
            completed: {
                ConnectedCodeHost: boolean('ConnectedCodeHost', false),
                DidSearch: boolean('DidSearch', false),
                FoundReferences: boolean('FoundReferences', false),
                EnabledRepository: boolean('EnabledRepository', false),
            },
        }}
    />
))
add('1/4 completed', () => (
    <ActivationDropdown
        {...commonProps}
        activation={{
            ...baseActivation,
            completed: {
                ConnectedCodeHost: boolean('ConnectedCodeHost', true),
                DidSearch: boolean('DidSearch', false),
                FoundReferences: boolean('FoundReferences', false),
                EnabledRepository: boolean('EnabledRepository', false),
            },
        }}
    />
))
