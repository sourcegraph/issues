import { createMemoryHistory } from 'history'
import * as sinon from 'sinon'
import assert from 'assert'
import { createLinkClickHandler } from './linkClickHandler'
import ReactDOM from 'react-dom'
import React from 'react'

describe('createLinkClickHandler', () => {
    it('handles clicks on links that stay inside the app', () => {
        jsdom.reconfigure({ url: 'https://sourcegraph.test/some/where' })

        const history = createMemoryHistory({ initialEntries: [] })
        expect(history).toHaveLength(0)

        const root = document.createElement('div')
        document.body.append(root)
        ReactDOM.render(
            <div onClick={createLinkClickHandler(history)}>
                <a href="https://sourcegraph.test/else/where">Test</a>
            </div>,
            root
        )

        const anchor = root.querySelector('a')
        assert(anchor)

        const spy = sinon.spy((_event: MouseEvent) => undefined)
        window.addEventListener('click', spy)

        anchor.click()

        sinon.assert.calledOnce(spy)
        expect(spy.args[0][0].defaultPrevented).toBe(true)

        expect(history).toHaveLength(1)
        expect(history.entries[0].pathname).toBe('/else/where')
    })

    it('ignores clicks on links that go outside the app', () => {
        jsdom.reconfigure({ url: 'https://sourcegraph.test/some/where' })
        const history = createMemoryHistory({ initialEntries: [] })
        expect(history).toHaveLength(0)

        const root = document.createElement('div')
        document.body.append(root)
        ReactDOM.render(
            <div onClick={createLinkClickHandler(history)}>
                <a href="https://github.com/some/where">Test</a>
            </div>,
            root
        )

        const anchor = root.querySelector('a')
        assert(anchor)

        const spy = sinon.spy((_event: MouseEvent) => undefined)
        window.addEventListener('click', spy)

        anchor.click()

        sinon.assert.calledOnce(spy)
        expect(spy.args[0][0].defaultPrevented).toBe(false)
        expect(history).toHaveLength(0)
    })
})
