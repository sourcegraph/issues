import React, { useCallback } from 'react'
import { SourcegraphIconButton } from '../../components/SourcegraphIconButton'
import { interval, Observable } from 'rxjs'
import { switchMap, filter, take, tap } from 'rxjs/operators'
import { useEventObservable } from '../../../../../shared/src/util/useObservable'
import { getPlatformName } from '../../util/context'

export const SignInButton: React.FunctionComponent<{
    className?: string
    iconClassName?: string
    sourcegraphURL: string
    /**
     * Gets called when the user closed the sign in tab.
     * Does not guarantee the sign in was sucessful.
     */
    onSignInClose?: () => void
}> = ({ className, iconClassName, sourcegraphURL, onSignInClose }) => {
    const signInUrl = new URL(`/sign-in?close=true&utm_source=${getPlatformName()}`, sourcegraphURL).href

    const [nextSignInClick] = useEventObservable(
        useCallback(
            (signInClicks: Observable<React.MouseEvent>) =>
                signInClicks.pipe(
                    switchMap(event => {
                        const tab = window.open(signInUrl, '_blank')
                        if (!tab) {
                            return []
                        }
                        event.preventDefault()
                        return interval(300).pipe(
                            filter(() => tab.closed),
                            take(1)
                        )
                    }),
                    tap(onSignInClose)
                ),
            [onSignInClose, signInUrl]
        )
    )

    return (
        <SourcegraphIconButton
            href={signInUrl}
            label="Sign in to Sourcegraph"
            title="Sign into Sourcegraph to get hover tooltips, go to definition and more"
            ariaLabel="Sign into Sourcegraph to get hover tooltips, go to definition and more"
            className={className}
            iconClassName={iconClassName}
            onClick={nextSignInClick}
        />
    )
}
