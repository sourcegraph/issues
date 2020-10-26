import * as React from 'react'
import { InfoDropdown } from '../input/InfoDropdown'
import { QueryBuilderState } from './QueryBuilder'

/** An example demonstrating the capabilities of the search field. */
export interface QueryFieldExample {
    /** A markdown string describing the example. */
    description: string
    /** The value for the example. Will be displayed as an inline code block. */
    value: string
}
interface Props {
    /** The field title */
    title: string
    /** An example displaying the shortcut for this field. */
    placeholder: string
    /** An optional example for sourcegraph.com that displays the shortcut for this field. */
    dotComPlaceholder?: string
    /** An description of the input field that is displayed below the field. */
    description: string
    /**
     * An appropriate identifier for this field, to be used as a suffix for CSS classes and testing IDs.
     * Must be a single or hyphenated word, and unique amongst the other fields in the query builder.
     */
    shortName: keyof QueryBuilderState['fields']
    /** A list of useful examples demonstrating valid values that can be inputted for this search field. */
    examples: QueryFieldExample[]
    /** Handler for when an input field changes. */
    onInputChange: Record<
        keyof QueryBuilderState['fields'],
        React.ChangeEventHandler<HTMLInputElement | HTMLSelectElement>
    >

    isSourcegraphDotCom: boolean
}

export const QueryBuilderInputRow: React.FunctionComponent<Props> = props => {
    const placeholder = props.isSourcegraphDotCom ? props.dotComPlaceholder || props.placeholder : props.placeholder
    return (
        <div className="query-builder-input-row">
            <label className="query-builder-input-row__label" htmlFor={`query-builder__${props.shortName}`}>
                {props.title}:
            </label>
            <div className="query-builder-input-row__input">
                <input
                    data-testid={`test-${props.shortName}`}
                    id={`query-builder-${props.shortName}`}
                    className="form-control query-builder__input"
                    spellCheck={false}
                    autoCapitalize="off"
                    autoComplete="off"
                    placeholder={placeholder}
                    onChange={props.onInputChange[props.shortName]}
                />
            </div>
            <InfoDropdown title={props.title} markdown={props.description} examples={props.examples} />
        </div>
    )
}
