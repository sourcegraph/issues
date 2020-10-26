import React from 'react'
import { _fetchSavedSearches, authUser } from './utils'
import { cleanup, fireEvent, render } from '@testing-library/react'
import { NOOP_TELEMETRY_SERVICE } from '../../../../shared/src/telemetry/telemetryService'
import { SavedSearchesPanel } from './SavedSearchesPanel'
import { SearchPatternType } from '../../graphql-operations'

describe('SavedSearchesPanel', () => {
    afterAll(cleanup)

    let container: HTMLElement

    const defaultProps = {
        patternType: SearchPatternType.literal,
        authenticatedUser: authUser,
        fetchSavedSearches: _fetchSavedSearches,
        telemetryService: NOOP_TELEMETRY_SERVICE,
    }

    it('should show correct mode and number of entries when clicking on "my searches" and "all searches" buttons', () => {
        container = render(<SavedSearchesPanel {...defaultProps} />).container
        let savedSearchEntries = container.querySelectorAll('.test-saved-search-entry')
        expect(savedSearchEntries.length).toBe(2)
        const mySearchesButton = container.querySelector('.test-saved-search-panel-my-searches')!
        fireEvent.click(mySearchesButton)
        savedSearchEntries = container.querySelectorAll('.test-saved-search-entry')
        expect(savedSearchEntries.length).toBe(1)
        const allSearchesButton = container.querySelector('.test-saved-search-panel-all-searches')!
        fireEvent.click(allSearchesButton)
        savedSearchEntries = container.querySelectorAll('.test-saved-search-entry')
        expect(savedSearchEntries.length).toBe(2)
    })
})
