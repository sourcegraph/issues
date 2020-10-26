import React from 'react'
import { createMemoryHistory } from 'history'
import { storiesOf } from '@storybook/react'
import { WebStory } from '../components/WebStory'
import { GlobalNavbar } from './GlobalNavbar'
import { NOOP_TELEMETRY_SERVICE } from '../../../shared/src/telemetry/telemetryService'
import { ThemePreference } from '../theme'
import { ThemeProps } from '../../../shared/src/theme'
import { SearchPatternType } from '../graphql-operations'
import { Services } from '../../../shared/src/api/client/services'
import { SourcegraphContext } from '../jscontext'
import { SuiteFunction } from 'mocha'
import { AuthenticatedUser } from '../auth'

window.context = { assetsRoot: 'https://sourcegraph.com/.assets' } as SourcegraphContext & SuiteFunction

const history = createMemoryHistory()

const defaultProps = (
    props: ThemeProps
): Omit<
    React.ComponentProps<typeof GlobalNavbar>,
    'authenticatedUser' | 'variant' | 'isSearchRelatedPage' | 'authRequired'
> => ({
    isSourcegraphDotCom: false,
    settingsCascade: {
        final: null,
        subjects: null,
    },
    location: history.location,
    history,
    extensionsController: {
        services: {} as Services,
    } as any,
    telemetryService: NOOP_TELEMETRY_SERVICE,
    themePreference: ThemePreference.Light,
    onThemePreferenceChange: () => undefined,
    setVersionContext: () => undefined,
    availableVersionContexts: [],
    globbing: false,
    patternType: SearchPatternType.literal,
    setPatternType: () => undefined,
    caseSensitive: false,
    setCaseSensitivity: () => undefined,
    platformContext: {} as any,
    keyboardShortcuts: [],
    filtersInQuery: {} as any,
    onFiltersInQueryChange: () => undefined,
    splitSearchModes: false,
    interactiveSearchMode: false,
    toggleSearchMode: () => undefined,
    copyQueryButton: false,
    versionContext: undefined,
    showOnboardingTour: false,
    isLightTheme: props.isLightTheme,
    navbarSearchQueryState: { cursorPosition: 0, query: '' },
    onNavbarQueryChange: () => {},
    isExtensionAlertAnimating: false,
    showCampaigns: true,
    activation: undefined,
    hideNavLinks: false,
})

const { add } = storiesOf('web/nav/GlobalNav', module)

add('Anonymous viewer', () => (
    <WebStory>
        {webProps => (
            <GlobalNavbar
                {...defaultProps(webProps)}
                authRequired={false}
                authenticatedUser={null}
                variant="default"
                isSearchRelatedPage={false}
            />
        )}
    </WebStory>
))

add('Auth required', () => (
    <WebStory>
        {webProps => (
            <GlobalNavbar
                {...defaultProps(webProps)}
                authRequired={true}
                authenticatedUser={null}
                variant="default"
                isSearchRelatedPage={false}
            />
        )}
    </WebStory>
))

add('Authenticated viewer', () => (
    <WebStory>
        {webProps => (
            <GlobalNavbar
                {...defaultProps(webProps)}
                authRequired={false}
                authenticatedUser={
                    { username: 'alice', organizations: { nodes: [{ name: 'acme' }] } } as AuthenticatedUser
                }
                variant="default"
                isSearchRelatedPage={false}
            />
        )}
    </WebStory>
))
