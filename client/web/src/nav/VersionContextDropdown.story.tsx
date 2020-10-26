import * as H from 'history'
import { storiesOf } from '@storybook/react'
import React from 'react'
import { VersionContextDropdown, VersionContextDropdownProps } from './VersionContextDropdown'
import { subtypeOf } from '../../../shared/src/util/types'
import { action } from '@storybook/addon-actions'
import { SearchPatternType } from '../graphql-operations'
import { WebStory } from '../components/WebStory'

const { add } = storiesOf('web/VersionContextDropdown', module).addDecorator(story => (
    <WebStory>{webProps => story()}</WebStory>
))

const setVersionContext = action('setVersionContext')
const history = H.createMemoryHistory({ keyLength: 0 })
const commonProps = subtypeOf<Partial<VersionContextDropdownProps>>()({
    alwaysShow: true,
    history,
    // Make sure the dropdown is not rendered outside the theme-light container
    portal: false,
    caseSensitive: false,
    patternType: SearchPatternType.literal,
    availableVersionContexts: [
        { name: 'test 1', description: 'test 1', revisions: [{ rev: 'test', repo: 'github.com/test/test' }] },
        { name: 'test 2', description: 'test 2', revisions: [{ rev: 'test', repo: 'github.com/test/test' }] },
        { name: 'test 3', description: 'test 3', revisions: [{ rev: 'test', repo: 'github.com/test/test' }] },
    ],
    navbarSearchQuery: 'test',
    setVersionContext,
})

add('No context selected', () => <VersionContextDropdown {...commonProps} versionContext={undefined} />, {
    design: {
        type: 'figma',
        url: 'https://www.figma.com/file/Sax8ctk8GhvWd0vrzkHSDK/Version-Contexts?node-id=97%3A175',
    },
})
add('Context selected', () => <VersionContextDropdown {...commonProps} versionContext="test 1" />, {
    design: {
        type: 'figma',
        url: 'https://www.figma.com/file/Sax8ctk8GhvWd0vrzkHSDK/Version-Contexts?node-id=95%3A22516',
    },
})
add('Selected context appears at the top of the list', () => (
    <VersionContextDropdown {...commonProps} versionContext="test 3" />
))
add('Not first child', () => (
    <>
        <div />
        <VersionContextDropdown {...commonProps} versionContext="test 4" />
    </>
))
