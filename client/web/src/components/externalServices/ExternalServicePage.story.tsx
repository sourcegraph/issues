import { storiesOf } from '@storybook/react'
import React from 'react'
import { ExternalServicePage } from './ExternalServicePage'
import { NOOP_TELEMETRY_SERVICE } from '../../../../shared/src/telemetry/telemetryService'
import { fetchExternalService as _fetchExternalService } from './backend'
import { of } from 'rxjs'
import { ExternalServiceKind } from '../../graphql-operations'
import { WebStory } from '../WebStory'

const { add } = storiesOf('web/External services/ExternalServicePage', module)
    .addDecorator(story => <div className="p-3 container">{story()}</div>)
    .addParameters({
        chromatic: {
            // Delay screenshot taking, so Monaco has some time to get syntax highlighting prepared.
            delay: 2000,
        },
    })

const fetchExternalService: typeof _fetchExternalService = () =>
    of({
        id: 'service123',
        kind: ExternalServiceKind.GITHUB,
        warning: null,
        config: '{"githubconfig": true}',
        displayName: 'GitHub.com',
        webhookURL: null,
    })

add('View external service config', () => (
    <WebStory>
        {webProps => (
            <ExternalServicePage
                {...webProps}
                afterUpdateRoute="/site-admin/after"
                telemetryService={NOOP_TELEMETRY_SERVICE}
                externalServiceID="service123"
                fetchExternalService={fetchExternalService}
                autoFocusForm={false}
            />
        )}
    </WebStory>
))
