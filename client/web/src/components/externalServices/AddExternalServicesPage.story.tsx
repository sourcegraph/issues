import { storiesOf } from '@storybook/react'
import React from 'react'
import { AddExternalServicesPage } from './AddExternalServicesPage'
import { NOOP_TELEMETRY_SERVICE } from '../../../../shared/src/telemetry/telemetryService'
import { fetchExternalService as _fetchExternalService } from './backend'
import { codeHostExternalServices, nonCodeHostExternalServices } from './externalServices'
import { WebStory } from '../WebStory'

const { add } = storiesOf('web/External services/AddExternalServicesPage', module)
    .addDecorator(story => <div className="p-3 container">{story()}</div>)
    .addParameters({
        chromatic: {
            // Delay screenshot taking, so Monaco has some time to get syntax highlighting prepared.
            delay: 2000,
        },
    })

add('Overview', () => (
    <WebStory>
        {webProps => (
            <AddExternalServicesPage
                {...webProps}
                routingPrefix="/site-admin"
                telemetryService={NOOP_TELEMETRY_SERVICE}
                afterCreateRoute="/site-admin/after"
                codeHostExternalServices={codeHostExternalServices}
                nonCodeHostExternalServices={nonCodeHostExternalServices}
                autoFocusForm={false}
            />
        )}
    </WebStory>
))

add('Add connection by kind', () => (
    <WebStory initialEntries={['/page?id=github']}>
        {webProps => (
            <AddExternalServicesPage
                {...webProps}
                routingPrefix="/site-admin"
                telemetryService={NOOP_TELEMETRY_SERVICE}
                afterCreateRoute="/site-admin/after"
                codeHostExternalServices={codeHostExternalServices}
                nonCodeHostExternalServices={nonCodeHostExternalServices}
                autoFocusForm={false}
            />
        )}
    </WebStory>
))
