import * as React from 'react'

/**
 * Converts a synthetic React event to a persisted, native Event object.
 *
 * @param event The synthetic React event object
 */
export const toNativeEvent = <E extends React.SyntheticEvent<T>, T>(event: E): E['nativeEvent'] => {
    event.persist()
    return event.nativeEvent
}
