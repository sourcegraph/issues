// This file contains modifier functions to customize JSON serialization of snapshot tests using enzyme-to-json.
// These are used in share/dev/enzymeSerializer.js

import { Json } from 'enzyme-to-json'
import * as H from 'history'
import { isPlainObject } from 'lodash'
import { Observable, Subject } from 'rxjs'

function maskProps(props: Json['props'], key = '', seenProperties: ReadonlySet<unknown> = new Set()): unknown {
    if (!props) {
        return props
    }
    if (typeof props === 'object' && props !== null) {
        if (seenProperties.has(props)) {
            return '[Cyclic structure]'
        }
    }
    const treeSeenProperties = new Set(seenProperties)
    treeSeenProperties.add(props)
    if (key === 'history') {
        return '[History]'
    }
    if (key === 'location' && typeof props === 'object' && props !== null) {
        return `[Location path=${H.createPath(props as H.Location)}]`
    }
    if (props instanceof Subject) {
        return '[Subject]'
    }
    if (props instanceof Observable) {
        return '[Observable]'
    }

    if (Array.isArray(props)) {
        return props.map(item => maskProps(item, '', treeSeenProperties))
    }
    if (isPlainObject(props)) {
        return Object.fromEntries(
            Object.entries(props).map(([key, value]) => [key, maskProps(value, key, treeSeenProperties)])
        )
    }
    return props
}

export function replaceVerboseObjects(json: Json): Json | string {
    if (/Memo\(\w+Icon\)/.test(json.type)) {
        json.children = []
        return json
    }
    if (json.props) {
        return {
            ...json,
            props: maskProps(json.props) as Json['props'],
        }
    }

    return json
}
