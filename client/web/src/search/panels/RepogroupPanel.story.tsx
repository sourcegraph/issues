import React from 'react'
import { _fetchRecentSearches } from './utils'
import { NOOP_TELEMETRY_SERVICE } from '../../../../shared/src/telemetry/telemetryService'
import { RepogroupPanel } from './RepogroupPanel'
import { storiesOf } from '@storybook/react'
import { WebStory } from '../../components/WebStory'

const { add } = storiesOf('web/search/panels/RepogroupPanel', module)
    .addParameters({
        design: {
            type: 'figma',
            url:
                'https://www.figma.com/file/zCGglxWBFm8Fv5DwdcHdAQ/Repository-group-home-page-panel-14393?node-id=1%3A159',
        },
        chromatic: { viewports: [800] },
    })
    .addDecorator(story => <div style={{ width: '800px' }}>{story()}</div>)

const props = {
    authenticatedUser: null,
    telemetryService: NOOP_TELEMETRY_SERVICE,
}

add('Populated', () => <WebStory>{() => <RepogroupPanel {...props} />}</WebStory>)
