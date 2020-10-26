import React, { useState } from 'react'
import { storiesOf } from '@storybook/react'
import { OptionsPage, OptionsPageProps } from './OptionsPage'
import brandedStyles from '../../branded.scss'
import { Observable, of } from 'rxjs'
import { action } from '@storybook/addon-actions'
import { boolean, text } from '@storybook/addon-knobs'
import GithubIcon from 'mdi-react/GithubIcon'
import { BrandedStory } from '../../../../branded/src/components/BrandedStory'
import { subtypeOf } from '../../../../shared/src/util/types'

const validateSourcegraphUrl = (): Observable<string | undefined> => of(undefined)
const invalidSourcegraphUrl = (): Observable<string | undefined> => of('Arbitrary error string')

const commonProps = subtypeOf<Partial<OptionsPageProps>>()({
    onChangeOptionFlag: action('onChangeOptionFlag'),
    optionFlags: [
        { key: 'allowErrorReporting', label: 'Allow error reporting', value: false },
        { key: 'experimentalLinkPreviews', label: 'Experimental link previews', value: false },
        { key: 'experimentalTextFieldCompletion', label: 'Experimental text field completion', value: false },
    ],
    version: text('version', '0.0.0'),
    onChangeSourcegraphUrl: action('onChangeSourcegraphUrl'),
})

const requestPermissionsHandler = action('requestPermission')

storiesOf('browser/Options/OptionsPage', module)
    .addDecorator(story => <BrandedStory styles={brandedStyles}>{() => story()}</BrandedStory>)
    .add('Default', () => (
        <OptionsPage
            {...commonProps}
            showPrivateRepositoryAlert={boolean('isCurrentRepositoryPrivate', false)}
            showSourcegraphCloudAlert={boolean('showSourcegraphCloudAlert', false)}
            validateSourcegraphUrl={validateSourcegraphUrl}
            onToggleActivated={action('onToggleActivated')}
            isActivated={true}
            sourcegraphUrl={text('sourcegraphUrl', 'https://sourcegraph.com')}
            isFullPage={true}
        />
    ))
    .add('Interactive', () => {
        const [isActivated, setIsActivated] = useState(false)
        return (
            <OptionsPage
                {...commonProps}
                isActivated={isActivated}
                onToggleActivated={setIsActivated}
                validateSourcegraphUrl={validateSourcegraphUrl}
                sourcegraphUrl={text('sourcegraphUrl', 'https://sourcegraph.com')}
                showPrivateRepositoryAlert={boolean('showPrivateRepositoryAlert', false)}
                showSourcegraphCloudAlert={boolean('showSourcegraphCloudAlert', false)}
                isFullPage={true}
            />
        )
    })
    .add('URL validation error', () => {
        const [isActivated, setIsActivated] = useState(false)
        return (
            <OptionsPage
                {...commonProps}
                isActivated={isActivated}
                onToggleActivated={setIsActivated}
                validateSourcegraphUrl={invalidSourcegraphUrl}
                sourcegraphUrl={text('sourcegraphUrl', 'https://not-sourcegraph.com')}
                isFullPage={true}
            />
        )
    })
    .add('Asking for permission', () => (
        <OptionsPage
            {...commonProps}
            validateSourcegraphUrl={validateSourcegraphUrl}
            onToggleActivated={action('onToggleActivated')}
            isActivated={true}
            sourcegraphUrl={text('sourcegraphUrl', 'https://sourcegraph.com')}
            isFullPage={true}
            currentHost="github.com"
            permissionAlert={{ name: 'GitHub', icon: GithubIcon }}
            requestPermissionsHandler={requestPermissionsHandler}
        />
    ))
    .add('On private repository', () => (
        <OptionsPage
            {...commonProps}
            validateSourcegraphUrl={validateSourcegraphUrl}
            onToggleActivated={action('onToggleActivated')}
            isActivated={true}
            sourcegraphUrl={text('sourcegraphUrl', 'https://sourcegraph.com')}
            isFullPage={true}
            currentHost="github.com"
            showPrivateRepositoryAlert={true}
            requestPermissionsHandler={requestPermissionsHandler}
        />
    ))
    .add('On Sourcegraph Cloud', () => (
        <OptionsPage
            {...commonProps}
            validateSourcegraphUrl={validateSourcegraphUrl}
            onToggleActivated={action('onToggleActivated')}
            isActivated={true}
            sourcegraphUrl={text('sourcegraphUrl', 'https://sourcegraph.com')}
            isFullPage={true}
            currentHost="sourcegraph.com"
            requestPermissionsHandler={requestPermissionsHandler}
            showSourcegraphCloudAlert={true}
        />
    ))
