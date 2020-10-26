import { storiesOf } from '@storybook/react'
import React from 'react'
import { ExternalServicesPage } from './ExternalServicesPage'
import { NOOP_TELEMETRY_SERVICE } from '../../../../shared/src/telemetry/telemetryService'
import { queryExternalServices as _queryExternalServices } from './backend'
import { of } from 'rxjs'
import { ExternalServiceKind } from '../../graphql-operations'
import { WebStory } from '../WebStory'

const { add } = storiesOf('web/External services/ExternalServicesPage', module).addDecorator(story => (
    <div className="p-3 container">{story()}</div>
))

const queryExternalServices: typeof _queryExternalServices = () =>
    of({
        totalCount: 1,
        pageInfo: {
            endCursor: null,
            hasNextPage: false,
        },
        nodes: [
            {
                id: 'service1',
                kind: ExternalServiceKind.GITHUB,
                displayName: 'GitHub.com',
                config: '{"githubconfig":true}',
            },
        ],
    })

add('List of external services', () => (
    <WebStory>
        {webProps => (
            <ExternalServicesPage
                {...webProps}
                routingPrefix="/site-admin"
                afterDeleteRoute="/site-admin/after"
                telemetryService={NOOP_TELEMETRY_SERVICE}
                authenticatedUser={{ id: '123' }}
                queryExternalServices={queryExternalServices}
            />
        )}
    </WebStory>
))
