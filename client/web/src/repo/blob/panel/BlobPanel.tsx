import * as H from 'history'
import { isEqual } from 'lodash'
import * as React from 'react'
import { from, Observable, Subject, Subscription } from 'rxjs'
import { distinctUntilChanged, map, startWith, switchMap, tap } from 'rxjs/operators'
import { getActiveCodeEditorPosition } from '../../../../../shared/src/api/client/services/viewerService'
import * as extensionApiTypes from '@sourcegraph/extension-api-types'
import { Entry } from '../../../../../shared/src/api/client/services/registry'
import {
    ProvidePanelViewSignature,
    PanelViewProviderRegistrationOptions,
} from '../../../../../shared/src/api/client/services/panelViews'
import {
    ContributableViewContainer,
    ReferenceParameters,
    TextDocumentPositionParameters,
} from '../../../../../shared/src/api/protocol'
import { ActivationProps } from '../../../../../shared/src/components/activation/Activation'
import { ExtensionsControllerProps } from '../../../../../shared/src/extensions/controller'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { PlatformContextProps } from '../../../../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../../../../shared/src/settings/settings'
import { AbsoluteRepoFile, ModeSpec, parseHash, UIPositionSpec } from '../../../../../shared/src/util/url'
import { RepoHeaderContributionsLifecycleProps } from '../../RepoHeader'
import { RepoRevisionSidebarCommits } from '../../RepoRevisionSidebarCommits'
import { ThemeProps } from '../../../../../shared/src/theme'
import { AuthenticatedUser } from '../../../auth'
import { MaybeLoadingResult } from '@sourcegraph/codeintellify'
import { wrapRemoteObservable } from '../../../../../shared/src/api/client/api/common'

interface Props
    extends AbsoluteRepoFile,
        Partial<UIPositionSpec>,
        ModeSpec,
        RepoHeaderContributionsLifecycleProps,
        SettingsCascadeProps,
        PlatformContextProps,
        ExtensionsControllerProps,
        ThemeProps,
        ActivationProps {
    location: H.Location
    history: H.History
    repoID: GQL.ID
    repoName: string
    commitID: string
    authenticatedUser: AuthenticatedUser | null
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
            provideLocations: (parameters: P) => Observable<MaybeLoadingResult<extensionApiTypes.Location[]>>,
            extraParameters?: Pick<P, Exclude<keyof P, keyof TextDocumentPositionParameters>>
        ): Entry<PanelViewProviderRegistrationOptions, ProvidePanelViewSignature> => ({
            registrationOptions: { id, container: ContributableViewContainer.Panel },
            provider: from(this.props.extensionsController.services.viewer.activeViewerUpdates).pipe(
                map(activeEditor =>
                    activeEditor && activeEditor.type === 'CodeEditor'
                        ? {
                              ...activeEditor,
                              model: this.props.extensionsController.services.model.getPartialModel(
                                  activeEditor.resource
                              ),
                          }
                        : undefined
                ),
                map(activeEditor => {
                    const parameters: TextDocumentPositionParameters | null = getActiveCodeEditorPosition(activeEditor)
                    if (!parameters) {
                        return null
                    }
                    return {
                        title,
                        content: '',
                        priority,

                        // This disable directive is necessary because TypeScript is not yet smart
                        // enough to know that (typeof params & typeof extraParams) is P.
                        //
                        // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
                        locationProvider: provideLocations({ ...parameters, ...extraParameters } as P).pipe(
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
                    from(this.props.extensionsController.extHostAPI).pipe(
                        switchMap(extensionHostAPI => wrapRemoteObservable(extensionHostAPI.getDefinition(parameters)))
                    )
                ),
                entryForViewProviderRegistration<ReferenceParameters>(
                    'references',
                    'References',
                    180,
                    parameters =>
                        this.props.extensionsController.services.textDocumentReferences.getLocations(parameters),
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
