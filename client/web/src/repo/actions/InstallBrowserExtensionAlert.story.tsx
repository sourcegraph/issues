import { action } from '@storybook/addon-actions'
import { storiesOf } from '@storybook/react'
import React from 'react'
import { WebStory } from '../../components/WebStory'
import { InstallBrowserExtensionAlert } from './InstallBrowserExtensionAlert'

const onAlertDismissed = action('onAlertDismissed')

const { add } = storiesOf('web/repo/actions/InstallBrowserExtensionAlert', module).addDecorator(story => (
    <div className="container mt-3">{story()}</div>
))

// Disable Chromatic for the non-GitHub alerts since they are mostly the same

const services = ['github', 'gitlab', 'phabricator', 'bitbucketServer'] as const

for (const serviceType of services) {
    add(
        `${serviceType} (Chrome)`,
        () => (
            <WebStory>
                {() => (
                    <InstallBrowserExtensionAlert
                        isChrome={true}
                        onAlertDismissed={onAlertDismissed}
                        codeHostIntegrationMessaging="browser-extension"
                        externalURLs={[
                            {
                                __typename: 'ExternalLink',
                                url: '',
                                serviceType,
                            },
                        ]}
                    />
                )}
            </WebStory>
        ),
        {
            chromatic: {
                disable: serviceType !== 'github',
            },
        }
    )

    add(
        `${serviceType} (non-Chrome)`,
        () => (
            <WebStory>
                {() => (
                    <InstallBrowserExtensionAlert
                        isChrome={false}
                        onAlertDismissed={onAlertDismissed}
                        codeHostIntegrationMessaging="browser-extension"
                        externalURLs={[
                            {
                                __typename: 'ExternalLink',
                                url: '',
                                serviceType,
                            },
                        ]}
                    />
                )}
            </WebStory>
        ),
        {
            chromatic: {
                disable: serviceType !== 'github',
            },
        }
    )

    add(
        `${serviceType} (native integration installed)`,
        () => (
            <WebStory>
                {() => (
                    <InstallBrowserExtensionAlert
                        isChrome={false}
                        onAlertDismissed={onAlertDismissed}
                        codeHostIntegrationMessaging="native-integration"
                        externalURLs={[
                            {
                                __typename: 'ExternalLink',
                                url: '',
                                serviceType,
                            },
                        ]}
                    />
                )}
            </WebStory>
        ),
        {
            chromatic: {
                disable: serviceType !== 'github',
            },
        }
    )
}
