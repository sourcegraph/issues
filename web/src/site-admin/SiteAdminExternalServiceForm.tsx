import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import * as H from 'history'
import * as React from 'react'
import * as GQL from '../../../shared/src/graphql/schema'
import { ErrorLike } from '../../../shared/src/util/errors'
import { Form } from '../components/Form'
import { DynamicallyImportedMonacoSettingsEditor } from '../settings/DynamicallyImportedMonacoSettingsEditor'
import { AddExternalServiceOptions } from './externalServices'
import { ErrorAlert, ErrorMessage } from '../components/alerts'
import { TelemetryProps } from '../../../shared/src/telemetry/telemetryService'
import { ThemeProps } from '../../../shared/src/theme'

interface Props extends Pick<AddExternalServiceOptions, 'jsonSchema' | 'editorActions'>, ThemeProps, TelemetryProps {
    history: H.History
    input: GQL.AddExternalServiceInput
    error?: ErrorLike
    warning?: string | null
    mode: 'edit' | 'create'
    loading: boolean
    hideDisplayNameField?: boolean
    submitName?: string
    onSubmit: (event?: React.FormEvent<HTMLFormElement>) => void
    onChange: (change: GQL.AddExternalServiceInput) => void
}

/**
 * Form for submitting a new or updated external service.
 */
export class SiteAdminExternalServiceForm extends React.Component<Props, {}> {
    public render(): JSX.Element | null {
        return (
            <Form className="external-service-form" onSubmit={this.props.onSubmit}>
                {this.props.error && <ErrorAlert error={this.props.error} history={this.props.history} />}
                {this.props.warning && (
                    <div className="alert alert-warning">
                        <h4>Warning</h4>
                        <ErrorMessage error={this.props.warning} history={this.props.history} />
                    </div>
                )}
                {this.props.hideDisplayNameField || (
                    <div className="form-group">
                        <label className="font-weight-bold" htmlFor="test-external-service-form-display-name">
                            Display name:
                        </label>
                        <input
                            id="test-external-service-form-display-name"
                            type="text"
                            className="form-control"
                            required={true}
                            autoCorrect="off"
                            autoComplete="off"
                            autoFocus={true}
                            spellCheck={false}
                            value={this.props.input.displayName}
                            onChange={this.onDisplayNameChange}
                            disabled={this.props.loading}
                        />
                    </div>
                )}

                <div className="form-group">
                    <DynamicallyImportedMonacoSettingsEditor
                        // DynamicallyImportedMonacoSettingsEditor does not re-render the passed input.config
                        // if it thinks the config is dirty. We want to always replace the config if the kind changes
                        // so the editor is keyed on the kind.
                        value={this.props.input.config}
                        jsonSchema={this.props.jsonSchema}
                        canEdit={false}
                        loading={this.props.loading}
                        height={350}
                        isLightTheme={this.props.isLightTheme}
                        onChange={this.onConfigChange}
                        history={this.props.history}
                        actions={this.props.editorActions}
                        className="test-external-service-editor"
                        telemetryService={this.props.telemetryService}
                    />
                    <p className="form-text text-muted">
                        <small>Use Ctrl+Space for completion, and hover over JSON properties for documentation.</small>
                    </p>
                </div>
                <button
                    type="submit"
                    className={`btn btn-primary mb-3 ${
                        this.props.mode === 'create'
                            ? 'test-add-external-service-button'
                            : 'test-update-external-service-button'
                    }`}
                    disabled={this.props.loading}
                >
                    {this.props.loading && <LoadingSpinner className="icon-inline" />}
                    {this.props.submitName ?? (this.props.mode === 'edit' ? 'Update repositories' : 'Add repositories')}
                </button>
            </Form>
        )
    }

    private onDisplayNameChange: React.ChangeEventHandler<HTMLInputElement> = event => {
        this.props.onChange({ ...this.props.input, displayName: event.currentTarget.value })
    }

    private onConfigChange = (config: string): void => {
        this.props.onChange({ ...this.props.input, config })
    }
}
