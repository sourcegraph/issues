import classNames from 'classnames'
import * as H from 'history'
import * as React from 'react'
import { ActionNavItemsClassProps, ActionsNavItems } from '../../../../shared/src/actions/ActionsNavItems'
import { ContributionScope } from '../../../../shared/src/api/client/context/context'
import { ContributableMenu } from '../../../../shared/src/api/protocol'
import { ExtensionsControllerProps } from '../../../../shared/src/extensions/controller'
import { PlatformContextProps } from '../../../../shared/src/platform/context'
import { TelemetryProps } from '../../../../shared/src/telemetry/telemetryService'
import { DiffOrBlobInfo, FileInfoWithContent } from '../code-hosts/shared/codeHost'
import { OpenDiffOnSourcegraph } from './OpenDiffOnSourcegraph'
import { OpenOnSourcegraph } from './OpenOnSourcegraph'
import { SignInButton } from '../code-hosts/shared/SignInButton'
import { ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import { isHTTPAuthError } from '../../../../shared/src/backend/fetch'
import { defaultRevisionToCommitID } from '../code-hosts/shared/util/fileInfo'

export interface ButtonProps {
    className?: string
}

export interface CodeViewToolbarClassProps extends ActionNavItemsClassProps {
    /**
     * Class name for the `<ul>` element wrapping all toolbar items
     */
    className?: string

    /**
     * The scope of this toolbar (e.g., the view component that it is associated with).
     */
    scope?: ContributionScope
}

export interface CodeViewToolbarProps
    extends PlatformContextProps<'forceUpdateTooltip' | 'settings' | 'requestGraphQL'>,
        ExtensionsControllerProps,
        TelemetryProps,
        CodeViewToolbarClassProps {
    sourcegraphURL: string

    /**
     * Information about the file or diff the toolbar is displayed on.
     */
    fileInfoOrError: DiffOrBlobInfo<FileInfoWithContent> | ErrorLike

    buttonProps?: ButtonProps
    onSignInClose: () => void
    location: H.Location
}

export const CodeViewToolbar: React.FunctionComponent<CodeViewToolbarProps> = props => (
    <ul className={classNames('code-view-toolbar', props.className)}>
        <ActionsNavItems
            {...props}
            listItemClass={classNames('code-view-toolbar__item', props.listItemClass)}
            menu={ContributableMenu.EditorTitle}
            extensionsController={props.extensionsController}
            platformContext={props.platformContext}
            location={props.location}
            scope={props.scope}
        />{' '}
        {isErrorLike(props.fileInfoOrError) ? (
            isHTTPAuthError(props.fileInfoOrError) ? (
                <SignInButton
                    sourcegraphURL={props.sourcegraphURL}
                    onSignInClose={props.onSignInClose}
                    className={props.actionItemClass}
                    iconClassName={props.actionItemIconClass}
                />
            ) : null
        ) : (
            <>
                {!('blob' in props.fileInfoOrError) && props.fileInfoOrError.head && props.fileInfoOrError.base && (
                    <li className={classNames('code-view-toolbar__item', props.listItemClass)}>
                        <OpenDiffOnSourcegraph
                            ariaLabel="View file diff on Sourcegraph"
                            platformContext={props.platformContext}
                            className={props.actionItemClass}
                            iconClassName={props.actionItemIconClass}
                            openProps={{
                                sourcegraphURL: props.sourcegraphURL,
                                repoName: props.fileInfoOrError.base.repoName,
                                filePath: props.fileInfoOrError.base.filePath,
                                revision: defaultRevisionToCommitID(props.fileInfoOrError.base).revision,
                                commit: {
                                    baseRev: defaultRevisionToCommitID(props.fileInfoOrError.base).revision,
                                    headRev: defaultRevisionToCommitID(props.fileInfoOrError.head).revision,
                                },
                            }}
                        />
                    </li>
                )}{' '}
                {
                    // Only show the "View file" button if we were able to fetch the file contents
                    // from the Sourcegraph instance
                    'blob' in props.fileInfoOrError && props.fileInfoOrError.blob.content !== undefined && (
                        <li className={classNames('code-view-toolbar__item', props.listItemClass)}>
                            <OpenOnSourcegraph
                                ariaLabel="View file on Sourcegraph"
                                className={props.actionItemClass}
                                iconClassName={props.actionItemIconClass}
                                openProps={{
                                    sourcegraphURL: props.sourcegraphURL,
                                    repoName: props.fileInfoOrError.blob.repoName,
                                    filePath: props.fileInfoOrError.blob.filePath,
                                    revision: defaultRevisionToCommitID(props.fileInfoOrError.blob).revision,
                                }}
                            />
                        </li>
                    )
                }
            </>
        )}
    </ul>
)
