import React from 'react'
import { mount } from 'enzyme'
import { PanelContainer } from './PanelContainer'

describe('PanelContainer', () => {
    const defaultProps = {
        title: 'Test Panel',
        state: 'loading',
        loadingContent: <div>Loading</div>,
        populatedContent: <div>Content</div>,
        emptyContent: <div>Empty</div>,
    }

    test('loading state', () => {
        expect(mount(<PanelContainer {...defaultProps} state="loading" />)).toMatchSnapshot()
    })

    test('empty state', () => {
        expect(mount(<PanelContainer {...defaultProps} state="empty" />)).toMatchSnapshot()
    })

    test('content state', () => {
        expect(mount(<PanelContainer {...defaultProps} state="populated" />)).toMatchSnapshot()
    })

    test('with action buttons', () => {
        const actionButtons = <button type="button">Button</button>
        expect(
            mount(<PanelContainer {...defaultProps} state="populated" actionButtons={actionButtons} />)
        ).toMatchSnapshot()
    })
})
