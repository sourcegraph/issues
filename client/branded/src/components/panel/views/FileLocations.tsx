import { Location } from '@sourcegraph/extension-api-types'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { Badged } from 'sourcegraph'
import * as H from 'history'
import { upperFirst } from 'lodash'
import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import * as React from 'react'
import { Observable, Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, map, startWith, switchMap } from 'rxjs/operators'
import { FetchFileParameters } from '../../../../../shared/src/components/CodeExcerpt'
import { FileMatch, IFileMatch, ILineMatch } from '../../../../../shared/src/components/FileMatch'
import { VirtualList } from '../../../../../shared/src/components/VirtualList'
import { SettingsCascadeProps } from '../../../../../shared/src/settings/settings'
import { asError, ErrorLike, isErrorLike } from '../../../../../shared/src/util/errors'
import { property, isDefined } from '../../../../../shared/src/util/types'
import { parseRepoURI, toPrettyBlobURL, toRepoURL } from '../../../../../shared/src/util/url'
import { VersionContextProps } from '../../../../../shared/src/search/util'

export const FileLocationsError: React.FunctionComponent<{ error: ErrorLike }> = ({ error }) => (
    <div className="file-locations__error alert alert-danger m-2">
        <AlertCircleIcon className="icon-inline" /> Error getting locations: {upperFirst(error.message)}
    </div>
)

export const FileLocationsNotFound: React.FunctionComponent = () => (
    <div className="file-locations__not-found m-2">
        <MapSearchIcon className="icon-inline" /> No locations found
    </div>
)

interface Props extends SettingsCascadeProps, VersionContextProps {
    location: H.Location
    /**
     * The observable that emits the locations.
     */
    locations: Observable<Location[] | null>

    /** The icon to use for each location. */
    icon: React.ComponentType<{ className?: string }>

    /** Called when a location is selected. */
    onSelect?: () => void

    className?: string

    isLightTheme: boolean

    fetchHighlightedFileLines: (parameters: FetchFileParameters, force?: boolean) => Observable<string[]>
}

const LOADING = 'loading' as const

interface State {
    /**
     * Locations (inside files identified by LSP-style git:// URIs) to display, loading, or an error if they failed
     * to load.
     */
    locationsOrError: typeof LOADING | Location[] | null | ErrorLike

    itemsToShow: number
}

/**
 * Displays a flat list of file excerpts. For a tree view, use FileLocationsTree.
 */
export class FileLocations extends React.PureComponent<Props, State> {
    public state: State = {
        locationsOrError: LOADING,
        itemsToShow: 3,
    }

    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        const locationsChanges = this.componentUpdates.pipe(
            map(({ locations }) => locations),
            distinctUntilChanged()
        )

        this.subscriptions.add(
            locationsChanges
                .pipe(
                    switchMap(query => query.pipe(catchError(error => [asError(error) as ErrorLike]))),
                    startWith(LOADING),
                    map(result => ({ locationsOrError: result }))
                )
                .subscribe(
                    stateUpdate => this.setState(stateUpdate),
                    error => console.error(error)
                )
        )

        this.componentUpdates.next(this.props)
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        if (isErrorLike(this.state.locationsOrError)) {
            return <FileLocationsError error={this.state.locationsOrError} />
        }
        if (this.state.locationsOrError === LOADING) {
            return <LoadingSpinner className="icon-inline m-1" />
        }
        if (this.state.locationsOrError === null || this.state.locationsOrError.length === 0) {
            return <FileLocationsNotFound />
        }

        // Locations by fully qualified URI, like git://github.com/gorilla/mux?revision#mux.go
        const locationsByURI = new Map<string, Location[]>()

        // URIs with >0 locations, in order (to avoid jitter as more results stream in).
        const orderedURIs: { uri: string; repo: string }[] = []

        if (this.state.locationsOrError) {
            for (const location of this.state.locationsOrError) {
                if (!locationsByURI.has(location.uri)) {
                    locationsByURI.set(location.uri, [])

                    const { repoName } = parseRepoURI(location.uri)
                    orderedURIs.push({ uri: location.uri, repo: repoName })
                }
                locationsByURI.get(location.uri)!.push(location)
            }
        }

        return (
            <div className={`file-locations ${this.props.className || ''}`}>
                <VirtualList
                    itemsToShow={this.state.itemsToShow}
                    onShowMoreItems={this.onShowMoreItems}
                    items={orderedURIs.map(({ uri, repo }, index) => (
                        <FileMatch
                            key={index}
                            location={this.props.location}
                            expanded={true}
                            result={referencesToFileMatch(uri, locationsByURI.get(uri)!)}
                            icon={this.props.icon}
                            onSelect={this.onSelect}
                            showAllMatches={true}
                            isLightTheme={this.props.isLightTheme}
                            fetchHighlightedFileLines={this.props.fetchHighlightedFileLines}
                            settingsCascade={this.props.settingsCascade}
                        />
                    ))}
                />
            </div>
        )
    }

    private onShowMoreItems = (): void => {
        this.setState(state => ({ itemsToShow: state.itemsToShow + 3 }))
    }

    private onSelect = (): void => {
        if (this.props.onSelect) {
            this.props.onSelect()
        }
    }
}

function referencesToFileMatch(uri: string, references: Badged<Location>[]): IFileMatch {
    const parsedUri = parseRepoURI(uri)
    return {
        file: {
            path: parsedUri.filePath || '',
            url: toPrettyBlobURL({
                repoName: parsedUri.repoName,
                filePath: parsedUri.filePath!,
                revision: parsedUri.commitID || '',
            }),
            commit: {
                oid: (parsedUri.commitID || parsedUri.revision)!,
            },
        },
        repository: {
            name: parsedUri.repoName,
            // This is the only usage of toRepoURL, and it is arguably simpler than getting the value from the
            // GraphQL API. We will be removing these old-style git: URIs eventually, so it's not worth fixing this
            // deprecated usage.
            url: toRepoURL(parsedUri),
        },
        limitHit: false,
        lineMatches: references.filter(property('range', isDefined)).map(
            (reference): ILineMatch => ({
                preview: '',
                limitHit: false,
                lineNumber: reference.range.start.line,
                offsetAndLengths: [
                    [reference.range.start.character, reference.range.end.character - reference.range.start.character],
                ],
                badge: reference.badge,
            })
        ),
    }
}
