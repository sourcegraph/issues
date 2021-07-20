// adapted from https://github.com/radix-ui/primitives/blob/2f139a832ba0cdfd445c937ebf63c2e79e0ef7ed/packages/react/polymorphic/src/polymorphic.ts
// Would have liked to use it directly instead of copying but they are
// (rightfully) treating it as an internal utility, so copy/paste it is to
// prevent any needless churn if they make breaking changes.

import type * as React from 'react'

type Merge<P1 = {}, P2 = {}> = Omit<P1, keyof P2> & P2

type NarrowIntrinsic<E> = E extends keyof JSX.IntrinsicElements ? E : never

type ForwardReferenceExoticComponent<E, OwnProps> = React.ForwardRefExoticComponent<
    Merge<E extends React.ElementType ? React.ComponentPropsWithRef<E> : never, OwnProps & { as?: E }>
>

export interface ForwardReferenceComponent<
    IntrinsicElementString,
    OwnProps = {}
    /*
     * Extends original type to ensure built in React types play nice with
     * polymorphic components still e.g. `React.ElementRef` etc.
     */
> extends ForwardReferenceExoticComponent<IntrinsicElementString, OwnProps> {
    /*
     * When `as` prop is passed, use this overload. Merges original own props
     * (without DOM props) and the inferred props from `as` element with the own
     * props taking precendence.
     *
     * We explicitly avoid `React.ElementType` and manually narrow the prop types
     * so that events are typed when using JSX.IntrinsicElements.
     */
    <As extends keyof JSX.IntrinsicElements | React.ComponentType<any> = NarrowIntrinsic<IntrinsicElementString>>(
        props: As extends keyof JSX.IntrinsicElements
            ? Merge<JSX.IntrinsicElements[As], OwnProps & { as: As }>
            : As extends React.ComponentType<infer P>
            ? Merge<P, OwnProps & { as: As }>
            : never
    ): React.ReactElement | null
}
