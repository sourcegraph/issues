import React from 'react'
import { InstallBrowserExtensionAlert } from './InstallBrowserExtensionAlert'
import { mount } from 'enzyme'
import { noop } from 'lodash'

describe('InstallBrowserExtensionAlert', () => {
    const serviceTypes = ['github', 'gitlab', 'phabricator', 'bitbucketServer', null] as const
    const integrationTypes = ['Chrome', 'non-Chrome', 'native integration'] as const
    for (const serviceType of serviceTypes) {
        for (const integrationType of integrationTypes) {
            test(`${serviceType ?? 'none'} (${integrationType})`, () => {
                expect(
                    mount(
                        <InstallBrowserExtensionAlert
                            isChrome={integrationType === 'Chrome'}
                            onAlertDismissed={noop}
                            codeHostIntegrationMessaging={
                                integrationType === 'native integration' ? 'native-integration' : 'browser-extension'
                            }
                            externalURLs={
                                serviceType
                                    ? [
                                          {
                                              __typename: 'ExternalLink',
                                              url: '',
                                              serviceType,
                                          },
                                      ]
                                    : []
                            }
                        />
                    )
                ).toMatchSnapshot()
            })
        }
    }
})
