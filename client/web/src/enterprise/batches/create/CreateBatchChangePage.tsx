import classNames from 'classnames'
import React, { useCallback, useState } from 'react'

import { CodeSnippet } from '@sourcegraph/branded/src/components/CodeSnippet'
import { Link } from '@sourcegraph/shared/src/components/Link'
import {
    SettingsCascadeProps,
    SettingsOrgSubject,
    SettingsSubject,
    SettingsUserSubject,
} from '@sourcegraph/shared/src/settings/settings'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { isErrorLike } from '@sourcegraph/shared/src/util/errors'
import { Container, PageHeader } from '@sourcegraph/wildcard'

import batchSpecSchemaJSON from '../../../../../../schema/batch_spec.schema.json'
import { BatchChangesIcon } from '../../../batches/icons'
import { ErrorAlert } from '../../../components/alerts'
import { PageTitle } from '../../../components/PageTitle'
import { SidebarGroup, SidebarGroupHeader, SidebarGroupItems } from '../../../components/Sidebar'
import { BatchSpecExecutionCreateFields } from '../../../graphql-operations'
import { Settings } from '../../../schema/settings.schema'
import { MonacoSettingsEditor } from '../../../settings/MonacoSettingsEditor'

import { createBatchSpecExecution } from './backend'
import styles from './CreateBatchChangePage.module.scss'
import combySample from './samples/comby.batch.yaml'
import helloWorldSample from './samples/empty.batch.yaml'
import goImportsSample from './samples/go-imports.batch.yaml'
import minimalSample from './samples/minimal.batch.yaml'

interface SampleTabHeaderProps {
    sample: Sample
    active: boolean
    setSelectedSample: (sample: Sample) => void
}

const SampleTabHeader: React.FunctionComponent<SampleTabHeaderProps> = ({ sample, active, setSelectedSample }) => {
    const onClick = useCallback<React.MouseEventHandler>(
        event => {
            event.preventDefault()
            setSelectedSample(sample)
        },
        [setSelectedSample, sample]
    )
    return (
        <button
            type="button"
            onClick={onClick}
            className={classNames(
                'btn text-left sidebar__link--inactive d-flex sidebar-nav-link w-100',
                active && 'btn-primary'
            )}
        >
            {sample.name}
        </button>
    )
}

interface Sample {
    name: string
    file: string
}

const samples: Sample[] = [
    { name: 'Hello world', file: helloWorldSample },
    { name: 'Modify with comby', file: combySample },
    { name: 'Update go imports', file: goImportsSample },
    { name: 'Minimal', file: minimalSample },
]

export interface CreateBatchChangePageProps extends SettingsCascadeProps<Settings>, ThemeProps {
    headingElement: 'h1' | 'h2'
}

export const CreateBatchChangePage: React.FunctionComponent<CreateBatchChangePageProps> = ({
    settingsCascade,
    isLightTheme,
    headingElement,
}) => {
    const isBatchChangesExecutionEnabled = Boolean(
        settingsCascade !== null &&
            !isErrorLike(settingsCascade.final) &&
            settingsCascade.final?.experimentalFeatures?.batchChangesExecution
    )
    const [selectedSample, setSelectedSample] = useState<Sample>(samples[0])

    return (
        <>
            <PageTitle title="Create batch change" />
            <PageHeader
                path={[{ icon: BatchChangesIcon, text: 'Create batch change' }]}
                headingElement={headingElement}
                description={
                    <>
                        Follow these steps to create a Batch Change. Need help? View the{' '}
                        <a href="/help/batch_changes" rel="noopener noreferrer" target="_blank">
                            documentation.
                        </a>
                    </>
                }
                className="mb-3"
            />
            <h2>1. Write a batch spec YAML file</h2>
            <Container className="mb-3">
                <p className="mb-0">
                    The batch spec (
                    <a
                        href="https://docs.sourcegraph.com/batch_changes/references/batch_spec_yaml_reference"
                        rel="noopener noreferrer"
                        target="_blank"
                    >
                        syntax reference
                    </a>
                    ) describes what the batch change does. You'll provide it when previewing, creating, and updating
                    batch changes. We recommend committing it to source control.
                </p>
            </Container>
            <div className="d-flex mb-3">
                <div className="flex-shrink-0">
                    <SidebarGroup>
                        <SidebarGroupItems>
                            <SidebarGroupHeader label="Examples" />
                            {samples.map(sample => (
                                <SampleTabHeader
                                    key={sample.name}
                                    sample={sample}
                                    active={selectedSample.name === sample.name}
                                    setSelectedSample={setSelectedSample}
                                />
                            ))}
                        </SidebarGroupItems>
                    </SidebarGroup>
                </div>
                <Container className="ml-3 flex-grow-1">
                    <CodeSnippet code={selectedSample.file} language="yaml" className="mb-0" />
                </Container>
            </div>
            <h2>2. Preview the batch change with Sourcegraph CLI</h2>
            <Container className="mb-3">
                <p>
                    Use the{' '}
                    <a href="https://github.com/sourcegraph/src-cli" rel="noopener noreferrer" target="_blank">
                        Sourcegraph CLI (src)
                    </a>{' '}
                    to preview the commits and changesets that your batch change will make:
                </p>
                <CodeSnippet code={`src batch preview -f ${selectedSample.name}`} language="bash" className="mb-3" />
                <p className="mb-0">
                    Follow the URL printed in your terminal to see the preview and (when you're ready) create the batch
                    change.
                </p>
            </Container>
            {isBatchChangesExecutionEnabled && (
                <CreateBatchSpecExecutionForm
                    settingsCascade={settingsCascade}
                    isLightTheme={isLightTheme}
                    initialContent={selectedSample.file}
                />
            )}
        </>
    )
}

interface CreateBatchSpecExecutionFormProps extends ThemeProps, SettingsCascadeProps<Settings> {
    initialContent: string
}

const CreateBatchSpecExecutionForm: React.FunctionComponent<CreateBatchSpecExecutionFormProps> = ({
    initialContent,
    isLightTheme,
    settingsCascade,
}) => {
    const namespaces: SettingsSubject[] =
        (settingsCascade !== null &&
            !isErrorLike(settingsCascade) &&
            settingsCascade.subjects !== null &&
            settingsCascade.subjects.map(({ subject }) => subject).filter(subject => !isErrorLike(subject))) ||
        []
    const userNamespace = namespaces.find(
        (namespace): namespace is SettingsUserSubject => namespace.__typename === 'User'
    )
    const organizationNamespaces = namespaces.filter(
        (namespace): namespace is SettingsOrgSubject => namespace.__typename === 'Org'
    )

    if (!userNamespace) {
        throw new Error('Bye')
    }

    const [content, setContent] = useState<string>(initialContent)
    const [batchSpecExecution, setBatchSpecExecution] = useState<BatchSpecExecutionCreateFields>()
    const [isLoading, setIsLoading] = useState<boolean | Error>(false)
    const [selectedNamespace, setSelectedNamespace] = useState<string>(userNamespace.id)
    const onSelectNamespace = useCallback<React.ChangeEventHandler<HTMLSelectElement>>(event => {
        setSelectedNamespace(event.target.value)
    }, [])
    const submitBatchSpec = useCallback<React.MouseEventHandler>(async () => {
        setBatchSpecExecution(undefined)
        setIsLoading(true)
        try {
            const exec = await createBatchSpecExecution(content, selectedNamespace)
            setBatchSpecExecution(exec)
            setIsLoading(false)
        } catch (error) {
            setIsLoading(error)
        }
    }, [content, selectedNamespace])

    return (
        <>
            <h2>
                <span className="badge badge-info text-uppercase">Experimental</span> Or run your batch spec server side
            </h2>
            <Container>
                <div className="form-group d-flex align-items-center justify-content-end">
                    <label className="text-nowrap mr-2 mb-0" htmlFor={NAMESPACE_SELECTOR_ID}>
                        <strong>Select namespace:</strong>
                    </label>
                    <select
                        className={classNames(styles.namespaceSelector, 'form-control')}
                        id={NAMESPACE_SELECTOR_ID}
                        value={selectedNamespace}
                        onChange={onSelectNamespace}
                    >
                        <option value={userNamespace.id}>{userNamespace.displayName ?? userNamespace.username}</option>
                        {organizationNamespaces.map(namespace => (
                            <option key={namespace.id} value={namespace.id}>
                                {namespace.displayName ?? namespace.name}
                            </option>
                        ))}
                    </select>
                </div>
                <MonacoSettingsEditor
                    isLightTheme={isLightTheme}
                    language="yaml"
                    value={content}
                    jsonSchema={batchSpecSchemaJSON}
                    className="mb-3"
                    onChange={setContent}
                />
                <button
                    type="button"
                    className="btn btn-primary"
                    onClick={submitBatchSpec}
                    disabled={isLoading === true}
                >
                    Run batch spec
                </button>
                {batchSpecExecution && (
                    <div className="mt-3 mb-0 alert alert-success">
                        Running batch spec.{' '}
                        <Link
                            to={`${batchSpecExecution.namespace.url}/batch-changes/executions/${batchSpecExecution.id}`}
                        >
                            Check it out here.
                        </Link>
                    </div>
                )}
                {isErrorLike(isLoading) && <ErrorAlert error={isLoading} />}
            </Container>
        </>
    )
}

const NAMESPACE_SELECTOR_ID = 'batch-spec-execution-namespace-selector'
