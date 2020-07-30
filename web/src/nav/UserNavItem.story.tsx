import { action } from '@storybook/addon-actions'
import { storiesOf } from '@storybook/react'
import * as H from 'history'
import React, { useCallback } from 'react'
import { MemoryRouter } from 'react-router'
import * as GQL from '../../../shared/src/graphql/schema'
import { ThemePreference } from '../theme'
import { UserNavItem } from './UserNavItem'
import webStyles from '../SourcegraphWebApp.scss'

const onThemePreferenceChange = action('onThemePreferenceChange')

const { add } = storiesOf('web/UserNavItem', module).addDecorator(story => (
    <>
        <style>{webStyles}</style>
        <div className="theme-light">{story()}</div>
    </>
))

const OpenUserNavItem: React.FunctionComponent<UserNavItem['props']> = props => {
    const openDropdown = useCallback((userNavItem: UserNavItem | null) => {
        if (userNavItem) {
            userNavItem.setState({ isOpen: true })
        }
    }, [])
    return <UserNavItem {...props} ref={openDropdown} />
}

add(
    'Site admin',
    () => (
        <MemoryRouter>
            <OpenUserNavItem
                authenticatedUser={{
                    username: 'alice',
                    avatarURL: null,
                    session: { __typename: 'Session', canSignOut: true },
                    settingsURL: '#',
                    siteAdmin: true,
                    organizations: {
                        __typename: 'OrgConnection',
                        totalCount: 3,
                        nodes: [
                            { id: '0', settingsURL: '#', displayName: 'Acme Corp' },
                            { id: '1', settingsURL: '#', displayName: 'Beta Inc' },
                        ] as GQL.Org[],
                    },
                }}
                isLightTheme={true}
                themePreference={ThemePreference.Light}
                location={H.createMemoryHistory().location}
                onThemePreferenceChange={onThemePreferenceChange}
                showDotComMarketing={true}
            />
        </MemoryRouter>
    ),
    {
        design: {
            type: 'figma',
            url: 'https://www.figma.com/file/BkY8Ak997QauG0Iu2EqArv/Sourcegraph-Components?node-id=368%3A834',
        },
    }
)
