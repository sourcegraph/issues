import { storiesOf } from '@storybook/react'
import React from 'react'
import { NEVER } from 'rxjs'

import { EMPTY_SETTINGS_CASCADE } from '@sourcegraph/shared/src/settings/settings'
import { NOOP_TELEMETRY_SERVICE } from '@sourcegraph/shared/src/telemetry/telemetryService'

import { WebStory } from '../../components/WebStory'

import { SearchNotebook } from './SearchNotebook'

import { BlockInitializer } from '.'

const { add } = storiesOf('web/search/notebook/SearchNotebook', module).addDecorator(story => (
    <div className="p-3 container">{story()}</div>
))

const blocks: BlockInitializer[] = [
    { type: 'md', input: '# Markdown' },
    { type: 'query', input: 'Query' },
    { type: 'md', input: '# Markdown 1' },
]

add('default', () => (
    <WebStory>
        {props => (
            <SearchNotebook
                {...props}
                isMacPlatform={true}
                globbing={true}
                telemetryService={NOOP_TELEMETRY_SERVICE}
                streamSearch={() => NEVER}
                fetchHighlightedFileLineRanges={() => NEVER}
                onSerializeBlocks={() => {}}
                blocks={blocks}
                settingsCascade={EMPTY_SETTINGS_CASCADE}
            />
        )}
    </WebStory>
))

add('default read-only', () => (
    <WebStory>
        {props => (
            <SearchNotebook
                {...props}
                isReadOnly={true}
                isMacPlatform={true}
                globbing={true}
                telemetryService={NOOP_TELEMETRY_SERVICE}
                streamSearch={() => NEVER}
                fetchHighlightedFileLineRanges={() => NEVER}
                onSerializeBlocks={() => {}}
                blocks={blocks}
                settingsCascade={EMPTY_SETTINGS_CASCADE}
            />
        )}
    </WebStory>
))
