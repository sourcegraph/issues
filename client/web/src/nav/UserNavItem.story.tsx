import { action } from '@storybook/addon-actions'
import { boolean, select } from '@storybook/addon-knobs'
import { storiesOf } from '@storybook/react'
import React from 'react'
import { ThemePreference } from '../theme'
import { UserNavItem } from './UserNavItem'
import { WebStory } from '../components/WebStory'

const onThemePreferenceChange = action('onThemePreferenceChange')

const { add } = storiesOf('web/UserNavItem', module)

add(
    'Site admin',
    () => (
        <WebStory>
            {webProps => (
                <UserNavItem
                    {...webProps}
                    testIsOpen={true}
                    authenticatedUser={{
                        username: 'alice',
                        avatarURL: null,
                        session: { canSignOut: true },
                        settingsURL: '#',
                        siteAdmin: true,
                        organizations: {
                            nodes: [
                                {
                                    id: '0',
                                    name: 'acme',
                                    displayName: 'Acme Corp',
                                    url: '/organizations/acme',
                                    settingsURL: '/organizations/acme/settings',
                                },
                                {
                                    id: '1',
                                    name: 'beta',
                                    displayName: 'Beta Inc',
                                    url: '/organizations/beta',
                                    settingsURL: '/organizations/beta/settings',
                                },
                            ],
                        },
                    }}
                    themePreference={webProps.isLightTheme ? ThemePreference.Light : ThemePreference.Dark}
                    onThemePreferenceChange={onThemePreferenceChange}
                    showDotComMarketing={boolean('showDotComMarketing', true)}
                    isExtensionAlertAnimating={false}
                    codeHostIntegrationMessaging={select(
                        'codeHostIntegrationMessaging',
                        ['browser-extension', 'native-integration'] as const,
                        'browser-extension'
                    )}
                />
            )}
        </WebStory>
    ),
    {
        design: {
            type: 'figma',
            url: 'https://www.figma.com/file/HWLuLefEdev5KYtoEGHjFj/Sourcegraph-Components-Contractor?node-id=1346%3A0',
        },
    }
)
