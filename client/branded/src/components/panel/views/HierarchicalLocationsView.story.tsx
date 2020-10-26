import { storiesOf } from '@storybook/react'
import React from 'react'
import { HierarchicalLocationsView, HierarchicalLocationsViewProps } from './HierarchicalLocationsView'
import webStyles from '../../../../../web/src/main.scss'
import { BrandedStory } from '../../BrandedStory'
import * as H from 'history'
import { Location } from '@sourcegraph/extension-api-types'
import { of } from 'rxjs'
import { createContextService } from '../../../../../shared/src/api/client/context/contextService'
import {
    ContributionsEntry,
    ContributionUnsubscribable,
} from '../../../../../shared/src/api/client/services/contribution'
import { noop } from 'lodash'

const { add } = storiesOf('branded/HierarchicalLocationsView', module).addDecorator(story => (
    <BrandedStory styles={webStyles}>{() => <div className="p-5">{story()}</div>}</BrandedStory>
))

const LOCATIONS: Location[] = [
    {
        uri: 'git://github.com/foo/bar#file1.txt',
        range: {
            start: {
                line: 1,
                character: 0,
            },
            end: {
                line: 1,
                character: 10,
            },
        },
    },
    {
        uri: 'git://github.com/foo/bar#file2.txt',
        range: {
            start: {
                line: 2,
                character: 0,
            },
            end: {
                line: 2,
                character: 10,
            },
        },
    },
    {
        uri: 'git://github.com/baz/qux#file3.txt',
        range: {
            start: {
                line: 3,
                character: 0,
            },
            end: {
                line: 3,
                character: 10,
            },
        },
    },
    {
        uri: 'git://github.com/baz/qux#file4.txt',
        range: {
            start: {
                line: 4,
                character: 0,
            },
            end: {
                line: 4,
                character: 10,
            },
        },
    },
    {
        uri: 'git://github.com/baz/qux#file4.txt',
        range: {
            start: {
                line: 5,
                character: 0,
            },
            end: {
                line: 5,
                character: 10,
            },
        },
    },
]

const PROPS: HierarchicalLocationsViewProps = {
    extensionsController: {
        services: {
            context: createContextService({ clientApplication: 'other' }),
            contribution: {
                registerContributions: (entry: ContributionsEntry): ContributionUnsubscribable => ({
                    entry,
                    unsubscribe: noop,
                }),
            },
        },
    },
    settingsCascade: { subjects: null, final: null },
    location: H.createLocation('/'),
    locations: of({ isLoading: false, result: LOCATIONS }),
    defaultGroup: 'git://github.com/foo/bar',
    isLightTheme: true,
    fetchHighlightedFileLines: () => of(['line1\n', 'line2\n', 'line3\n', 'line4']),
    versionContext: undefined,
}

add('Single repo', () => (
    <HierarchicalLocationsView
        {...PROPS}
        locations={of({ isLoading: false, result: LOCATIONS.filter(({ uri }) => uri.includes('github.com/foo/bar')) })}
    />
))

add('Grouped by repo', () => <HierarchicalLocationsView {...PROPS} />)

add('Grouped by repo and file', () => (
    <HierarchicalLocationsView
        {...PROPS}
        settingsCascade={{
            subjects: null,
            final: {
                'panel.locations.groupByFile': true,
            },
        }}
    />
))
