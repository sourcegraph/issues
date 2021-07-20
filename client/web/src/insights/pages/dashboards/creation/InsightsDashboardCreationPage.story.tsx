import { storiesOf } from '@storybook/react'
import React from 'react'

import { NOOP_TELEMETRY_SERVICE } from '@sourcegraph/shared/src/telemetry/telemetryService'

import { WebStory } from '../../../../components/WebStory'
import { authUser } from '../../../../search/panels/utils'
import { InsightsApiContext } from '../../../core/backend/api-provider'
import { createMockInsightAPI } from '../../../core/backend/insights-api'
import { SETTINGS_CASCADE } from '../../../mocks/settings-cascade'

import { InsightsDashboardCreationPage } from './InsightsDashboardCreationPage'

const { add } = storiesOf('web/insights/InsightsDashboardCreationPage', module)
    .addDecorator(story => <WebStory>{() => story()}</WebStory>)
    .addParameters({
        chromatic: {
            viewports: [576, 1440],
        },
    })

const PLATFORM_CONTEXT = {
    // eslint-disable-next-line @typescript-eslint/require-await
    updateSettings: async (...args: any[]) => {
        console.log('PLATFORM CONTEXT update settings with', { ...args })
    },
}

const mockAPI = createMockInsightAPI({})

add('Page', () => (
    <InsightsApiContext.Provider value={mockAPI}>
        <InsightsDashboardCreationPage
            telemetryService={NOOP_TELEMETRY_SERVICE}
            platformContext={PLATFORM_CONTEXT}
            settingsCascade={SETTINGS_CASCADE}
            authenticatedUser={authUser}
        />
    </InsightsApiContext.Provider>
))
