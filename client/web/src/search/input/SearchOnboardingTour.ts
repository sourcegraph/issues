/**
 * This file contains utility functions for the search onboarding tour.
 */
import * as H from 'history'
import { isEqual } from 'lodash'
import { useCallback, useEffect, useMemo, useState } from 'react'
import Shepherd from 'shepherd.js'
import Tour from 'shepherd.js/src/types/tour'
import { LANGUAGES } from '../../../../shared/src/search/query/filters'
import { scanSearchQuery } from '../../../../shared/src/search/query/scanner'
import { Token } from '../../../../shared/src/search/query/token'
import { useLocalStorage } from '../../../../shared/src/util/useLocalStorage'
import { daysActiveCount } from '../../marketing/util'
import { eventLogger } from '../../tracking/eventLogger'
import { QueryState } from '../helpers'
import { MonacoQueryInputProps } from './MonacoQueryInput'

export const HAS_CANCELLED_TOUR_KEY = 'has-cancelled-onboarding-tour'
export const HAS_COMPLETED_TOUR_KEY = 'has-completed-onboarding-tour'
export const HAS_SEEN_TOUR_KEY = 'has-seen-onboarding-tour'

const defaultTourOptions: Shepherd.Tour.TourOptions = {
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
function generateStepTooltip(options: {
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
function generateBottomRow(tour: Shepherd.Tour, stepNumber: number, totalStepCount: number): HTMLElement {
    const closeTourButton = document.createElement('button')
    closeTourButton.className = 'btn btn-link p-0 test-tour-close-button'
    closeTourButton.textContent = 'Close tour'
    closeTourButton.addEventListener('click', () => {
        tour.cancel()
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
function createStep1Tooltip(
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

type TourStepID = 'filter-repository' | 'filter-lang' | 'add-query-term'

const TOUR_STEPS = ['filter-repository', 'filter-lang', 'add-query-term'] as TourStepID[]

/**
 * Returns `true` if, while on the filter-(repository|lang) step,
 * the search query is a (repo|lang) filter with no value.
 */
const shouldTriggerSuggestions = (currentTourStep: TourStepID | undefined, queryTokens: Token[]): boolean => {
    if (queryTokens.length !== 1) {
        return false
    }
    const filterToken = queryTokens[0]
    if (filterToken.type !== 'filter' || filterToken.value !== undefined) {
        return false
    }
    return currentTourStep === 'filter-repository'
        ? filterToken.field.value === 'repo'
        : currentTourStep === 'filter-lang'
        ? filterToken.field.value === 'lang'
        : false
}

/**
 * Returns `true` if, while on the filter-(repository|lang) step,
 * the search query is a valid (repo|lang) filter followed by whitespace.
 * -
 */
const shouldAdvanceLangOrRepoStep = (currentTourStep: TourStepID | undefined, queryTokens: Token[]): boolean => {
    if (queryTokens.length !== 2) {
        return false
    }
    const [filterToken, whitespaceToken] = queryTokens
    if (filterToken.type !== 'filter' || whitespaceToken.type !== 'whitespace') {
        return false
    }
    if (currentTourStep === 'filter-repository') {
        return filterToken.field.value === 'repo' && filterToken.value !== undefined
    }
    if (currentTourStep === 'filter-lang') {
        return (
            filterToken.field.value === 'lang' &&
            filterToken.value?.type === 'literal' &&
            LANGUAGES.includes(filterToken.value?.value)
        )
    }
    return false
}

/**
 * Returns true if, while on the add-query-term step, the search query
 * contains a search pattern.
 */
const shouldShowSubmitSearch = (currentTourStep: TourStepID | undefined, queryTokens: Token[]): boolean =>
    currentTourStep === 'add-query-term' && queryTokens.some(({ type }) => type === 'pattern')

/**
 * A hook returning the current step ID of the Shepherd Tour.
 */
const useCurrentStep = (tour: Tour): TourStepID | undefined => {
    const [currentStep, setCurrentStep] = useState<TourStepID | undefined>()
    useEffect(() => {
        setCurrentStep(TOUR_STEPS.find(stepID => isEqual(tour.getCurrentStep(), tour.getById(stepID))))
        const listener = ({ step }: { step: Shepherd.Step }): void => {
            setCurrentStep(TOUR_STEPS.find(stepID => isEqual(step, tour.getById(stepID))))
        }
        tour.on('show', listener)
        return () => {
            tour.off('show', listener)
        }
    }, [tour, setCurrentStep])
    return currentStep
}

const useTourWithSteps = ({
    inputLocation,
    setQueryState,
}: Pick<UseSearchOnboardingTourOptions, 'inputLocation' | 'setQueryState'>): Tour => {
    const tour = useMemo(() => new Shepherd.Tour(defaultTourOptions), [])
    useEffect(() => {
        if (inputLocation === 'search-homepage') {
            tour.addSteps([
                {
                    id: 'start-tour',
                    text: createStep1Tooltip(
                        tour,
                        () => {
                            setQueryState({ query: 'lang:' })
                            tour.show('filter-lang')
                        },
                        () => {
                            setQueryState({ query: 'repo:' })
                            tour.show('filter-repository')
                        }
                    ),
                    attachTo: {
                        element: '.search-page__input-container',
                        on: 'bottom',
                    },
                },
                {
                    id: 'filter-lang',
                    text: generateStepTooltip({
                        tour,
                        dangerousTitleHtml: 'Type to filter the language autocomplete',
                        stepNumber: 2,
                        totalStepCount: 5,
                    }),
                    when: {
                        show() {
                            eventLogger.log('ViewedOnboardingTourFilterLangStep')
                        },
                    },
                    attachTo: {
                        element: '.search-page__input-container',
                        on: 'top',
                    },
                },
                {
                    id: 'filter-repository',
                    text: generateStepTooltip({
                        tour,
                        dangerousTitleHtml:
                            "Type the name of a repository you've used recently to filter the autocomplete list",
                        stepNumber: 2,
                        totalStepCount: 5,
                    }),
                    when: {
                        show() {
                            eventLogger.log('ViewedOnboardingTourFilterRepoStep')
                        },
                    },
                    attachTo: {
                        element: '.search-page__input-container',
                        on: 'top',
                    },
                },
                {
                    id: 'add-query-term',
                    text: generateStepTooltip({
                        tour,
                        dangerousTitleHtml: 'Add code to your search',
                        stepNumber: 3,
                        totalStepCount: 5,
                        description: 'Type the name of a function, variable or other code.',
                    }),
                    when: {
                        show() {
                            eventLogger.log('ViewedOnboardingTourAddQueryTermStep')
                        },
                    },
                    attachTo: {
                        element: '.search-page__input-container',
                        on: 'bottom',
                    },
                },
                {
                    id: 'submit-search',
                    text: generateStepTooltip({
                        tour,
                        dangerousTitleHtml: 'Use <kbd>return</kbd> or the search button to run your search',
                        stepNumber: 4,
                        totalStepCount: 5,
                    }),
                    when: {
                        show() {
                            eventLogger.log('ViewedOnboardingTourSubmitSearchStep')
                        },
                    },
                    attachTo: {
                        element: '.search-button',
                        on: 'top',
                    },
                    advanceOn: { selector: '.search-button__btn', event: 'click' },
                },
            ])
        } else {
            tour.addSteps([
                {
                    id: 'view-search-reference',
                    text: generateStepTooltip({
                        tour,
                        dangerousTitleHtml: 'Review the search reference',
                        stepNumber: 5,
                        totalStepCount: 5,
                    }),
                    attachTo: {
                        element: '.search-help-dropdown-button',
                        on: 'bottom',
                    },
                    when: {
                        show() {
                            eventLogger.log('ViewedOnboardingTourSearchReferenceStep')
                        },
                    },
                    advanceOn: { selector: '.search-help-dropdown-button', event: 'click' },
                },
            ])
        }
    }, [tour, inputLocation, setQueryState])
    return tour
}

type OnboardingTourQueryParameters = [{ key: 'onboardingTour'; value: 'true' }]

interface UseSearchOnboardingTourOptions {
    /**
     * Whether the onboarding tour feature flag is enabled.
     */
    showOnboardingTour: boolean
    /**
     * Where the onboarding tour is being displayed.
     */
    inputLocation: 'search-homepage' | 'global-navbar'

    /**
     * A callback allowing the onboarding tour to trigger
     * updates to the search query.
     */
    setQueryState: (queryState: QueryState) => void

    /**
     * The query currently displayed in the query input.
     */
    queryState: QueryState
    history: H.History
    location: H.Location
}

/**
 * Represents the object returned by `useSearchOnboardingTour`.
 *
 * The subset of MonacoQueryInput props should be passed down to the input component.
 */
interface UseSearchOnboardingTourReturnValue
    extends Pick<MonacoQueryInputProps, 'onCompletionItemSelected' | 'onSuggestionsInitialized' | 'onFocus'> {
    /**
     * Query parameters that should be added when submitting a search,
     * which will trigger the display of further onboarding tour steps on the search result page.
     */
    additionalQueryParameters: OnboardingTourQueryParameters | undefined
    /**
     * Whether the query input should be focused by default
     * (`false` on the search homepage when the tour is active).
     */
    shouldFocusQueryInput: boolean
}

/**
 * A hook that handles displaying and running the search onboarding tour,
 * to be used in conjunction with the MonacoQueryInput.
 *
 * See {@link UseSearchOnboardingTourOptions} and {@link UseSearchOnboardingTourReturnValue}
 */
export const useSearchOnboardingTour = ({
    showOnboardingTour,
    inputLocation,
    queryState,
    setQueryState,
    history,
    location,
}: UseSearchOnboardingTourOptions): UseSearchOnboardingTourReturnValue => {
    const tour = useTourWithSteps({ inputLocation, setQueryState })
    // True when the user has seen the tour on the search homepage
    const [hasSeenTour, setHasSeenTour] = useLocalStorage(HAS_SEEN_TOUR_KEY, false)
    // True when the user has manually cancelled the tour
    const [hasCancelledTour, setHasCancelledTour] = useLocalStorage(HAS_CANCELLED_TOUR_KEY, false)
    // True when the user has completed the tour on the search results page
    const [hasCompletedTour, setHasCompletedTour] = useLocalStorage(HAS_COMPLETED_TOUR_KEY, false)
    const shouldShowTour = useMemo(
        () =>
            showOnboardingTour &&
            daysActiveCount === 1 &&
            !hasCancelledTour &&
            !hasCompletedTour &&
            (inputLocation === 'global-navbar' || !hasSeenTour),
        [hasCancelledTour, hasCompletedTour, hasSeenTour, showOnboardingTour, inputLocation]
    )

    // Start the tour on the 'view-search-reference' step if the onboardingTour
    // query parameter is present in the URL. Clean the parameter from the URL.
    useEffect(() => {
        const queryParameters = new URLSearchParams(location.search)
        if (queryParameters.has('onboardingTour')) {
            if (shouldShowTour) {
                tour.start()
                tour.show('view-search-reference')
            }
            queryParameters.delete('onboardingTour')
            history.replace({
                search: queryParameters.toString(),
                hash: history.location.hash,
            })
        }
    }, [tour, shouldShowTour, location.search, history])

    // Start the Tour when the query input is focused on the search homepage.
    const onFocus = useCallback(() => {
        if (shouldShowTour && !tour.isActive() && inputLocation === 'search-homepage') {
            tour.start()
        }
    }, [shouldShowTour, tour, inputLocation])
    // If we're showing the onboarding tour, do not focus the query input by default on the search homepage.
    const shouldFocusQueryInput = useMemo(() => (shouldShowTour ? inputLocation !== 'search-homepage' : true), [
        shouldShowTour,
        inputLocation,
    ])

    // Hook into Tour cancellation and completion events.
    useEffect(() => {
        const onCancelled = (): void => {
            setHasCancelledTour(true)
            // If the user closed the tour, we don't want to show
            // any further popups, so set this to false.
            setAdditionalQueryParameters(undefined)
        }
        const onCompleted = (): void => {
            if (inputLocation === 'search-homepage') {
                // When the tour is 'completed' on the search homepage,
                // set HAS_SEEN to true, but not HAS_COMPLETED,
                // as we'll still want the user to see the last step on the
                // search results page.
                setHasSeenTour(true)
            } else {
                // On other pages, set the tour to completed.
                setHasCompletedTour(true)
            }
        }
        tour.on('cancel', onCancelled)
        tour.on('complete', onCompleted)
        return () => {
            tour.off('cancel', onCancelled)
            tour.off('complete', onCompleted)
        }
    }, [tour, setHasCompletedTour, setHasCancelledTour, setHasSeenTour, inputLocation])

    // 'Complete' tour on unmount.
    // This will not necessarily result in HAS_COMPLETED_CODE_MONITOR
    // being set to true (see completion event handler).
    useEffect(
        () => () => {
            if (tour.isActive()) {
                tour.complete()
            }
        },
        [tour]
    )
    // Upon mounting, set whether we should add the onboardingTour query parameter upon submitting searches
    // (used to display further steps of the tour on the search results page),
    // and log the ViewOnboardingTour event if the tour is being seen on the search homepage.
    const [additionalQueryParameters, setAdditionalQueryParameters] = useState<
        OnboardingTourQueryParameters | undefined
    >()
    useEffect(() => {
        if (shouldShowTour) {
            setAdditionalQueryParameters([{ key: 'onboardingTour', value: 'true' }])
            if (inputLocation === 'search-homepage') {
                eventLogger.log('ViewOnboardingTour')
            }
        }
    }, [tour, shouldShowTour, setAdditionalQueryParameters, inputLocation])

    // A handle allowing to trigger display of the MonacoQueryInput suggestions widget.
    const [suggestions, onSuggestionsInitialized] = useState<{ trigger: () => void }>()

    // On query or step changes, advance the Tour if appropriate.
    const currentStep = useCurrentStep(tour)
    const queryTokens = useMemo((): Token[] => {
        const scannedQuery = scanSearchQuery(queryState.query)
        return scannedQuery.type === 'success' ? scannedQuery.term : []
    }, [queryState.query])
    useEffect(() => {
        if (!tour.isActive()) {
            return
        }
        if (shouldTriggerSuggestions(currentStep, queryTokens)) {
            suggestions?.trigger()
        } else if (shouldAdvanceLangOrRepoStep(currentStep, queryTokens)) {
            tour.show('add-query-term')
        } else if (shouldShowSubmitSearch(currentStep, queryTokens)) {
            tour.show('submit-search')
        }
    }, [suggestions, queryTokens, tour, currentStep])

    // When a completion item is selected,
    // advance the repo or lang step if appropriate.
    const onCompletionItemSelected = useCallback(() => {
        if (shouldAdvanceLangOrRepoStep(currentStep, queryTokens)) {
            tour.show('add-query-term')
        }
    }, [queryTokens, tour, currentStep])

    return {
        onCompletionItemSelected,
        onFocus,
        onSuggestionsInitialized,
        additionalQueryParameters,
        shouldFocusQueryInput,
    }
}
