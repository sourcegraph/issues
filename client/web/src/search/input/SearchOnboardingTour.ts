/**
 * This file contains utility functions for the search onboarding tour.
 */
import Shepherd from 'shepherd.js'
import { eventLogger } from '../../tracking/eventLogger'
import { isEqual } from 'lodash'
import { LANGUAGES } from '../../../../shared/src/search/parser/filters'

export const HAS_CANCELLED_TOUR_KEY = 'has-cancelled-onboarding-tour'
export const HAS_SEEN_TOUR_KEY = 'has-seen-onboarding-tour'

export const defaultTourOptions: Shepherd.Tour.TourOptions = {
    useModalOverlay: false,
    defaultStepOptions: {
        arrow: true,
        classes: 'web-content tour-card card py-4 px-3 shadow-lg',
        popperOptions: {
            // Removes default behavior of autofocusing steps
            modifiers: [
                {
                    name: 'focusAfterRender',
                    enabled: false,
                },
                { name: 'offset', options: { offset: [0, 8] } },
            ],
        },
        attachTo: { on: 'bottom' },
        scrollTo: false,
    },
}
/**
 * generateStep creates the content for tooltips for the search tour. All steps that just contain
 * simple text should use this function to populate the step's `text` field.
 */
export function generateStepTooltip(options: {
    tour: Shepherd.Tour
    dangerousTitleHtml: string
    stepNumber: number
    totalStepCount: number
    description?: string
    additionalContent?: HTMLElement
}): HTMLElement {
    const element = document.createElement('div')
    element.className = `d-flex flex-column test-tour-step-${options.stepNumber}`
    const titleElement = document.createElement('h4')
    titleElement.innerHTML = options.dangerousTitleHtml
    titleElement.className = 'font-weight-bold'
    element.append(titleElement)
    if (options.description) {
        const descriptionElement = document.createElement('p')
        descriptionElement.textContent = options.description
        descriptionElement.className = 'tour-card__description mb-0'
        element.append(descriptionElement)
    }
    if (options.additionalContent) {
        const additionalContentContainer = document.createElement('div')
        additionalContentContainer.append(options.additionalContent)
        element.append(options.additionalContent)
    }
    const bottomRow = generateBottomRow(options.tour, options.stepNumber, options.totalStepCount)
    element.append(bottomRow)
    return element
}

/**
 * Generates the bottom row of the tooltip, which shows the current step number and a "close tour" button.
 *
 * @param tour the tour instance.
 * @param stepNumber the step number.
 */
export function generateBottomRow(tour: Shepherd.Tour, stepNumber: number, totalStepCount: number): HTMLElement {
    const closeTourButton = document.createElement('button')
    closeTourButton.className = 'btn btn-link p-0'
    closeTourButton.textContent = 'Close tour'
    closeTourButton.addEventListener('click', () => {
        tour.cancel()
        localStorage.setItem(HAS_CANCELLED_TOUR_KEY, 'true')
        eventLogger.log('CloseOnboardingTourClicked', { stage: stepNumber })
    })

    const bottomRow = document.createElement('div')
    bottomRow.className = 'd-flex justify-content-between mt-2'

    const stepNumberLabel = document.createElement('span')
    stepNumberLabel.className = 'font-weight-light font-italic'
    stepNumberLabel.textContent = `${stepNumber} of ${totalStepCount}`
    bottomRow.append(stepNumberLabel)

    bottomRow.append(closeTourButton)
    return bottomRow
}

/**
 * Generates the tooltip content for the first step in the tour.
 *
 * @param languageButtonHandler the handler for the "search a language" button.
 * @param repositoryButtonHandler the handler for the "search a repository" button.
 */
export function createStep1Tooltip(
    tour: Shepherd.Tour,
    languageButtonHandler: () => void,
    repositoryButtonHandler: () => void
): HTMLElement {
    const list = document.createElement('ul')
    list.className = 'my-4 list-dashed'
    const languageListItem = document.createElement('li')
    languageListItem.className = 'p-0 mb-2'

    const languageButton = document.createElement('button')
    languageButton.className = 'btn btn-link p-0 pl-1 test-tour-language-button'
    languageButton.textContent = 'Search a language'
    languageListItem.append(languageButton)
    languageButton.addEventListener('click', () => {
        languageButtonHandler()
        eventLogger.log('OnboardingTourLanguageOptionClicked')
    })
    const repositoryListItem = document.createElement('li')
    repositoryListItem.className = 'p-0 mb-2 test-tour-repo-button'
    const repositoryButton = document.createElement('button')
    repositoryButton.className = 'btn btn-link p-0 pl-1'
    repositoryButton.textContent = 'Search a repository'
    repositoryButton.addEventListener('click', () => {
        repositoryButtonHandler()
        eventLogger.log('OnboardingTourRepositoryOptionClicked')
    })
    repositoryListItem.append(repositoryButton)
    list.append(languageListItem)
    list.append(repositoryListItem)
    return generateStepTooltip({
        tour,
        dangerousTitleHtml: 'Code search tour',
        stepNumber: 1,
        totalStepCount: 5,
        description: 'How would you like to begin?',
        additionalContent: list,
    })
}

/**
 * Generates the tooltip content for the "add code" step in the repository path, which asks users to input their own terms into the query.
 *
 * @param tour the tour instance
 */
export function createAddCodeStepTooltip(tour: Shepherd.Tour): HTMLElement {
    return generateStepTooltip({
        tour,
        dangerousTitleHtml: 'Add code to your search',
        stepNumber: 3,
        totalStepCount: 5,
        description: 'Type the name of a function, variable or other code.',
    })
}

/**
 * Generates the tooltip content for the "add code" step in the language path, which asks users to input their own terms into the query.
 * It provides an example based on the language they selected in the previous step.
 *
 * @param tour the tour instance.
 * @param languageQuery the current query including a `lang:` filter. Used for language queries so we know what examples to suggest.
 * @param exampleCallback the callback to be run when clicking the example query.
 */
export function createAddCodeStepWithLanguageExampleTooltip(tour: Shepherd.Tour): HTMLElement {
    return generateStepTooltip({
        tour,
        dangerousTitleHtml: 'Add code to your search',
        stepNumber: 3,
        totalStepCount: 5,
        description: 'Type the name of a function, variable or other code.',
    })
}

/**
 * Determines whether a query contains a valid `lang:$LANGUAGE` query. There is an edge case where this will return true for
 * language names that are subsets of other languages (e.g. java is a subset of javascript). The caller should ensure there is
 * enough debouncing time for this edge case to be mitigated.
 */
export const isValidLangQuery = (query: string): boolean => LANGUAGES.map(lang => `lang:${lang}`).includes(query)

export const isCurrentTourStep = (step: string, tour?: Shepherd.Tour): boolean | undefined =>
    tour && isEqual(tour.getCurrentStep(), tour.getById(step))

export const advanceLangStep = (query: string, tour: Shepherd.Tour | undefined): void => {
    if (query !== 'lang:' && isValidLangQuery(query.trim()) && tour?.getById('filter-lang').isOpen()) {
        tour?.show('add-query-term')
    }
}

export const advanceRepoStep = (query: string, tour: Shepherd.Tour | undefined): void => {
    if (tour?.getById('filter-repository').isOpen() && query !== 'repo:') {
        tour?.show('add-query-term')
    }
}

export const runAdvanceLangOrRepoStep = (query: string, tour: Shepherd.Tour | undefined): void => {
    if (tour) {
        if (query !== 'lang:' && isValidLangQuery(query.trim()) && tour.getById('filter-lang').isOpen()) {
            advanceLangStep(query, tour)
        } else if (tour.getById('filter-repository').isOpen() && query !== 'repo:') {
            advanceRepoStep(query, tour)
        }
    }
}
