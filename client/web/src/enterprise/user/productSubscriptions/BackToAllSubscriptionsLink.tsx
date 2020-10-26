import ArrowLeftIcon from 'mdi-react/ArrowLeftIcon'
import React from 'react'
import { Link } from 'react-router-dom'
import * as GQL from '../../../../../shared/src/graphql/schema'

export const BackToAllSubscriptionsLink: React.FunctionComponent<{ user: Pick<GQL.IUser, 'settingsURL'> }> = ({
    user,
}) => (
    <Link to={`${user.settingsURL!}/subscriptions`} className="btn btn-link btn-sm mb-3">
        <ArrowLeftIcon className="icon-inline" /> All subscriptions
    </Link>
)
