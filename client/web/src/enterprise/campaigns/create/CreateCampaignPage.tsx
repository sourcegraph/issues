import React, { useCallback, useMemo, useState } from 'react'
import { PageTitle } from '../../../components/PageTitle'
import { PageHeader } from '../../../components/PageHeader'
import { CampaignsIconFlushLeft } from '../icons'
import { BreadcrumbSetters } from '../../../components/Breadcrumbs'
import { AuthenticatedUser } from '../../../auth'
import helloWorldSample from './samples/empty.campaign.yaml'
import combySample from './samples/comby.campaign.yaml'
import goImportsSample from './samples/go-imports.campaign.yaml'
import minimalSample from './samples/minimal.campaign.yaml'
import classNames from 'classnames'
import { CodeSnippet } from '../../../../../branded/src/components/CodeSnippet'

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
        <li className="nav-item">
            <a href="" onClick={onClick} className={classNames('nav-link', active && 'active')}>
                {sample.name}
            </a>
        </li>
    )
}

interface Sample {
    name: string
    file: string
}

const samples: Sample[] = [
    { name: 'hello-world.campaign.yaml', file: helloWorldSample },
    { name: 'modify-with-comby.campaign.yaml', file: combySample },
    { name: 'update-go-imports.campaign.yaml', file: goImportsSample },
    { name: 'minimal.campaign.yaml', file: minimalSample },
]

export interface CreateCampaignPageProps extends BreadcrumbSetters {
    authenticatedUser: Pick<AuthenticatedUser, 'username'>
}

export const CreateCampaignPage: React.FunctionComponent<CreateCampaignPageProps> = ({
    authenticatedUser,
    useBreadcrumb,
}) => {
    const [selectedSample, setSelectedSample] = useState<Sample>(samples[0])
    useBreadcrumb(useMemo(() => ({ element: <>Create campaign</>, key: 'createCampaignPage' }), []))
    return (
        <>
            <PageTitle title="Create campaign" />
            <PageHeader icon={CampaignsIconFlushLeft} title="Create campaign" />
            <div className="pt-3">
                <h2>1. Write a campaign spec YAML file</h2>
                <p>
                    The campaign spec (
                    <a
                        href="https://docs.sourcegraph.com/user/campaigns#campaign-specs"
                        rel="noopener noreferrer"
                        target="_blank"
                    >
                        syntax reference
                    </a>
                    ) describes what the campaign does. You'll provide it when previewing, creating, and updating
                    campaigns. We recommend committing it to source control.
                </p>
                <h4>Examples:</h4>
                <ul className="nav nav-pills mb-2">
                    {samples.map(sample => (
                        <SampleTabHeader
                            key={sample.name}
                            sample={sample}
                            active={selectedSample.name === sample.name}
                            setSelectedSample={setSelectedSample}
                        />
                    ))}
                </ul>
                <CodeSnippet code={selectedSample.file} language="yaml" className="mb-4" />
                <h2>2. Preview the campaign with Sourcegraph CLI</h2>
                <p>
                    Use the{' '}
                    <a href="https://github.com/sourcegraph/src-cli" rel="noopener noreferrer" target="_blank">
                        Sourcegraph CLI (src)
                    </a>{' '}
                    to preview the commits and changesets that your campaign will make:
                </p>
                <CodeSnippet
                    code={`src campaign preview -namespace ${authenticatedUser.username} -f ${selectedSample.name}`}
                    language="shell"
                    className="mb-3"
                />
                <p>
                    Follow the URL printed in your terminal to see the preview and (when you're ready) create the
                    campaign.
                </p>
                <hr className="mt-4" />
                <p className="text-muted">
                    Want more help? See{' '}
                    <a href="/help/user/campaigns" rel="noopener noreferrer" target="_blank">
                        campaigns documentation
                    </a>
                    .
                </p>
            </div>
        </>
    )
}
