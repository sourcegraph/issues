import classNames from 'classnames'
import React, { useCallback, useState } from 'react'

import { CodeSnippet } from '@sourcegraph/branded/src/components/CodeSnippet'
import { Container, PageHeader } from '@sourcegraph/wildcard'

import { BatchChangesIcon } from '../../../batches/icons'
import { PageTitle } from '../../../components/PageTitle'
import { SidebarGroup, SidebarGroupHeader, SidebarGroupItems } from '../../../components/Sidebar'

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

export interface CreateBatchChangePageProps {
    headingElement: 'h1' | 'h2'
}

export const CreateBatchChangePage: React.FunctionComponent<CreateBatchChangePageProps> = ({ headingElement }) => {
    const [selectedSample, setSelectedSample] = useState<Sample>(samples[0])
    return (
        <>
            <PageTitle title="Create batch change" />
            <PageHeader
                // TODO: correct to URL for namespaced create pages.
                path={[{ icon: BatchChangesIcon, to: '/batch-changes' }, { text: 'Create batch change' }]}
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
                <Container className={classNames(styles.codeContainer, 'ml-3 flex-grow-1')}>
                    <CodeSnippet
                        code={selectedSample.file}
                        language="yaml"
                        className={classNames(styles.codeSnippet, 'mb-0')}
                    />
                </Container>
            </div>
            <h2>2. Preview the batch change with Sourcegraph CLI</h2>
            <Container>
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
        </>
    )
}
