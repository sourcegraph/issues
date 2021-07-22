import { render } from '@testing-library/react'
import { createLocation, createMemoryHistory } from 'history'
import React from 'react'
import { MemoryRouter } from 'react-router'

import { setLinkComponent } from '@sourcegraph/shared/src/components/Link'
import { extensionsController, NOOP_SETTINGS_CASCADE } from '@sourcegraph/shared/src/util/searchTestHelpers'

import { SearchPatternType } from '../graphql-operations'
import {
    mockFetchAutoDefinedSearchContexts,
    mockFetchSearchContexts,
    mockGetUserSearchContextNamespaces,
} from '../searchContexts/testHelpers'
import { ThemePreference } from '../theme'

import { GlobalNavbar } from './GlobalNavbar'

jest.mock('../search/input/SearchNavbarItem', () => ({ SearchNavbarItem: 'SearchNavbarItem' }))
jest.mock('../components/branding/BrandLogo', () => ({ BrandLogo: 'BrandLogo' }))

const PROPS: React.ComponentProps<typeof GlobalNavbar> = {
    authenticatedUser: null,
    authRequired: false,
    extensionsController,
    location: createLocation('/'),
    history: createMemoryHistory(),
    keyboardShortcuts: [],
    isSourcegraphDotCom: false,
    navbarSearchQueryState: { query: 'q' },
    onNavbarQueryChange: () => undefined,
    onThemePreferenceChange: () => undefined,
    isLightTheme: true,
    themePreference: ThemePreference.Light,
    parsedSearchQuery: 'r:golang/oauth2 test f:travis',
    patternType: SearchPatternType.literal,
    setPatternType: () => undefined,
    caseSensitive: false,
    setCaseSensitivity: () => undefined,
    platformContext: {} as any,
    settingsCascade: NOOP_SETTINGS_CASCADE,
    batchChangesEnabled: false,
    enableCodeMonitoring: false,
    telemetryService: {} as any,
    hideNavLinks: true, // used because reactstrap Popover is incompatible with react-test-renderer
    isExtensionAlertAnimating: false,
    showSearchBox: true,
    versionContext: undefined,
    setVersionContext: () => Promise.resolve(),
    availableVersionContexts: [],
    showSearchContext: false,
    showSearchContextManagement: false,
    selectedSearchContextSpec: '',
    setSelectedSearchContextSpec: () => undefined,
    defaultSearchContextSpec: '',
    variant: 'default',
    globbing: false,
    showOnboardingTour: false,
    branding: undefined,
    routes: [],
    fetchAutoDefinedSearchContexts: mockFetchAutoDefinedSearchContexts(),
    fetchSearchContexts: mockFetchSearchContexts,
    hasUserAddedRepositories: false,
    hasUserAddedExternalServices: false,
    getUserSearchContextNamespaces: mockGetUserSearchContextNamespaces,
    featureFlags: new Map(),
}

describe('GlobalNavbar', () => {
    setLinkComponent(({ children, ...props }) => <a {...props}>{children}</a>)
    afterAll(() => setLinkComponent(() => null)) // reset global env for other tests

    test('default', () => {
        const { asFragment } = render(
            <MemoryRouter>
                <GlobalNavbar {...PROPS} />
            </MemoryRouter>
        )
        expect(asFragment()).toMatchSnapshot()
    })

    test('low-profile', () => {
        const { asFragment } = render(
            <MemoryRouter>
                <GlobalNavbar {...PROPS} variant="low-profile" />
            </MemoryRouter>
        )
        expect(asFragment()).toMatchSnapshot()
    })

    test('no-search-input', () => {
        const { asFragment } = render(
            <MemoryRouter>
                <GlobalNavbar {...PROPS} variant="no-search-input" />
            </MemoryRouter>
        )
        expect(asFragment()).toMatchSnapshot()
    })
})
