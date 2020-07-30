import WarningIcon from 'mdi-react/WarningIcon'
import * as React from 'react'
import { LinkOrSpan } from '../../../shared/src/components/LinkOrSpan'
import { Path } from '../../../shared/src/components/Path'
import { ConfiguredRegistryExtension, isExtensionEnabled } from '../../../shared/src/extensions/extension'
import * as GQL from '../../../shared/src/graphql/schema'
import { PlatformContextProps } from '../../../shared/src/platform/context'
import { ExtensionManifest } from '../../../shared/src/schema/extensionSchema'
import { SettingsCascadeProps } from '../../../shared/src/settings/settings'
import { isErrorLike } from '../../../shared/src/util/errors'
import { isExtensionAdded } from './extension/extension'
import { ExtensionConfigurationState } from './extension/ExtensionConfigurationState'
import { WorkInProgressBadge } from './extension/WorkInProgressBadge'
import { ExtensionToggle } from './ExtensionToggle'
import { isEncodedImage } from '../../../shared/src/util/icon'

interface Props extends SettingsCascadeProps, PlatformContextProps<'updateSettings'> {
    node: Pick<
        ConfiguredRegistryExtension<
            Pick<
                GQL.RegistryExtension,
                'id' | 'extensionIDWithoutRegistry' | 'isWorkInProgress' | 'viewerCanAdminister' | 'url'
            >
        >,
        'id' | 'manifest' | 'registryExtension'
    >
    subject: Pick<GQL.SettingsSubject, 'id' | 'viewerCanAdminister'>
}

const stopPropagation: React.MouseEventHandler<HTMLElement> = event => {
    event.stopPropagation()
}

/** Displays an extension as a card. */
export class ExtensionCard extends React.PureComponent<Props> {
    public render(): JSX.Element | null {
        const { node, ...props } = this.props
        const manifest: ExtensionManifest | undefined =
            node.manifest && !isErrorLike(node.manifest) ? node.manifest : undefined
        let iconURL: URL | undefined
        try {
            if (manifest?.icon) {
                iconURL = new URL(manifest.icon)
            }
        } catch {
            // noop
        }

        return (
            <div className="d-flex">
                <div className="extension-card card">
                    <div
                        className="card-body extension-card__body d-flex flex-column position-relative"
                        // Prevent toggle clicks from propagating to the stretched-link (and
                        // navigating to the extension detail page).
                        onClick={stopPropagation}
                    >
                        <div className="d-flex">
                            {manifest?.icon &&
                                iconURL &&
                                iconURL.protocol === 'data:' &&
                                isEncodedImage(manifest.icon) && (
                                    <img className="extension-card__icon mr-2" src={manifest.icon} />
                                )}
                            <div className="text-truncate w-100">
                                <div className="d-flex align-items-center">
                                    <h4 className="card-title extension-card__body-title mb-0 mr-1 text-truncate font-weight-normal flex-1">
                                        <LinkOrSpan to={node.registryExtension?.url} className="stretched-link">
                                            <Path
                                                path={
                                                    node.registryExtension
                                                        ? node.registryExtension.extensionIDWithoutRegistry
                                                        : node.id
                                                }
                                            />
                                        </LinkOrSpan>
                                    </h4>
                                    {node.registryExtension?.isWorkInProgress && (
                                        <WorkInProgressBadge
                                            viewerCanAdminister={node.registryExtension.viewerCanAdminister}
                                        />
                                    )}
                                    {props.subject &&
                                        (props.subject.viewerCanAdminister ? (
                                            <ExtensionToggle
                                                extension={node}
                                                settingsCascade={this.props.settingsCascade}
                                                platformContext={this.props.platformContext}
                                                className="extension-card__toggle"
                                            />
                                        ) : (
                                            <ExtensionConfigurationState
                                                isAdded={isExtensionAdded(props.settingsCascade.final, node.id)}
                                                isEnabled={isExtensionEnabled(props.settingsCascade.final, node.id)}
                                                enabledIconOnly={true}
                                                className="small"
                                            />
                                        ))}
                                </div>
                                <div className="mt-1">
                                    {node.manifest ? (
                                        isErrorLike(node.manifest) ? (
                                            <span className="text-danger small" title={node.manifest.message}>
                                                <WarningIcon className="icon-inline" /> Invalid manifest
                                            </span>
                                        ) : (
                                            node.manifest.description && (
                                                <div className="text-muted text-truncate">
                                                    {node.manifest.description}
                                                </div>
                                            )
                                        )
                                    ) : (
                                        <span className="text-warning small">
                                            <WarningIcon className="icon-inline" /> No manifest
                                        </span>
                                    )}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        )
    }
}
