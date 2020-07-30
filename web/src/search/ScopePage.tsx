import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React, { useEffect, useState, useMemo, useCallback } from 'react'
import { RouteComponentProps } from 'react-router'
import { Link } from 'react-router-dom'
import { of } from 'rxjs'
import { catchError } from 'rxjs/operators'
import { RepoLink } from '../../../shared/src/components/RepoLink'
import * as GQL from '../../../shared/src/graphql/schema'
import { isSettingsValid, SettingsCascadeProps } from '../../../shared/src/settings/settings'
import { renderMarkdown } from '../../../shared/src/util/markdown'
import { Form } from '../components/Form'
import { HeroPage } from '../components/HeroPage'
import { PageTitle } from '../components/PageTitle'
import { Settings } from '../schema/settings.schema'
import { eventLogger } from '../tracking/eventLogger'
import { fetchReposByQuery } from './backend'
import { submitSearch, QueryState } from './helpers'
import { QueryInput } from './input/QueryInput'
import { SearchButton } from './input/SearchButton'
import { PatternTypeProps, CaseSensitivityProps, CopyQueryButtonProps } from '.'
import { ErrorAlert } from '../components/alerts'
import { asError, isErrorLike } from '../../../shared/src/util/errors'
import { useObservable } from '../../../shared/src/util/useObservable'
import { Markdown } from '../../../shared/src/components/Markdown'
import { pluralize } from '../../../shared/src/util/strings'
import * as H from 'history'
import { VersionContextProps } from '../../../shared/src/search/util'
import { OptionalAuthProps } from '../auth'

const ScopeNotFound: React.FunctionComponent = () => (
    <HeroPage
        icon={MapSearchIcon}
        title="404: Not Found"
        subtitle={
            <>
                No search page found with this scope ID. Add an ID and description to a search scope to create a search
                page. See{' '}
                <Link to="/help/user/search/scopes#creating-search-scope-pages">search scope documentation</Link>.
            </>
        }
    />
)

interface Props
    extends RouteComponentProps<{ id: GQL.Scalars['ID'] }>,
        SettingsCascadeProps,
        PatternTypeProps,
        CaseSensitivityProps,
        CopyQueryButtonProps,
        VersionContextProps,
        OptionalAuthProps {
    onNavbarQueryChange: (queryState: QueryState) => void
    history: H.History

    /** Whether globbing is enabled for filters. */
    globbing: boolean
}

/**
 * A page with a search bar and list of repositories for a single search scope.
 */
export const ScopePage: React.FunctionComponent<Props> = ({ settingsCascade, onNavbarQueryChange, ...props }) => {
    useEffect(() => eventLogger.logViewEvent('Scope'), [])

    const [queryState, setQueryState] = useState<QueryState>({ query: '', cursorPosition: 0 })

    const searchScopes = useMemo(
        () => isSettingsValid<Settings>(settingsCascade) && settingsCascade.final['search.scopes'],
        [settingsCascade]
    )
    const searchScope = useMemo(
        () => (searchScopes ? searchScopes.find(scope => scope.id === props.match.params.id) : undefined),
        [props.match.params.id, searchScopes]
    )
    useEffect(() => {
        if (searchScope) {
            onNavbarQueryChange({
                query: searchScope.value,
                cursorPosition: searchScope.value.length,
            })
        }
    }, [onNavbarQueryChange, searchScope])

    const scopeRepositories = useObservable(
        useMemo(() => {
            if (searchScope?.value.includes('repo:') || searchScope?.value.includes('repogroup:')) {
                return fetchReposByQuery(searchScope.value).pipe(catchError(error => of(asError(error))))
            }
            return of([])
            // False positive: https://github.com/facebook/react/issues/19064
            // eslint-disable-next-line react-hooks/exhaustive-deps
        }, [searchScope])
    )

    const onSubmit = useCallback(
        (event: React.FormEvent<HTMLFormElement>): void => {
            event.preventDefault()
            submitSearch({
                ...props,
                query: `${searchScope ? searchScope.value : ''} ${queryState.query}`,
                source: 'scopePage',
            })
        },
        [props, queryState.query, searchScope]
    )

    const [repositoriesFirst, setRepositoriesFirst] = useState(50)
    const showMoreRepositories = useCallback(() => setRepositoriesFirst(previousValue => previousValue + 50), [])

    if (!searchScopes) {
        return null
    }
    if (!searchScope) {
        return <ScopeNotFound />
    }
    return (
        <div className="container mt-3">
            <PageTitle title={searchScope.name} />
            <header>
                <h1>{searchScope.name}</h1>
                {searchScope.description && (
                    <Markdown dangerousInnerHTML={renderMarkdown(searchScope.description)} history={props.history} />
                )}
            </header>
            <section className="mb-5">
                <Form className="d-flex" onSubmit={onSubmit}>
                    <QueryInput
                        {...props}
                        value={queryState}
                        onChange={setQueryState}
                        prependQueryForSuggestions={searchScope.value}
                        autoFocus={true}
                        location={props.location}
                        history={props.history}
                        settingsCascade={settingsCascade}
                        placeholder="Search..."
                    />
                    <SearchButton />
                </Form>
                <div className="d-flex align-items-center m-1 text-muted">
                    <span className="mr-1">Scope:</span>
                    <code className="border rounded p-1">{searchScope.value}</code>
                </div>
            </section>

            {isErrorLike(scopeRepositories) ? (
                <ErrorAlert error={scopeRepositories} history={props.history} />
            ) : (
                scopeRepositories &&
                (scopeRepositories.length > 0 ? (
                    <section className="card d-inline-flex">
                        <h3 className="card-header">Repositories</h3>
                        <div className="list-group list-group-flush">
                            {scopeRepositories.slice(0, repositoriesFirst).map(repo => (
                                <RepoLink
                                    key={repo.name}
                                    repoName={repo.name}
                                    to={repo.url}
                                    className="list-group-item list-group-item-action"
                                />
                            ))}
                        </div>
                        <div className="card-footer">
                            <span>
                                {scopeRepositories.length}{' '}
                                {pluralize('repository', scopeRepositories.length, 'repositories')} in scope{' '}
                                {scopeRepositories.length > repositoriesFirst
                                    ? `(showing first ${repositoriesFirst})`
                                    : ''}{' '}
                            </span>
                            {repositoriesFirst < scopeRepositories.length && (
                                <button
                                    type="button"
                                    className="btn btn-secondary btn-sm p-1 ml-2"
                                    onClick={showMoreRepositories}
                                >
                                    Show more
                                </button>
                            )}
                        </div>
                    </section>
                ) : (
                    <div className="alert alert-warning">No repositories in scope.</div>
                ))
            )}
        </div>
    )
}
