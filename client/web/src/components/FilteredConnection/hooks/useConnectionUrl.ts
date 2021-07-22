import { useEffect } from 'react'
import { useHistory, useLocation } from 'react-router'

import { getUrlQuery, GetUrlQueryParameters } from '../utils'

interface UseConnectionURLParameters extends Pick<GetUrlQueryParameters, 'first' | 'visibleResultCount'> {
    enabled?: boolean
}

/**
 * This hook replicates how FilteredConnection updates the URL when key variables change.
 * We use this to ensure the URL is kept in sync with the current connection state.
 * This is to allow users to build complex requests that can still be shared with others.
 * It is closely coupled to useConnection, which also derives initial state from the URL.
 */
export const useConnectionUrl = ({ enabled, first, visibleResultCount }: UseConnectionURLParameters): void => {
    const location = useLocation()
    const history = useHistory()
    const searchFragment = getUrlQuery({
        first,
        visibleResultCount,
        search: location.search,
    })

    useEffect(() => {
        if (enabled && searchFragment && location.search !== `?${searchFragment}`) {
            history.replace({
                search: searchFragment,
                hash: location.hash,
            })
        }
    }, [enabled, history, location.hash, location.search, searchFragment])
}
