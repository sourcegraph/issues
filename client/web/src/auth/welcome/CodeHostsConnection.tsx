import { ApolloError } from '@apollo/client'
import React, { useEffect } from 'react'

import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { useSteps } from '@sourcegraph/wildcard/src/components/Steps'

import { UserCodeHosts } from '../../user/settings/codeHosts/UserCodeHosts'

interface CodeHostsConnection extends Omit<UserCodeHosts, 'onDidRemove' | 'onDidError'> {
    refetch: UserCodeHosts['onDidRemove']
    loading: boolean
    error: ApolloError | undefined
    onNavigation?: (called: boolean) => void
}

export const CodeHostsConnection: React.FunctionComponent<CodeHostsConnection> = ({
    user,
    context,
    refetch,
    externalServices,
    loading,
    onNavigation,
}) => {
    const { setComplete, currentIndex } = useSteps()

    useEffect(() => {
        if (Array.isArray(externalServices) && externalServices.length > 0) {
            setComplete(currentIndex, true)
        } else {
            setComplete(currentIndex, false)
        }
    }, [currentIndex, externalServices, setComplete])

    if (loading) {
        return (
            <div className="d-flex justify-content-center">
                <LoadingSpinner className="icon-inline" />
            </div>
        )
    }

    return (
        <>
            <div className="mb-4">
                <h3>Connect with code hosts</h3>
                <p className="text-muted">
                    Connect with providers where your source code is hosted. Then, choose the repositories you’d like to
                    search with Sourcegraph.
                </p>
            </div>
            <UserCodeHosts
                user={user}
                externalServices={externalServices}
                context={context}
                onNavigation={onNavigation}
                onDidError={error => console.warn('<UserCodeHosts .../>', error)}
                onDidRemove={refetch}
            />
        </>
    )
}
