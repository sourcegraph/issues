import { action } from '@storybook/addon-actions'
import { storiesOf } from '@storybook/react'
import { noop } from 'lodash'
import BitbucketIcon from 'mdi-react/BitbucketIcon'
import GithubIcon from 'mdi-react/GithubIcon'
import GitlabIcon from 'mdi-react/GitlabIcon'
import React, { useEffect, useState } from 'react'
import { PhabricatorIcon } from '../../../../shared/src/components/icons'
import { WebStory } from '../../components/WebStory'
import { InstallBrowserExtensionPopover } from './InstallBrowserExtensionPopover'

const onClose = action('onClose')
const onRejection = action('onRejection')
const onClickInstall = action('onClickInstall')

const { add } = storiesOf('web/repo/actions/InstallBrowserExtensionPopover', module).addDecorator(story => (
    <div className="container mt-3">{story()}</div>
))

add('GitHub', () => (
    <WebStory>
        {() => {
            const serviceType = 'github'
            const targetID = `view-on-${serviceType}`
            const [open, setOpen] = useState(false)
            // The popover cannot be open on initial render
            // since the target element isn't in the DOM yet
            useEffect(() => {
                setTimeout(() => setOpen(true), 0)
            }, [])
            return (
                <>
                    <button className="btn" id={targetID} onClick={() => setOpen(isOpen => !isOpen)}>
                        <GithubIcon className="icon-inline" />
                    </button>
                    <InstallBrowserExtensionPopover
                        url=""
                        serviceType={serviceType}
                        onClose={onClose}
                        onRejection={onRejection}
                        onClickInstall={onClickInstall}
                        targetID={targetID}
                        isOpen={open}
                        toggle={noop}
                    />
                </>
            )
        }}
    </WebStory>
))

// Disable Chromatic for the non-GitHub popovers since they are mostly the same

add(
    'GitLab',
    () => (
        <WebStory>
            {() => {
                const serviceType = 'gitlab'
                const targetID = `view-on-${serviceType}`
                const [open, setOpen] = useState(false)
                useEffect(() => {
                    setTimeout(() => setOpen(true), 0)
                }, [])
                return (
                    <>
                        <button className="btn" id={targetID} onClick={() => setOpen(isOpen => !isOpen)}>
                            <GitlabIcon className="icon-inline" />
                        </button>
                        <InstallBrowserExtensionPopover
                            url=""
                            serviceType={serviceType}
                            onClose={onClose}
                            onRejection={onRejection}
                            onClickInstall={onClickInstall}
                            targetID={targetID}
                            isOpen={open}
                            toggle={noop}
                        />
                    </>
                )
            }}
        </WebStory>
    ),
    {
        chromatic: {
            disable: true,
        },
    }
)

add(
    'Phabricator',
    () => (
        <WebStory>
            {() => {
                const serviceType = 'phabricator'
                const targetID = `view-on-${serviceType}`
                const [open, setOpen] = useState(false)
                useEffect(() => {
                    setTimeout(() => setOpen(true), 0)
                }, [])
                return (
                    <>
                        <button className="btn" id={targetID} onClick={() => setOpen(isOpen => !isOpen)}>
                            <PhabricatorIcon className="icon-inline" />
                        </button>
                        <InstallBrowserExtensionPopover
                            url=""
                            serviceType={serviceType}
                            onClose={onClose}
                            onRejection={onRejection}
                            onClickInstall={onClickInstall}
                            targetID={targetID}
                            isOpen={open}
                            toggle={noop}
                        />
                    </>
                )
            }}
        </WebStory>
    ),
    {
        chromatic: {
            disable: true,
        },
    }
)

add(
    'Bitbucket server',
    () => (
        <WebStory>
            {() => {
                const serviceType = 'bitbucketServer'
                const targetID = `view-on-${serviceType}`
                const [open, setOpen] = useState(false)
                useEffect(() => {
                    setTimeout(() => setOpen(true), 0)
                }, [])
                return (
                    <>
                        <button className="btn" id={targetID} onClick={() => setOpen(isOpen => !isOpen)}>
                            <BitbucketIcon className="icon-inline" />
                        </button>

                        <InstallBrowserExtensionPopover
                            url=""
                            serviceType={serviceType}
                            onClose={onClose}
                            onRejection={onRejection}
                            onClickInstall={onClickInstall}
                            targetID={targetID}
                            isOpen={open}
                            toggle={noop}
                        />
                    </>
                )
            }}
        </WebStory>
    ),
    {
        chromatic: {
            disable: true,
        },
    }
)
