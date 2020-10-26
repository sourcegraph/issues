import * as H from 'history'
import React from 'react'
import { MemoryRouter } from 'react-router'
import { ThemePreference } from '../theme'
import { UserNavItem, UserNavItemProps } from './UserNavItem'
import { mount } from 'enzyme'

describe('UserNavItem', () => {
    const USER: UserNavItemProps['authenticatedUser'] = {
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
    }

    const history = H.createMemoryHistory({ keyLength: 0 })

    test('simple', () => {
        expect(
            mount(
                <MemoryRouter>
                    <UserNavItem
                        isLightTheme={true}
                        onThemePreferenceChange={() => undefined}
                        themePreference={ThemePreference.Light}
                        location={history.location}
                        authenticatedUser={USER}
                        showDotComMarketing={true}
                        isExtensionAlertAnimating={false}
                        codeHostIntegrationMessaging="browser-extension"
                    />
                </MemoryRouter>
            )
        ).toMatchSnapshot()
    })
})
