import * as H from 'history'
import { storiesOf } from '@storybook/react'
import React from 'react'
import webMainStyles from '../SourcegraphWebApp.scss'
import bootstrapStyles from 'bootstrap/scss/bootstrap.scss'
import { StatusMessagesNavItem } from './StatusMessagesNavItem'
import { of } from 'rxjs'

const { add } = storiesOf('web/StatusMessagesNavItem', module).addDecorator(story => (
    <>
        <style>{bootstrapStyles}</style>
        <style>{webMainStyles}</style>
        <div className="theme-light">{story()}</div>
    </>
))

const history = H.createMemoryHistory({ keyLength: 0 })

add('StatusMessagesNavItem', () => (
    <StatusMessagesNavItem
        isSiteAdmin={true}
        history={history}
        fetchMessages={() => of([{ __typename: 'CloningProgress', message: 'CloningProgress' }])}
    />
))
