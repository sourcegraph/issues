import { RouteComponentProps } from 'react-router'
import { fetchAllConfigAndSettings, fetchMonitoringStats } from './backend'
import React, { useMemo } from 'react'
import { DynamicallyImportedMonacoSettingsEditor } from '../settings/DynamicallyImportedMonacoSettingsEditor'
import awsCodeCommitJSON from '../../../../schema/aws_codecommit.schema.json'
import bitbucketCloudSchemaJSON from '../../../../schema/bitbucket_cloud.schema.json'
import bitbucketServerSchemaJSON from '../../../../schema/bitbucket_server.schema.json'
import githubSchemaJSON from '../../../../schema/github.schema.json'
import gitlabSchemaJSON from '../../../../schema/gitlab.schema.json'
import gitoliteSchemaJSON from '../../../../schema/gitolite.schema.json'
import otherExternalServiceSchemaJSON from '../../../../schema/other_external_service.schema.json'
import phabricatorSchemaJSON from '../../../../schema/phabricator.schema.json'
import settingsSchemaJSON from '../../../../schema/settings.schema.json'
import siteSchemaJSON from '../../../../schema/site.schema.json'
import { PageTitle } from '../components/PageTitle'
import { useObservable } from '../../../shared/src/util/useObservable'
import { mapValues, values } from 'lodash'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { TelemetryProps } from '../../../shared/src/telemetry/telemetryService'
import { ThemeProps } from '../../../shared/src/theme'
import { ExternalServiceKind } from '../../../shared/src/graphql-operations'

/**
 * Minimal shape of a JSON Schema. These values are treated as opaque, so more specific types are
 * not needed.
 */
interface JSONSchema {
    $id: string
    definitions?: Record<string, { type: string }>
}

const externalServices: Record<ExternalServiceKind, JSONSchema> = {
    AWSCODECOMMIT: awsCodeCommitJSON,
    BITBUCKETCLOUD: bitbucketCloudSchemaJSON,
    BITBUCKETSERVER: bitbucketServerSchemaJSON,
    GITHUB: githubSchemaJSON,
    GITLAB: gitlabSchemaJSON,
    GITOLITE: gitoliteSchemaJSON,
    OTHER: otherExternalServiceSchemaJSON,
    PHABRICATOR: phabricatorSchemaJSON,
}

const allConfigSchema = {
    $id: 'all.schema.json#',
    allowComments: true,
    additionalProperties: false,
    properties: {
        site: siteSchemaJSON,
        externalServices: {
            type: 'object',
            properties: mapValues(externalServices, schema => ({ type: 'array', items: schema })),
        },
        settings: {
            type: 'object',
            properties: {
                subjects: {
                    type: 'array',
                    items: {
                        type: 'object',
                        properties: {
                            __typename: {
                                type: 'string',
                            },
                            settingsURL: {
                                type: ['string', 'null'],
                            },
                            contents: {
                                ...settingsSchemaJSON,
                                type: ['object', 'null'],
                            },
                        },
                    },
                },
                final: settingsSchemaJSON,
            },
        },
        alerts: {
            type: 'array',
            items: {
                type: 'object',
            },
        },
    },
    definitions: values(externalServices)
        .map(schema => schema.definitions)
        .concat([siteSchemaJSON.definitions, settingsSchemaJSON.definitions])
        .reduce((allDefinitions, definitions) => ({ ...allDefinitions, ...definitions }), {}),
}

interface Props extends RouteComponentProps, ThemeProps, TelemetryProps {}

export const SiteAdminReportBugPage: React.FunctionComponent<Props> = ({ isLightTheme, telemetryService, history }) => {
    const monitoringDaysBack = 7
    const monitoringStats = useObservable(useMemo(() => fetchMonitoringStats(monitoringDaysBack), []))
    const allConfig = useObservable(useMemo(fetchAllConfigAndSettings, []))
    return (
        <div>
            <PageTitle title="Report a bug - Admin" />
            <h2>Report a bug</h2>
            <p>
                <a
                    target="_blank"
                    rel="noopener noreferrer"
                    href="https://github.com/sourcegraph/sourcegraph/issues/new?assignees=&labels=&template=bug_report.md&title="
                >
                    Create an issue on the public issue tracker
                </a>
                , and include a description of the bug along with the info below (with secrets redacted). If the report
                contains sensitive information that should not be public, email the report to{' '}
                <a target="_blank" rel="noopener noreferrer" href="mailto:support@sourcegraph.com">
                    support@sourcegraph.com
                </a>{' '}
                instead.
            </p>
            <div className="card-header alert alert-warning">
                <div>
                    Please redact any secrets before sharing, whether on the public issue tracker or with
                    support@sourcegraph.com.
                </div>
            </div>
            {allConfig === undefined || monitoringStats === undefined ? (
                <LoadingSpinner className="icon-inline mt-2" />
            ) : (
                <DynamicallyImportedMonacoSettingsEditor
                    value={JSON.stringify(
                        monitoringStats ? { ...allConfig, ...monitoringStats } : { ...allConfig, alerts: null },
                        undefined,
                        2
                    )}
                    jsonSchema={allConfigSchema}
                    canEdit={false}
                    height={800}
                    isLightTheme={isLightTheme}
                    history={history}
                    readOnly={true}
                    telemetryService={telemetryService}
                />
            )}
        </div>
    )
}
