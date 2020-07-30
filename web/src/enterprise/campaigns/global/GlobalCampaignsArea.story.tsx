import { storiesOf } from '@storybook/react'
import { radios, boolean } from '@storybook/addon-knobs'
import React from 'react'
import { GlobalCampaignsArea } from './GlobalCampaignsArea'
import { createMemoryHistory } from 'history'
import webStyles from '../../../SourcegraphWebApp.scss'
import { MemoryRouter } from 'react-router'
import { NOOP_TELEMETRY_SERVICE } from '../../../../../shared/src/telemetry/telemetryService'
import { User } from '../../../../../shared/src/graphql/schema'

const { add } = storiesOf('web/campaigns/GlobalCampaignsArea', module).addDecorator(story => {
    const theme = radios('Theme', { Light: 'light', Dark: 'dark' }, 'light')
    document.body.classList.toggle('theme-light', theme === 'light')
    document.body.classList.toggle('theme-dark', theme === 'dark')
    return (
        <MemoryRouter>
            <style>{webStyles}</style>
            <div className="p-3 container">{story()}</div>
        </MemoryRouter>
    )
})

add('Dotcom', () => (
    <GlobalCampaignsArea
        location={createMemoryHistory().location}
        history={createMemoryHistory()}
        isSourcegraphDotCom={true}
        isLightTheme={true}
        telemetryService={NOOP_TELEMETRY_SERVICE}
        platformContext={undefined as any}
        extensionsController={undefined as any}
        authenticatedUser={boolean('isAuthenticated', false) ? ({ username: 'alice' } as User) : null}
        match={{ isExact: true, path: '/campaigns', url: 'http://test.test/campaigns', params: {} }}
    />
))

add('Private instance', () => (
    <GlobalCampaignsArea
        location={createMemoryHistory().location}
        history={createMemoryHistory()}
        isSourcegraphDotCom={false}
        isLightTheme={true}
        telemetryService={NOOP_TELEMETRY_SERVICE}
        platformContext={undefined as any}
        extensionsController={undefined as any}
        authenticatedUser={{ username: 'alice', siteAdmin: boolean('Site admin', false) } as User}
        match={{ isExact: true, path: '/campaigns', url: 'http://test.test/campaigns', params: {} }}
    />
))
