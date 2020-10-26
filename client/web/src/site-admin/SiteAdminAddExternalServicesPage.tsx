import React from 'react'
import {
    AddExternalServicesPage,
    AddExternalServicesPageProps,
} from '../components/externalServices/AddExternalServicesPage'
import { codeHostExternalServices, nonCodeHostExternalServices } from '../components/externalServices/externalServices'

export interface SiteAdminAddExternalServicesPageProps
    extends Omit<
        AddExternalServicesPageProps,
        'routingPrefix' | 'afterCreateRoute' | 'codeHostExternalServices' | 'nonCodeHostExternalServices'
    > {}

export const SiteAdminAddExternalServicesPage: React.FunctionComponent<SiteAdminAddExternalServicesPageProps> = props => (
    <AddExternalServicesPage
        {...props}
        routingPrefix="/site-admin"
        afterCreateRoute="/site-admin/repositories?repositoriesUpdated"
        codeHostExternalServices={codeHostExternalServices}
        nonCodeHostExternalServices={nonCodeHostExternalServices}
    />
)
