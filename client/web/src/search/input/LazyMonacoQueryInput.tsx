import React, { Suspense } from 'react'
import { MonacoQueryInputProps } from './MonacoQueryInput'
import { lazyComponent } from '../../util/lazyComponent'
import { Toggles } from './toggles/Toggles'

const MonacoQueryInput = lazyComponent(() => import('./MonacoQueryInput'), 'MonacoQueryInput')

/**
 * A plain query input displayed during lazy-loading of the MonacoQueryInput.
 * It has no suggestions, but still allows to type in and submit queries.
 */
export const PlainQueryInput: React.FunctionComponent<MonacoQueryInputProps> = ({
    queryState,
    autoFocus,
    onChange,
    ...props
}) => {
    const onInputChange = React.useCallback(
        (event: React.ChangeEvent<HTMLInputElement>) => {
            // cursorPosition is only used for legacy suggestions, it's OK to set it to 0 here.
            onChange({ query: event.target.value, cursorPosition: 0 })
        },
        [onChange]
    )
    return (
        <div className="query-input2 d-flex">
            <input
                type="text"
                autoFocus={autoFocus}
                className="form-control code lazy-monaco-query-input--intermediate-input"
                value={queryState.query}
                onChange={onInputChange}
                spellCheck={false}
            />
            <div className="query-input2__toggle-container">
                <Toggles {...props} navbarSearchQuery={queryState.query} />
            </div>
        </div>
    )
}

/**
 * A lazily-loaded {@link MonacoQueryInput}, displaying a read-only query field as a fallback during loading.
 */
export const LazyMonacoQueryInput: React.FunctionComponent<MonacoQueryInputProps> = props => (
    <Suspense fallback={<PlainQueryInput {...props} />}>
        <MonacoQueryInput {...props} />
    </Suspense>
)
