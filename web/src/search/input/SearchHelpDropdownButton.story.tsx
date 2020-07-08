import { storiesOf } from '@storybook/react'
import React from 'react'
import css from 'tagged-template-noop'
import webStyles from '../../SourcegraphWebApp.scss'
import bootstrapStyles from 'bootstrap/scss/bootstrap.scss'
import { SearchHelpDropdownButton } from './SearchHelpDropdownButton'

const cssVars = css`
    :root {
        --body-color: #2b3750;
    }
`

const { add } = storiesOf('web/SearchHelpDropdown', module).addDecorator(story => (
    <>
        <style>{bootstrapStyles}</style>
        <style>{webStyles}</style>
        <style>{cssVars}</style>
        <div className="theme-light">{story()}</div>
    </>
))

add('SearchHelpDropdown', () => <SearchHelpDropdownButton />)
