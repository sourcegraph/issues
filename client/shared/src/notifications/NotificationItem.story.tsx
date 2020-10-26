import { NotificationType } from '@sourcegraph/extension-api-classes'
import { action } from '@storybook/addon-actions'
import { number, select, text } from '@storybook/addon-knobs'
import { storiesOf } from '@storybook/react'
import React from 'react'
import { of } from 'rxjs'
import { NotificationType as NotificationTypeType } from 'sourcegraph'
import { NotificationItem } from './NotificationItem'
import notificationItemStyles from './NotificationItem.scss'
import webStyles from '../../../web/src/SourcegraphWebApp.scss'

const notificationClassNames = {
    [NotificationType.Log]: 'alert alert-secondary',
    [NotificationType.Success]: 'alert alert-success',
    [NotificationType.Info]: 'alert alert-info',
    [NotificationType.Warning]: 'alert alert-warning',
    [NotificationType.Error]: 'alert alert-danger',
}

const onDismiss = action('onDismiss')

const { add } = storiesOf('shared/NotificationItem', module).addDecorator(story => (
    <>
        <style>{webStyles}</style>
        <style>{notificationItemStyles}</style>
        <div className="theme-light" style={{ maxWidth: '20rem', margin: '2rem' }}>
            {story()}
        </div>
    </>
))

add('Without Progress', () => {
    const message = text('Message', 'My *custom* message')
    const type = select<NotificationTypeType>(
        'Type',
        NotificationType as Record<keyof typeof NotificationType, NotificationTypeType>,
        NotificationType.Error
    )
    const source = text('Source', 'some source')
    return (
        <NotificationItem
            notification={{ message, type, source }}
            notificationClassNames={notificationClassNames}
            onDismiss={onDismiss}
        />
    )
})

add('With progress', () => {
    const message = text('Message', 'My *custom* message')
    const type = select<NotificationTypeType>(
        'Type',
        NotificationType as Record<keyof typeof NotificationType, NotificationTypeType>,
        NotificationType.Info
    )
    const source = text('Source', 'some source')
    const progressMessage = text('Progress message', 'My *custom* progress message')
    const progressPercentage = number('Progress % (0-100)', 50)
    return (
        <NotificationItem
            notification={{
                message,
                type,
                source,
                progress: of({
                    message: progressMessage,
                    percentage: progressPercentage,
                }),
            }}
            notificationClassNames={notificationClassNames}
            onDismiss={onDismiss}
        />
    )
})
