import ExternalLinkIcon from 'mdi-react/ExternalLinkIcon'
import * as React from 'react'
import { InfoDropdown } from '../input/InfoDropdown'
import { QueryBuilderInputRow } from './QueryBuilderInputRow'
import { PatternTypeProps } from '..'
import { SearchPatternType } from '../../graphql-operations'

export interface QueryBuilderProps extends Pick<PatternTypeProps, 'patternType'> {
    /**
     * Called when there is a change to the query synthesized from this
     * component's fields.
     */
    onFieldsQueryChange: (query: string) => void
    isSourcegraphDotCom: boolean
}

interface QueryFields {
    type: string
    repo: string
    file: string
    language: string
    patterns: string
    exactMatch: string
    case: string
    author: string
    after: string
    before: string
    message: string
    count: string
    timeout: string
}

export interface QueryBuilderState {
    /**
     * The query constructed from the values in the input fields.
     */
    builderQuery: string
    typeOfSearch: 'code' | 'diff' | 'commit' | 'symbol'
    fields: QueryFields
}

/**
 * The individual input fields for the various elements of the search query syntax.
 */
export class QueryBuilder extends React.Component<QueryBuilderProps, QueryBuilderState> {
    constructor(props: QueryBuilderProps) {
        super(props)
        this.state = {
            builderQuery: '',
            typeOfSearch: 'code',
            fields: {
                type: '',
                repo: '',
                file: '',
                language: '',
                patterns: '',
                exactMatch: '',
                case: '',
                author: '',
                after: '',
                before: '',
                message: '',
                count: '',
                timeout: '',
            },
        }
    }

    private onInputChange = (key: keyof QueryBuilderState['fields']) => (
        event: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>
    ): void => {
        const { value } = event.target
        this.setState(
            ({ fields }) => {
                const newFields = { ...fields, [key]: value }

                const fieldsQueryParts: string[] = []
                for (const [inputField, inputValue] of Object.entries(newFields)) {
                    if (inputValue !== '') {
                        if (inputField === 'patterns') {
                            // Patterns should be added to the query as-is.
                            fieldsQueryParts.push(inputValue)
                        } else if (inputField === 'exactMatch') {
                            // Exact matches don't have a literal field operator (e.g. exactMatch:) in the query.
                            fieldsQueryParts.push(formatExactMatchField(inputValue, this.props.patternType))
                        } else if (inputField === 'type' && inputValue === 'code') {
                            // code searches don't need to be specified.
                            continue
                        } else {
                            fieldsQueryParts.push(formatFieldForQuery(inputField, inputValue))
                        }
                    }
                }
                const builderQuery = fieldsQueryParts.join(' ')

                return { fields: newFields, builderQuery }
            },
            () => {
                this.props.onFieldsQueryChange(this.state.builderQuery)
            }
        )
    }

    private onTypeChange = (event: React.ChangeEvent<HTMLSelectElement>): void => {
        this.onInputChange('type')(event)

        const searchType = event.target.value
        if (searchType === 'commit' || searchType === 'diff' || searchType === 'symbol') {
            this.setState({ typeOfSearch: searchType })
        } else {
            this.setState({ typeOfSearch: 'code' })
        }
    }

    private fieldsChanged = {
        type: this.onTypeChange,
        repo: this.onInputChange('repo'),
        file: this.onInputChange('file'),
        language: this.onInputChange('language'),
        patterns: this.onInputChange('patterns'),
        exactMatch: this.onInputChange('exactMatch'),
        case: this.onInputChange('case'),
        author: this.onInputChange('author'),
        after: this.onInputChange('after'),
        before: this.onInputChange('before'),
        message: this.onInputChange('message'),
        count: this.onInputChange('count'),
        timeout: this.onInputChange('timeout'),
    }

    public render(): JSX.Element | null {
        const documentationUrlPrefix = this.props.isSourcegraphDotCom ? 'https://docs.sourcegraph.com' : '/help'
        return (
            <div className="query-builder">
                <div className="query-builder__header">
                    <h3 className="query-builder__header-input">Match:</h3>
                </div>
                <div className="query-builder__section query-builder__section--blue">
                    <div className="query-builder__row">
                        <label className="query-builder__row-label" htmlFor="query-builder__type">
                            Type:
                        </label>
                        <div className="query-builder__row-input">
                            <select
                                id="query-builder__type"
                                className="form-control query-builder__input"
                                onChange={this.onTypeChange}
                                value={this.state.typeOfSearch}
                            >
                                <option value="code" defaultChecked={true}>
                                    Code (default)
                                </option>
                                <option value="diff">Commit diffs</option>
                                <option value="commit">Commit messages</option>
                                <option value="symbol">Symbols</option>
                            </select>
                        </div>
                        <InfoDropdown
                            title="Type"
                            markdown="Search code (file contents), diffs (added/changed/removed lines in commits), commit messages, or symbols."
                        />
                    </div>
                    {(this.state.typeOfSearch === 'commit' || this.state.typeOfSearch === 'diff') && (
                        <>
                            <QueryBuilderInputRow
                                onInputChange={this.fieldsChanged}
                                placeholder="alice"
                                title="Author"
                                description='Only include commits whose author matches. Your query is matched against a string of the form "Author Name <name@example.com>".'
                                isSourcegraphDotCom={this.props.isSourcegraphDotCom}
                                shortName="author"
                                examples={[
                                    { description: 'Search for commits authored by alice', value: 'alice' },
                                    {
                                        description: 'Search for commits by author email domain',
                                        value: '@example.com',
                                    },
                                ]}
                            />
                            <QueryBuilderInputRow
                                onInputChange={this.fieldsChanged}
                                placeholder="1 year ago"
                                title="Before"
                                description="Only include commits made before a specified date."
                                isSourcegraphDotCom={this.props.isSourcegraphDotCom}
                                shortName="before"
                                examples={[
                                    {
                                        description: 'Search for commits older than 3 months',
                                        value: '3 months ago',
                                    },
                                    {
                                        description: 'Search for commits before a specific date',
                                        value: '2019 Feb 20',
                                    },
                                ]}
                            />
                            <QueryBuilderInputRow
                                onInputChange={this.fieldsChanged}
                                placeholder="6 months ago"
                                title="After"
                                description="Only include commits made after a specified date."
                                isSourcegraphDotCom={this.props.isSourcegraphDotCom}
                                shortName="after"
                                examples={[
                                    {
                                        description: 'Search for commits less than 5 days old',
                                        value: '5 days ago',
                                    },
                                    {
                                        description: 'Search for commits after a specific date',
                                        value: '2019 Feb 20',
                                    },
                                ]}
                            />
                            <QueryBuilderInputRow
                                onInputChange={this.fieldsChanged}
                                placeholder="fix: typo"
                                title="Message"
                                description="Only include commits whose commit message matches."
                                isSourcegraphDotCom={this.props.isSourcegraphDotCom}
                                shortName="message"
                                examples={[
                                    {
                                        description: 'Search for commits whose message includes "fix: typo"',
                                        value: 'fix: typo',
                                    },
                                    {
                                        description: 'Search for commits whose message includes "middleware"',
                                        value: 'middleware',
                                    },
                                ]}
                            />
                        </>
                    )}
                    <hr className="my-3" />
                    <QueryBuilderInputRow
                        onInputChange={this.fieldsChanged}
                        placeholder="(read|write)File"
                        title="Patterns"
                        isSourcegraphDotCom={this.props.isSourcegraphDotCom}
                        shortName="patterns"
                        description="Match lines against this regexp. Supports full regular expressions (using the standard [RE2 syntax](https://github.com/google/re2/wiki/Syntax)). A space matches anything until the next query term; use `\s` to match only whitespace."
                        examples={[
                            {
                                description: 'Search for `readFile` or `writeFile`',
                                value: '(read|write)File',
                            },
                            {
                                description: 'Search for lines that start with `package` and end with `test`',
                                value: '^package test$',
                            },
                            {
                                description:
                                    'Search for the standalone word `set` (using the regexp special character \\b for word boundary)',
                                value: '\\bset\\b',
                            },
                        ]}
                    />
                    <QueryBuilderInputRow
                        onInputChange={this.fieldsChanged}
                        placeholder="open("
                        title="Exact string"
                        isSourcegraphDotCom={this.props.isSourcegraphDotCom}
                        shortName="exactMatch"
                        description="Match lines containing this exact string. Punctuation and special characters will be matched literally."
                        examples={[{ description: 'Search for `open(`', value: 'open(' }]}
                    />
                    <div className="query-builder__row">
                        <label className="query-builder__row-label" htmlFor="query-builder-case">
                            Case sensitive:
                        </label>
                        <div className="query-builder__row-input">
                            <select
                                id="query-builder-case"
                                className="form-control query-builder__input"
                                onChange={this.onCaseChange}
                            >
                                <option value="no" defaultChecked={true}>
                                    No
                                </option>
                                <option value="yes">Yes</option>
                            </select>
                        </div>
                        <InfoDropdown
                            title="Case sensitive"
                            markdown="Perform a case sensitive query. Matches are case insensitive by default."
                        />
                    </div>
                </div>
                <div className="query-builder__header">
                    <h3 className="query-builder__header-input">Search scope:</h3>
                </div>
                <div className="query-builder__section query-builder__section--purple">
                    <QueryBuilderInputRow
                        onInputChange={this.fieldsChanged}
                        placeholder="myorg/myrepo"
                        dotComPlaceholder="github.com/myorg/"
                        title="Repositories"
                        isSourcegraphDotCom={this.props.isSourcegraphDotCom}
                        shortName="repo"
                        description={`Specify the repositories to search in. ${
                            this.props.isSourcegraphDotCom ? '' : 'By default, all repositories are searched.'
                        } Supports regexp. To exclude repositories, use the \`-repo:\` keyword in the main search bar.\n\nAdd \`@mybranch\` to the end to search a non-default branch (or any other Git revspec).`}
                        examples={[
                            {
                                description: 'Search in repositories named `gorilla/mux` or `gorilla/pat`',
                                value: 'gorilla/(mux|pat)$',
                            },
                            {
                                description: 'Search in all repositories in a GitHub organization',
                                value: 'github.com/kubernetes/',
                            },
                            {
                                description: 'Search in a GitHub repository on a specific branch (other than master)',
                                value: 'github.com/kubernetes/kubernetes@release-0.4',
                            },
                        ]}
                    />
                    <QueryBuilderInputRow
                        onInputChange={this.fieldsChanged}
                        placeholder="docs/"
                        title="File paths"
                        isSourcegraphDotCom={this.props.isSourcegraphDotCom}
                        shortName="file"
                        description="Only include results from matching file paths. Supports regexp. To exclude files, use the `-file:` keyword in the main search bar."
                        examples={[
                            {
                                description: 'Search in files whose full path contains `internal`',
                                value: 'internal',
                            },
                            {
                                description: 'Search in the top-level directory `docs`',
                                value: '^docs/',
                            },
                        ]}
                    />
                    <QueryBuilderInputRow
                        onInputChange={this.fieldsChanged}
                        placeholder="typescript"
                        title="Language"
                        isSourcegraphDotCom={this.props.isSourcegraphDotCom}
                        shortName="language"
                        description="Only include results from files in the specified programming language. To exclude languages, use the \`-lang:\` keyword in the main search bar."
                        examples={[
                            {
                                description: 'Search in JavaScript files',
                                value: 'javascript',
                            },
                            {
                                description: 'Search in Go files',
                                value: 'go',
                            },
                            {
                                description: 'Search in Markdown documents',
                                value: 'markdown',
                            },
                        ]}
                    />
                </div>
                <div className="query-builder__docs-link">
                    <a target="blank" href={`${documentationUrlPrefix}/code_search/reference/queries`}>
                        View all search options in docs <ExternalLinkIcon className="icon-inline small" />
                    </a>
                </div>
            </div>
        )
    }

    private onCaseChange = (event: React.ChangeEvent<HTMLSelectElement>): void => {
        this.onInputChange('case')(event)
    }
}

/**
 *
 * @param alwaysQuote if true, the value will always be wrapped in quotes.
 */
function formatFieldForQuery(field: string, value: string, alwaysQuote?: boolean): string {
    // The user shouldn't include the 'repo:' (or other field name) in the value, but
    // if they do, then be helpful and remove it for them to avoid double fields like
    // 'repo:repo:foo'.
    if (field) {
        value = value.replace(new RegExp(`^${field}:`, 'g'), '')
    }

    // See if we need to double-quote value.
    const jsonValue = JSON.stringify(value)
    if (value.includes(' ') || jsonValue.slice(1, -1) !== value || alwaysQuote) {
        value = jsonValue
    }

    return field ? `${field}:${value}` : value
}

/**
 * Formats the value passed to the exact match field, wrapping it with quotes depending on whether
 * it's in regexp or literal mode.
 *
 * @param value the value passed to the exactMatch
 * @param patternType the current patternType
 */
function formatExactMatchField(value: string, patternType: SearchPatternType): string {
    if (patternType === SearchPatternType.literal) {
        return value
    }

    return JSON.stringify(value)
}
