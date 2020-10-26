import MapSearchIcon from 'mdi-react/MapSearchIcon'
import * as React from 'react'
import { HeroPage } from './HeroPage'

export const ErrorNotSupportedPage: React.FunctionComponent = () => (
    <HeroPage
        icon={MapSearchIcon}
        title="Not Found"
        subtitle="This feature is not enabled in the server configuration."
    />
)
