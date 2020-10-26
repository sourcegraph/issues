import * as H from 'history'
import { isEqual } from 'lodash'
import * as React from 'react'
import { from, Subject, Subscription, Observable, of } from 'rxjs'
import { distinctUntilChanged, map, startWith, switchMap, tap } from 'rxjs/operators'
import { Entry } from '../../../../../shared/src/api/client/services/registry'
import {
    ProvidePanelViewSignature,
    PanelViewProviderRegistrationOptions,
} from '../../../../../shared/src/api/client/services/panelViews'
import { ContributableViewContainer, TextDocumentPositionParameters } from '../../../../../shared/src/api/protocol'
import { ActivationProps } from '../../../../../shared/src/components/activation/Activation'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { PlatformContextProps } from '../../../../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../../../../shared/src/settings/settings'
import { AbsoluteRepoFile, ModeSpec, parseHash, UIPositionSpec } from '../../../../../shared/src/util/url'
import { RepoHeaderContributionsLifecycleProps } from '../../RepoHeader'
import { RepoRevisionSidebarCommits } from '../../RepoRevisionSidebarCommits'
import { ThemeProps } from '../../../../../shared/src/theme'
<<<<<<< HEAD:web/src/repo/blob/panel/BlobPanel.tsx
import { wrapRemoteObservable, finallyReleaseProxy } from '../../../../../shared/src/api/client/api/common'
import { Controller } from '../../../../../shared/src/extensions/controller'
import { MaybeLoadingResult } from '@sourcegraph/codeintellify'
import * as clientType from '@sourcegraph/extension-api-types'
=======
import { AuthenticatedUser } from '../../../auth'
>>>>>>> main:client/web/src/repo/blob/panel/BlobPanel.tsx

interface Props
    extends AbsoluteRepoFile,
        Partial<UIPositionSpec>,
        ModeSpec,
        RepoHeaderContributionsLifecycleProps,
        SettingsCascadeProps,
        PlatformContextProps,
        ThemeProps,
        ActivationProps {
    location: H.Location
    history: H.History
    repoID: GQL.ID
    repoName: string
    commitID: string
<<<<<<< HEAD:web/src/repo/blob/panel/BlobPanel.tsx
    authenticatedUser: GQL.IUser | null
    extensionsController: Controller
=======
    authenticatedUser: AuthenticatedUser | null
>>>>>>> main:client/web/src/repo/blob/panel/BlobPanel.tsx
}

export type BlobPanelTabID = 'info' | 'def' | 'references' | 'impl' | 'typedef' | 'history'

/** The subject (what the contextual information refers to). */
interface PanelSubject extends AbsoluteRepoFile, ModeSpec, Partial<UIPositionSpec> {
    repoID: string

    /**
     * Include the full URI fragment here because it represents the state of panels, and we want
     * panels to be re-rendered when this state changes.
     */
    hash: string
}

function toSubject(props: Props): PanelSubject {
    const parsedHash = parseHash(props.location.hash)
    return {
        repoName: props.repoName,
        repoID: props.repoID,
        commitID: props.commitID,
        revision: props.revision,
        filePath: props.filePath,
        mode: props.mode,
        position:
            parsedHash.line !== undefined ? { line: parsedHash.line, character: parsedHash.character || 0 } : undefined,
        hash: props.location.hash,
    }
}

/**
 * A null-rendered component that registers panel views for the blob.
 */
export class BlobPanel extends React.PureComponent<Props> {
    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()

    constructor(props: Props) {
        super(props)

        const componentUpdates = this.componentUpdates.pipe(startWith(this.props))

        // Changes to the subject, including upon the initial mount.
        const subjectChanges = componentUpdates.pipe(
            map(toSubject),
            distinctUntilChanged((a, b) => isEqual(a, b))
        )

        const entryForViewProviderRegistration = <P extends TextDocumentPositionParameters>(
            id: string,
            title: string,
            priority: number,
<<<<<<< HEAD:web/src/repo/blob/panel/BlobPanel.tsx
            provideLocations: (params: P) => Observable<MaybeLoadingResult<clientType.Location[]>>,
            extraParameters?: Pick<P, Exclude<keyof P, keyof TextDocumentPositionParams>>
=======
            registry: TextDocumentLocationProviderRegistry<P>,
            extraParameters?: Pick<P, Exclude<keyof P, keyof TextDocumentPositionParameters>>
>>>>>>> main:client/web/src/repo/blob/panel/BlobPanel.tsx
        ): Entry<PanelViewProviderRegistrationOptions, ProvidePanelViewSignature> => ({
            registrationOptions: { id, container: ContributableViewContainer.Panel },
            provider: from(this.props.extensionsController.extensionHostAPI).pipe(
                // Get TextDocumentPositionParams from selection of active viewer
                switchMap(extensionHostAPI =>
                    wrapRemoteObservable(extensionHostAPI.getActiveCodeEditorPosition(), this.subscriptions).pipe(
                        finallyReleaseProxy()
                    )
                ),
<<<<<<< HEAD:web/src/repo/blob/panel/BlobPanel.tsx
                map(textDocumentPositionParameters => {
                    if (!textDocumentPositionParameters) {
                        return null
                    }
                    return {
                        title,
                        content: '',
                        priority,
=======
                switchMap(activeEditor =>
                    registry.hasProvidersForActiveTextDocument(activeEditor).pipe(
                        map(hasProviders => {
                            if (!hasProviders) {
                                return null
                            }
                            const parameters: TextDocumentPositionParameters | null = getActiveCodeEditorPosition(
                                activeEditor
                            )
                            if (!parameters) {
                                return null
                            }
                            return {
                                title,
                                content: '',
                                priority,
>>>>>>> main:client/web/src/repo/blob/panel/BlobPanel.tsx

                        // This disable directive is necessary because TypeScript is not yet smart
                        // enough to know that (typeof params & typeof extraParams) is P.
                        //
                        // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
                        locationProvider: provideLocations({
                            ...textDocumentPositionParameters,
                            ...extraParameters,
                        } as P).pipe(
                            tap(({ result: locations }) => {
                                if (this.props.activation && id === 'references' && locations.length > 0) {
                                    this.props.activation.update({ FoundReferences: true })
                                }
                            })
                        ),
                    }
                })
            ),
        })

        this.subscriptions.add(
            this.props.extensionsController.services.panelViews.registerProviders([
                entryForViewProviderRegistration('def', 'Definition', 190, parameters =>
                    from(this.props.extensionsController.extensionHostAPI).pipe(
                        switchMap(extensionHostAPI => wrapRemoteObservable(extensionHostAPI.getDefinitions(parameters)))
                    )
                ),
                entryForViewProviderRegistration(
                    'references',
                    'References',
                    180,
                    parameters =>
                        from(this.props.extensionsController.extensionHostAPI).pipe(
                            switchMap(extensionHostAPI =>
                                wrapRemoteObservable(extensionHostAPI.getReferences(parameters))
                            )
                        ),
                    {
                        context: { includeDeclaration: false },
                    }
                ),
                {
                    // File history view.
                    registrationOptions: { id: 'history', container: ContributableViewContainer.Panel },
                    provider: subjectChanges.pipe(
                        map((subject: PanelSubject) => ({
                            title: 'History',
                            content: '',
                            priority: 150,
                            locationProvider: undefined,
                            reactElement: (
                                <RepoRevisionSidebarCommits
                                    key="commits"
                                    repoID={this.props.repoID}
                                    revision={subject.revision}
                                    filePath={subject.filePath}
                                    history={this.props.history}
                                    location={this.props.location}
                                />
                            ),
                        }))
                    ),
                },
            ])
        )
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return null
    }
}
