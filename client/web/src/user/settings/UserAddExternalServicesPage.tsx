import React from 'react'
import {
    AddExternalServicesPageProps,
    AddExternalServicesPage,
} from '../../components/externalServices/AddExternalServicesPage'
import { codeHostExternalServices } from '../../components/externalServices/externalServices'

export interface UserAddExternalServicesPageProps
    extends Omit<AddExternalServicesPageProps, 'codeHostExternalServices' | 'nonCodeHostExternalServices'> {}

export const UserAddExternalServicesPage: React.FunctionComponent<UserAddExternalServicesPageProps> = props => (
    <AddExternalServicesPage
        {...props}
        codeHostExternalServices={{
            github: codeHostExternalServices.github,
            gitlabcom: codeHostExternalServices.gitlabcom,
            bitbucket: codeHostExternalServices.bitbucket,
        }}
        nonCodeHostExternalServices={{}}
    />
)
