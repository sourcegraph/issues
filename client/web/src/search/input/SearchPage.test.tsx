import React from 'react'
import { authUser } from '../panels/utils'
import { cleanup, render } from '@testing-library/react'
import { createMemoryHistory } from 'history'
import { NOOP_TELEMETRY_SERVICE } from '../../../../shared/src/telemetry/telemetryService'
import { of } from 'rxjs'
import { SearchPage, SearchPageProps } from './SearchPage'
import { SearchPatternType } from '../../graphql-operations'
import { Services } from '../../../../shared/src/api/client/services'
import { ThemePreference } from '../../theme'

// Mock the Monaco input box to make this a shallow test
jest.mock('./SearchPageInput', () => ({
    SearchPageInput: () => null,
}))

describe('SearchPage', () => {
    afterAll(cleanup)

    let container: HTMLElement

    const history = createMemoryHistory()
    const defaultProps: SearchPageProps = {
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
        authenticatedUser: authUser,
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
        showRepogroupHomepage: false,
        showEnterpriseHomePanels: false,
        showOnboardingTour: false,
        showQueryBuilder: false,
        isLightTheme: true,
        fetchSavedSearches: () => of([]),
        fetchRecentSearches: () => of({ nodes: [], totalCount: 0, pageInfo: { hasNextPage: false, endCursor: null } }),
        fetchRecentFileViews: () => of({ nodes: [], totalCount: 0, pageInfo: { hasNextPage: false, endCursor: null } }),
    }

    it('should not show home panels if on Sourcegraph.com and showEnterpriseHomePanels disabled', () => {
        container = render(<SearchPage {...defaultProps} isSourcegraphDotCom={true} />).container
        const homePanels = container.querySelector('.home-panels')
        expect(homePanels).toBeFalsy()
    })

    it('should show home panels if on Sourcegraph.com and showEnterpriseHomePanels enabled', () => {
        container = render(<SearchPage {...defaultProps} isSourcegraphDotCom={true} showEnterpriseHomePanels={true} />)
            .container
        const homePanels = container.querySelector('.home-panels')
        expect(homePanels).toBeTruthy()
    })

    it('should show home panels if on Sourcegraph.com and showEnterpriseHomePanels enabled with user logged out', () => {
        container = render(
            <SearchPage
                {...defaultProps}
                isSourcegraphDotCom={true}
                showEnterpriseHomePanels={true}
                authenticatedUser={null}
            />
        ).container
        const homePanels = container.querySelector('.home-panels')
        expect(homePanels).toBeFalsy()
    })

    it('should not show home panels if showEnterpriseHomePanels disabled', () => {
        container = render(<SearchPage {...defaultProps} />).container
        const homePanels = container.querySelector('.home-panels')
        expect(homePanels).toBeFalsy()
    })

    it('should show home panels if showEnterpriseHomePanels enabled and not on Sourcegraph.com', () => {
        container = render(<SearchPage {...defaultProps} showEnterpriseHomePanels={true} />).container
        const homePanels = container.querySelector('.home-panels')
        expect(homePanels).toBeTruthy()
    })
})
