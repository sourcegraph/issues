import OpenInNewIcon from 'mdi-react/OpenInNewIcon'
import classNames from 'classnames'
import React, { useState, useCallback, useMemo } from 'react'
import { Link } from '@sourcegraph/shared/src/components/Link'
import { buildSearchURLQuery } from '@sourcegraph/shared/src/util/url'
import { useInputValidation, deriveInputClassName } from '@sourcegraph/shared/src/util/useInputValidation'
import { SearchPatternType } from '../../../graphql-operations'
import { scanSearchQuery } from '@sourcegraph/shared/src/search/query/scanner'
import { resolveFilter, validateFilter, FilterType } from '@sourcegraph/shared/src/search/query/filters'

interface TriggerAreaProps {
    query: string
    onQueryChange: (query: string) => void
    triggerCompleted: boolean
    setTriggerCompleted: (complete: boolean) => void
    startExpanded: boolean
}

const isDiffOrCommit = (value: string): boolean => value === 'diff' || value === 'commit'

export const FormTriggerArea: React.FunctionComponent<TriggerAreaProps> = ({
    query,
    onQueryChange,
    triggerCompleted,
    setTriggerCompleted,
    startExpanded,
}) => {
    const [showQueryForm, setShowQueryForm] = useState(startExpanded)
    const toggleQueryForm: React.FormEventHandler = useCallback(event => {
        event.preventDefault()
        setShowQueryForm(show => !show)
    }, [])

    const [isValidQuery, setIsValidQuery] = useState(false)
    const [hasTypeDiffOrCommitFilter, setHasTypeDiffOrCommitFilter] = useState(false)
    const [hasRepoFilter, setHasRepoFilter] = useState(false)
    const [hasPatternTypeFilter, setHasPatternTypeFilter] = useState(false)

    const [queryState, nextQueryFieldChange, queryInputReference, overrideState] = useInputValidation(
        useMemo(
            () => ({
                initialValue: query,
                synchronousValidators: [
                    (value: string) => {
                        const tokens = scanSearchQuery(value)

                        const isValidQuery = tokens.type === 'success'
                        setIsValidQuery(isValidQuery)

                        let hasTypeDiffOrCommitFilter = false
                        let hasRepoFilter = false
                        let hasPatternTypeFilter = false

                        if (tokens.type === 'success') {
                            const filters = tokens.term.filter(token => token.type === 'filter')
                            hasTypeDiffOrCommitFilter = filters.some(
                                filter =>
                                    filter.type === 'filter' &&
                                    resolveFilter(filter.field.value)?.type === FilterType.type &&
                                    filter.value &&
                                    isDiffOrCommit(filter.value.value)
                            )

                            hasRepoFilter = filters.some(
                                filter =>
                                    filter.type === 'filter' &&
                                    resolveFilter(filter.field.value)?.type === FilterType.repo &&
                                    filter.value
                            )

                            hasPatternTypeFilter = filters.some(
                                filter =>
                                    filter.type === 'filter' &&
                                    resolveFilter(filter.field.value)?.type === FilterType.patterntype &&
                                    filter.value &&
                                    validateFilter(filter.field.value, filter.value)
                            )
                        }

                        setHasTypeDiffOrCommitFilter(hasTypeDiffOrCommitFilter)
                        setHasRepoFilter(hasRepoFilter)
                        setHasPatternTypeFilter(hasPatternTypeFilter)

                        if (!isValidQuery) {
                            return 'Failed to parse query'
                        }

                        if (!hasTypeDiffOrCommitFilter) {
                            return 'Code monitors require queries to specify either `type:commit` or `type:diff`.'
                        }

                        if (!hasRepoFilter) {
                            return 'Code monitors require queries to specify a `repo:` filter.'
                        }

                        return undefined
                    },
                ],
            }),
            [query]
        )
    )

    const completeForm: React.FormEventHandler = useCallback(
        event => {
            event.preventDefault()
            setShowQueryForm(false)
            setTriggerCompleted(true)
            onQueryChange(`${queryState.value}${hasPatternTypeFilter ? '' : ' patternType:literal'}`)
        },
        [setTriggerCompleted, setShowQueryForm, onQueryChange, queryState, hasPatternTypeFilter]
    )

    const cancelForm: React.FormEventHandler = useCallback(
        event => {
            event.preventDefault()
            setShowQueryForm(false)
            overrideState({ value: query })
        },
        [setShowQueryForm, overrideState, query]
    )

    return (
        <>
            <h3>Trigger</h3>
            {showQueryForm && (
                <div className="code-monitor-form__card card p-3">
                    <div className="font-weight-bold">When there are new search results</div>
                    <span className="text-muted">
                        This trigger will fire when new search results are found for a given search query.
                    </span>
                    <span className="mt-4">Search query</span>
                    <div>
                        <div className="trigger-area__query-input mb-4">
                            <div className="d-flex flex-column flex-grow-1">
                                <input
                                    type="text"
                                    className={classNames(
                                        'trigger-area__query-input-field form-control my-2 test-trigger-input text-monospace',
                                        deriveInputClassName(queryState)
                                    )}
                                    onChange={nextQueryFieldChange}
                                    value={queryState.value}
                                    required={true}
                                    autoFocus={true}
                                    ref={queryInputReference}
                                    spellCheck={false}
                                    data-testid="trigger-query-edit"
                                />

                                <ul>
                                    <li>
                                        <input type="checkbox" disabled={true} checked={hasTypeDiffOrCommitFilter} />
                                        Contains a <code>type:diff</code> or <code>type:commit</code> filter
                                        <span
                                            data-tooltip="type:diff targets code present in new commits, while type:commit targets commit messages"
                                            data-placement="bottom"
                                        >
                                            ?
                                        </span>
                                    </li>
                                    <li>
                                        <input type="checkbox" disabled={true} checked={hasRepoFilter} />
                                        Contains a <code>repo:</code> filter
                                        <span
                                            data-tooltip="Code monitors can watch a maximum of 50 repos at a time. Target your query with repo: filters to narrow down your search."
                                            data-placement="bottom"
                                        >
                                            ?
                                        </span>
                                    </li>
                                    <li>
                                        <input type="checkbox" disabled={true} checked={isValidQuery} />
                                        Is a valid search query
                                    </li>
                                </ul>
                            </div>
                            <div className="trigger-area__query-input-preview-link p-2 my-2">
                                <Link
                                    to={`/search?${buildSearchURLQuery(
                                        queryState.value,
                                        SearchPatternType.literal,
                                        false
                                    )}`}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="trigger-area__query-input-preview-link-text test-preview-link"
                                >
                                    Preview results{' '}
                                    <OpenInNewIcon className="trigger-area__query-input-preview-link-icon ml-1 icon-inline" />
                                </Link>
                            </div>
                        </div>
                    </div>
                    <div>
                        <button
                            className="btn btn-secondary mr-1 test-submit-trigger"
                            onClick={completeForm}
                            type="submit"
                            disabled={queryState.kind !== 'VALID'}
                        >
                            Continue
                        </button>
                        <button type="button" className="btn btn-outline-secondary" onClick={cancelForm}>
                            Cancel
                        </button>
                    </div>
                </div>
            )}
            {!showQueryForm && (
                <button
                    type="button"
                    className="btn code-monitor-form__card--button card test-trigger-button"
                    aria-label="Edit trigger: When there are new search results"
                    onClick={toggleQueryForm}
                >
                    <div className="d-flex justify-content-between align-items-center w-100">
                        <div>
                            <div
                                className={classNames(
                                    'font-weight-bold',
                                    !triggerCompleted && 'code-monitor-form__card-link btn-link'
                                )}
                            >
                                When there are new search results
                            </div>
                            {triggerCompleted ? (
                                <code
                                    className="trigger-area__query-label text-break text-muted test-existing-query"
                                    data-testid="trigger-query-existing"
                                >
                                    {query}
                                </code>
                            ) : (
                                <span className="text-muted">
                                    This trigger will fire when new search results are found for a given search query.
                                </span>
                            )}
                        </div>
                        {triggerCompleted && <div className="btn-link">Edit</div>}
                    </div>
                </button>
            )}
            <small className="text-muted">
                {' '}
                What other events would you like to monitor?{' '}
                <a href="mailto:feedback@sourcegraph.com" target="_blank" rel="noopener">
                    Share feedback.
                </a>
            </small>
        </>
    )
}
